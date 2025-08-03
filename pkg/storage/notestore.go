package storage

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"gote/pkg/crypto"
	"gote/pkg/models"
	"gote/pkg/utils"
)

// NoteStore manages note storage and file system operations
type NoteStore struct {
	dataDir          string
	notes            map[string]*models.Note
	mutex            sync.RWMutex
	watcher          *fsnotify.Watcher
	key              []byte
	lastSync         time.Time
	fileModTimes     map[string]time.Time
	pendingDeletions map[string]bool // Track app-initiated deletions
}

// NewNoteStore creates a new note store instance
func NewNoteStore(dataDir string) *NoteStore {
	store := &NoteStore{
		dataDir:          dataDir,
		notes:            make(map[string]*models.Note),
		fileModTimes:     make(map[string]time.Time),
		pendingDeletions: make(map[string]bool),
	}

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatal("Failed to create data directory:", err)
	}

	// Initialize file system watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("Warning: Could not create file watcher: %v", err)
	} else {
		store.watcher = watcher

		// Add data directory to watcher
		if err := watcher.Add(dataDir); err != nil {
			log.Printf("Warning: Could not watch data directory: %v", err)
		}
	}

	return store
}

// GetDataDir returns the data directory path
func (s *NoteStore) GetDataDir() string {
	return s.dataDir
}

// LoadNotes loads notes from disk with the provided encryption key
func (s *NoteStore) LoadNotes(key []byte) error {
	s.mutex.Lock()
	s.key = key
	s.mutex.Unlock()

	// Start file watching
	s.startWatching()

	// Load notes from disk
	return s.syncFromDisk()
}

// startWatching starts the file system watcher goroutine
func (s *NoteStore) startWatching() {
	if s.watcher == nil {
		return
	}

	go func() {
		for {
			select {
			case event, ok := <-s.watcher.Events:
				if !ok {
					return
				}

				// Only process .json files with valid short hash names
				if !strings.HasSuffix(event.Name, ".json") {
					continue
				}

				filename := filepath.Base(event.Name)
				if !utils.IsValidShortHashFilename(filename) {
					log.Printf("Ignoring file with invalid name pattern: %s", filename)
					continue
				}

				log.Printf("File event: %s %s", event.Op, event.Name)

				switch {
				case event.Op&fsnotify.Create == fsnotify.Create:
					s.handleFileCreate(event.Name)
				case event.Op&fsnotify.Write == fsnotify.Write:
					s.handleFileWrite(event.Name)
				case event.Op&fsnotify.Remove == fsnotify.Remove:
					s.handleFileRemove(event.Name)
				case event.Op&fsnotify.Rename == fsnotify.Rename:
					s.handleFileRemove(event.Name)
				}

			case err, ok := <-s.watcher.Errors:
				if !ok {
					return
				}
				log.Printf("Watcher error: %v", err)
			}
		}
	}()
}

// handleFileCreate handles new file creation
func (s *NoteStore) handleFileCreate(filePath string) {
	s.handleFileWrite(filePath)
}

// handleFileWrite handles file modifications
func (s *NoteStore) handleFileWrite(filePath string) {
	if s.key == nil {
		return // Not authenticated yet
	}

	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		log.Printf("Error getting file info for %s: %v", filePath, err)
		return
	}

	// Check if this is a change we need to process
	s.mutex.Lock()
	lastModTime, exists := s.fileModTimes[filePath]
	currentModTime := fileInfo.ModTime()

	// If we already have this modification time, skip (probably our own write)
	if exists && !currentModTime.After(lastModTime) {
		s.mutex.Unlock()
		return
	}

	s.fileModTimes[filePath] = currentModTime
	s.mutex.Unlock()

	// Load the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("Error reading changed file %s: %v", filePath, err)
		return
	}

	var encryptedNote models.EncryptedNote
	if err := json.Unmarshal(data, &encryptedNote); err != nil {
		log.Printf("Error unmarshalling changed file %s: %v", filePath, err)
		return
	}

	// Decrypt the note content
	decryptedContent, err := crypto.Decrypt(encryptedNote.EncryptedData, s.key)
	if err != nil {
		log.Printf("Error decrypting changed file %s: %v", filePath, err)
		return
	}

	note := &models.Note{
		ID:        encryptedNote.ID,
		Content:   decryptedContent,
		CreatedAt: encryptedNote.CreatedAt,
		UpdatedAt: encryptedNote.UpdatedAt,
	}

	s.mutex.Lock()
	existingNote, exists := s.notes[note.ID]

	// Only update if the external file is newer than what we have in memory
	if !exists || note.UpdatedAt.After(existingNote.UpdatedAt) {
		s.notes[note.ID] = note
		log.Printf("Updated note %s from external file change", note.ID)
	} else {
		log.Printf("Skipped updating note %s - in-memory version is newer", note.ID)
	}
	s.mutex.Unlock()
}

// handleFileRemove handles file deletion
func (s *NoteStore) handleFileRemove(filePath string) {
	// Extract note ID from filename
	filename := filepath.Base(filePath)
	if !strings.HasSuffix(filename, ".json") {
		return
	}

	// Only process files with valid short hash names
	if !utils.IsValidShortHashFilename(filename) {
		return
	}

	noteID := strings.TrimSuffix(filename, ".json")

	s.mutex.Lock()
	// Check if this was an app-initiated deletion
	wasAppDeleted := s.pendingDeletions[noteID]
	delete(s.pendingDeletions, noteID) // Clean up the tracking
	delete(s.notes, noteID)
	delete(s.fileModTimes, filePath)
	s.mutex.Unlock()

	if wasAppDeleted {
		log.Printf("Note %s deleted successfully", noteID)
	} else {
		log.Printf("Removed note %s due to external file deletion", noteID)
	}
}

// syncFromDisk performs a full sync from disk
func (s *NoteStore) syncFromDisk() error {
	if s.key == nil {
		return fmt.Errorf("not authenticated")
	}

	files, err := filepath.Glob(filepath.Join(s.dataDir, "*.json"))
	if err != nil {
		return fmt.Errorf("error reading data directory: %v", err)
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Track which notes exist on disk
	diskNotes := make(map[string]bool)

	for _, file := range files {
		// Only process files with valid short hash names
		filename := filepath.Base(file)
		if !utils.IsValidShortHashFilename(filename) {
			log.Printf("Ignoring file with invalid name pattern during sync: %s", filename)
			continue
		}

		fileInfo, err := os.Stat(file)
		if err != nil {
			log.Printf("Error getting file info for %s: %v", file, err)
			continue
		}

		data, err := os.ReadFile(file)
		if err != nil {
			log.Printf("Error reading file %s: %v", file, err)
			continue
		}

		var encryptedNote models.EncryptedNote
		if err := json.Unmarshal(data, &encryptedNote); err != nil {
			log.Printf("Error unmarshalling encrypted note from %s: %v", file, err)
			continue
		}

		// Decrypt the note content
		decryptedContent, err := crypto.Decrypt(encryptedNote.EncryptedData, s.key)
		if err != nil {
			log.Printf("Error decrypting note from %s: %v", file, err)
			continue
		}

		note := &models.Note{
			ID:        encryptedNote.ID,
			Content:   decryptedContent,
			CreatedAt: encryptedNote.CreatedAt,
			UpdatedAt: encryptedNote.UpdatedAt,
		}

		diskNotes[note.ID] = true
		s.fileModTimes[file] = fileInfo.ModTime()

		// Update note if it's newer or doesn't exist in memory
		existingNote, exists := s.notes[note.ID]
		if !exists || note.UpdatedAt.After(existingNote.UpdatedAt) {
			s.notes[note.ID] = note
		}
	}

	// Remove notes that no longer exist on disk
	for noteID := range s.notes {
		if !diskNotes[noteID] {
			delete(s.notes, noteID)
		}
	}

	s.lastSync = time.Now()
	return nil
}

// saveNote saves a note to disk
func (s *NoteStore) saveNote(note *models.Note, key []byte) error {
	// Encrypt the note content
	encryptedContent, err := crypto.Encrypt(note.Content, key)
	if err != nil {
		return err
	}

	encryptedNote := models.EncryptedNote{
		ID:            note.ID,
		EncryptedData: encryptedContent,
		CreatedAt:     note.CreatedAt,
		UpdatedAt:     note.UpdatedAt,
	}

	filename := filepath.Join(s.dataDir, fmt.Sprintf("%s.json", note.ID))
	data, err := json.MarshalIndent(encryptedNote, "", "  ")
	if err != nil {
		return err
	}

	// Write the file
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return err
	}

	// Update our modification time tracking to prevent processing our own write
	if fileInfo, err := os.Stat(filename); err == nil {
		s.mutex.Lock()
		s.fileModTimes[filename] = fileInfo.ModTime()
		s.mutex.Unlock()
	}

	return nil
}

// SaveNoteDirect saves a note to disk, bypassing in-memory update (for password change)
func (s *NoteStore) SaveNoteDirect(note *models.Note, key []byte) error {
	// Encrypt the note content
	encryptedContent, err := crypto.Encrypt(note.Content, key)
	if err != nil {
		return err
	}

	encryptedNote := models.EncryptedNote{
		ID:            note.ID,
		EncryptedData: encryptedContent,
		CreatedAt:     note.CreatedAt,
		UpdatedAt:     note.UpdatedAt,
	}

	filename := filepath.Join(s.dataDir, note.ID+".json")
	data, err := json.MarshalIndent(encryptedNote, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

// deleteNote removes a note from disk
func (s *NoteStore) deleteNote(id string) error {
	filename := filepath.Join(s.dataDir, fmt.Sprintf("%s.json", id))

	s.mutex.Lock()
	// Mark this deletion as app-initiated
	s.pendingDeletions[id] = true
	delete(s.notes, id)
	delete(s.fileModTimes, filename)
	s.mutex.Unlock()

	return os.Remove(filename)
}

// CreateNote creates a new note
func (s *NoteStore) CreateNote(content string, key []byte) (*models.Note, error) {
	note := &models.Note{
		ID:        utils.GenerateShortUUID(),
		Content:   content,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	s.mutex.Lock()
	s.notes[note.ID] = note
	s.mutex.Unlock()

	if err := s.saveNote(note, key); err != nil {
		s.mutex.Lock()
		delete(s.notes, note.ID)
		s.mutex.Unlock()
		return nil, err
	}

	return note, nil
}

// UpdateNote updates an existing note
func (s *NoteStore) UpdateNote(id string, content string, key []byte) (*models.Note, error) {
	s.mutex.Lock()
	note, exists := s.notes[id]
	if !exists {
		s.mutex.Unlock()
		return nil, fmt.Errorf("note not found")
	}

	note.Content = content
	note.UpdatedAt = time.Now()
	s.mutex.Unlock()

	if err := s.saveNote(note, key); err != nil {
		return nil, err
	}

	return note, nil
}

// GetNote retrieves a note by ID
func (s *NoteStore) GetNote(id string) (*models.Note, error) {
	s.mutex.RLock()
	note, exists := s.notes[id]
	s.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("note not found")
	}
	return note, nil
}

// GetAllNotes returns all notes sorted by update time
func (s *NoteStore) GetAllNotes() []*models.Note {
	s.mutex.RLock()
	notes := make([]*models.Note, 0, len(s.notes))
	for _, note := range s.notes {
		notes = append(notes, note)
	}
	s.mutex.RUnlock()

	// Sort by updated time, newest first
	sort.Slice(notes, func(i, j int) bool {
		return notes[i].UpdatedAt.After(notes[j].UpdatedAt)
	})

	return notes
}

// SearchNotes searches for notes containing the query string
func (s *NoteStore) SearchNotes(query string) []*models.Note {
	var results []*models.Note
	query = strings.ToLower(query)

	s.mutex.RLock()
	for _, note := range s.notes {
		if strings.Contains(strings.ToLower(note.Content), query) {
			results = append(results, note)
		}
	}
	s.mutex.RUnlock()

	// Sort by updated time, newest first
	sort.Slice(results, func(i, j int) bool {
		return results[i].UpdatedAt.After(results[j].UpdatedAt)
	})

	return results
}

// DeleteNote deletes a note by ID
func (s *NoteStore) DeleteNote(id string) error {
	s.mutex.Lock()
	_, exists := s.notes[id]
	s.mutex.Unlock()

	if !exists {
		return fmt.Errorf("note not found")
	}
	return s.deleteNote(id)
}

// Close cleans up the file watcher
func (s *NoteStore) Close() error {
	if s.watcher != nil {
		return s.watcher.Close()
	}
	return nil
}

// RefreshFromDisk forces a full refresh from disk
func (s *NoteStore) RefreshFromDisk() error {
	return s.syncFromDisk()
}

// MoveNoteToCorrupted moves a note file to the corrupted folder
func (s *NoteStore) MoveNoteToCorrupted(noteID string) error {
	corruptedDir := filepath.Join(s.dataDir, "corrupted")
	if err := os.MkdirAll(corruptedDir, 0755); err != nil {
		return err
	}
	oldPath := filepath.Join(s.dataDir, noteID+".json")
	newPath := filepath.Join(corruptedDir, noteID+".json")
	if err := os.Rename(oldPath, newPath); err != nil {
		return err
	}
	// Remove from in-memory store
	s.mutex.Lock()
	delete(s.notes, noteID)
	delete(s.fileModTimes, oldPath)
	s.mutex.Unlock()
	return nil
}

// ClearAllNotes removes all notes from storage and file system
func (s *NoteStore) ClearAllNotes() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Remove all files from data directory
	files, err := filepath.Glob(filepath.Join(s.dataDir, "*.json"))
	if err != nil {
		return fmt.Errorf("failed to list note files: %v", err)
	}

	for _, file := range files {
		if err := os.Remove(file); err != nil {
			log.Printf("Failed to remove file %s: %v", file, err)
		}
	}

	// Clear in-memory storage
	s.notes = make(map[string]*models.Note)
	s.fileModTimes = make(map[string]time.Time)

	return nil
}
