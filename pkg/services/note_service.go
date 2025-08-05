package services

import (
	"fmt"
	"gote/pkg/models"
	"gote/pkg/storage"
	"strings"
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
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("note ID cannot be empty")
	}

	return s.store.GetNote(id)
}

// CreateNote creates a new note
func (s *NoteService) CreateNote(content string, key []byte) (*models.Note, error) {
	if key == nil {
		return nil, fmt.Errorf("authentication required")
	}

	// Allow empty content for new notes - users can fill them in later
	return s.store.CreateNote(content, key)
}

// CreateNoteWithCategory creates a new note with a specific category
func (s *NoteService) CreateNoteWithCategory(content string, category models.NoteCategory, key []byte) (*models.Note, error) {
	if key == nil {
		return nil, fmt.Errorf("authentication required")
	}

	// Allow empty content for new notes - users can fill them in later
	return s.store.CreateNoteWithCategory(content, category, key)
}

// UpdateNote updates an existing note
func (s *NoteService) UpdateNote(id, content string, key []byte) (*models.Note, error) {
	if key == nil {
		return nil, fmt.Errorf("authentication required")
	}

	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("note ID cannot be empty")
	}

	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("note content cannot be empty")
	}

	return s.store.UpdateNote(id, content, key)
}

// UpdateNoteCategory updates the category of a note
func (s *NoteService) UpdateNoteCategory(id string, category models.NoteCategory, key []byte) (*models.Note, error) {
	if key == nil {
		return nil, fmt.Errorf("authentication required")
	}

	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("note ID cannot be empty")
	}

	return s.store.UpdateNoteCategory(id, category, key)
}

// DeleteNote moves a note to trash (or permanently deletes if already in trash)
func (s *NoteService) DeleteNote(id string, key []byte) error {
	if key == nil {
		return fmt.Errorf("authentication required")
	}

	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("note ID cannot be empty")
	}

	// Get the note to check its current category
	note, err := s.store.GetNote(id)
	if err != nil {
		return err
	}

	// If the note is already in trash, permanently delete it
	if note.Category == models.CategoryTrash {
		return s.store.PermanentlyDeleteNote(id)
	}

	// Otherwise, move it to trash
	_, err = s.store.MoveToTrash(id, key)
	return err
}

// GetNotesByCategory returns notes filtered by category
func (s *NoteService) GetNotesByCategory(category models.NoteCategory) []*models.Note {
	return s.store.GetNotesByCategory(category)
}

// MoveToTrash moves a note to trash category
func (s *NoteService) MoveToTrash(id string, key []byte) (*models.Note, error) {
	if key == nil {
		return nil, fmt.Errorf("authentication required")
	}

	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("note ID cannot be empty")
	}

	return s.store.MoveToTrash(id, key)
}

// PermanentlyDeleteNote permanently deletes a note (only works for trash items)
func (s *NoteService) PermanentlyDeleteNote(id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("note ID cannot be empty")
	}

	return s.store.PermanentlyDeleteNote(id)
}

// SearchNotes searches for notes containing the query
func (s *NoteService) SearchNotes(query string) []*models.Note {
	return s.store.SearchNotes(query)
}

// SyncFromDisk syncs notes from disk
func (s *NoteService) SyncFromDisk() error {
	return s.store.RefreshFromDisk()
}
