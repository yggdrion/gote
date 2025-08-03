package services

import (
	"encoding/json"
	"fmt"
	"gote/pkg/auth"
	"gote/pkg/crypto"
	"gote/pkg/models"
	"gote/pkg/storage"
	"os"
	"path/filepath"
)

// NoteService handles note business logic
type NoteService struct {
	store *storage.NoteStore
}

// NewNoteService creates a new note service
func NewNoteService(store *storage.NoteStore) *NoteService {
	return &NoteService{
		store: store,
	}
}

// LoadNotes initializes the note store with an encryption key
func (s *NoteService) LoadNotes(key []byte) error {
	return s.store.LoadNotes(key)
}

// GetAllNotes returns all notes
func (s *NoteService) GetAllNotes() []*models.Note {
	return s.store.GetAllNotes()
}

// GetNote returns a specific note by ID
func (s *NoteService) GetNote(id string) (*models.Note, error) {
	return s.store.GetNote(id)
}

// CreateNote creates a new note
func (s *NoteService) CreateNote(content string, key []byte) (*models.Note, error) {
	return s.store.CreateNote(content, key)
}

// UpdateNote updates an existing note
func (s *NoteService) UpdateNote(id, content string, key []byte) (*models.Note, error) {
	return s.store.UpdateNote(id, content, key)
}

// DeleteNote deletes a note
func (s *NoteService) DeleteNote(id string) error {
	return s.store.DeleteNote(id)
}

// SearchNotes searches for notes containing the query
func (s *NoteService) SearchNotes(query string) []*models.Note {
	return s.store.SearchNotes(query)
}

// SyncFromDisk refreshes notes from disk
func (s *NoteService) SyncFromDisk() error {
	return s.store.RefreshFromDisk()
}

// ReencryptAllNotes re-encrypts all notes with a new password
func (s *NoteService) ReencryptAllNotes(oldPassword, newPassword, notesPath string, authManager *auth.Manager) error {
	oldKey := crypto.DeriveKey(oldPassword)
	newKey := crypto.DeriveKey(newPassword)

	noteFiles, err := filepath.Glob(filepath.Join(notesPath, "*.json"))
	if err != nil {
		return fmt.Errorf("failed to list note files: %v", err)
	}

	var corruptedNotes []string
	for _, file := range noteFiles {
		data, err := os.ReadFile(file)
		if err != nil {
			corruptedNotes = append(corruptedNotes, filepath.Base(file))
			continue
		}

		var encryptedNote models.EncryptedNote
		if err := json.Unmarshal(data, &encryptedNote); err != nil {
			corruptedNotes = append(corruptedNotes, filepath.Base(file))
			continue
		}

		// Decrypt with old key
		decryptedContent, err := crypto.Decrypt(encryptedNote.EncryptedData, oldKey)
		if err != nil {
			corruptedNotes = append(corruptedNotes, encryptedNote.ID)
			continue
		}

		// Create note with decrypted content
		note := &models.Note{
			ID:        encryptedNote.ID,
			Content:   decryptedContent,
			CreatedAt: encryptedNote.CreatedAt,
			UpdatedAt: encryptedNote.UpdatedAt,
		}

		// Save with new key
		if err := s.store.SaveNoteDirect(note, newKey); err != nil {
			return fmt.Errorf("failed to save note %s: %v", note.ID, err)
		}
	}

	// Store new password hash only after all notes are successfully re-encrypted
	if err := authManager.StorePasswordHash(newPassword); err != nil {
		return fmt.Errorf("failed to update password hash: %v", err)
	}

	if len(corruptedNotes) > 0 {
		return fmt.Errorf("password changed successfully, but %d corrupted notes were skipped: %v", len(corruptedNotes), corruptedNotes)
	}

	return nil
}
