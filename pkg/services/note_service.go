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

// DeleteNote deletes a note
func (s *NoteService) DeleteNote(id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("note ID cannot be empty")
	}

	return s.store.DeleteNote(id)
}

// MoveNoteToTrash moves a note to the trash folder
func (s *NoteService) MoveNoteToTrash(id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("note ID cannot be empty")
	}

	return s.store.MoveNoteToTrash(id)
}

// RestoreNoteFromTrash restores a note from the trash folder
func (s *NoteService) RestoreNoteFromTrash(id string, key []byte) error {
	if key == nil {
		return fmt.Errorf("authentication required")
	}

	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("note ID cannot be empty")
	}

	return s.store.RestoreNoteFromTrash(id, key)
}

// GetTrashedNotes returns all notes in the trash folder
func (s *NoteService) GetTrashedNotes(key []byte) ([]*models.Note, error) {
	if key == nil {
		return nil, fmt.Errorf("authentication required")
	}

	return s.store.GetTrashedNotes(key)
}

// PermanentlyDeleteNote permanently deletes a note from the trash folder
func (s *NoteService) PermanentlyDeleteNote(id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("note ID cannot be empty")
	}

	return s.store.PermanentlyDeleteNote(id)
}

// EmptyTrash permanently deletes all notes in the trash folder
func (s *NoteService) EmptyTrash() error {
	return s.store.EmptyTrash()
}

// SearchNotes searches for notes containing the query
func (s *NoteService) SearchNotes(query string) []*models.Note {
	return s.store.SearchNotes(query)
}

// SyncFromDisk syncs notes from disk
func (s *NoteService) SyncFromDisk() error {
	return s.store.RefreshFromDisk()
}
