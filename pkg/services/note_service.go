package services

import (
	"fmt"
	"gote/pkg/auth"
	"gote/pkg/errors"
	"gote/pkg/models"
	"gote/pkg/storage"
	"log"
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

// GetNote returns a specific note by ID with validation
func (s *NoteService) GetNote(id string) (*models.Note, error) {
	// Validate note ID
	validator := errors.NewValidator()
	if result := validator.ValidateNoteID(id); !result.IsValid {
		err := result.GetFirstError()
		err.Log()
		return nil, err
	}

	note, err := s.store.GetNote(id)
	if err != nil {
		// Wrap storage errors with user-friendly messages
		if err.Error() == "note not found" {
			appErr := errors.ErrNoteNotFound.WithContext("noteId", id)
			appErr.Log()
			return nil, appErr
		}

		appErr := errors.Wrap(err, errors.ErrTypeFileSystem, "NOTE_READ_FAILED",
			"failed to read note").
			WithUserMessage("Unable to load the requested note").
			WithContext("noteId", id)
		appErr.Log()
		return nil, appErr
	}

	return note, nil
}

// CreateNote creates a new note with validation and error handling
func (s *NoteService) CreateNote(content string, key []byte) (*models.Note, error) {
	if key == nil {
		err := errors.ErrNotAuthenticated
		err.Log()
		return nil, err
	}

	// Validate note content
	validator := errors.NewValidator()
	if result := validator.ValidateNoteContent(content); !result.IsValid {
		err := result.GetFirstError()
		err.Log()
		return nil, err
	}

	// Create note with retry logic for transient failures
	retryHandler := errors.NewRetryHandler(3)
	var note *models.Note

	err := retryHandler.Execute(func() error {
		var err error
		note, err = s.store.CreateNote(content, key)
		if err != nil {
			return errors.Wrap(err, errors.ErrTypeFileSystem, "NOTE_CREATE_FAILED",
				"failed to create note").
				WithUserMessage("Unable to save the note. Please try again").
				WithRetryable(true)
		}
		return nil
	})

	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			appErr.Log()
			return nil, appErr
		}
		return nil, err
	}

	log.Printf("Note created successfully: %s", note.ID)
	return note, nil
}

// UpdateNote updates an existing note with validation and error handling
func (s *NoteService) UpdateNote(id, content string, key []byte) (*models.Note, error) {
	if key == nil {
		err := errors.ErrNotAuthenticated
		err.Log()
		return nil, err
	}

	// Validate inputs
	validator := errors.NewValidator()
	if result := validator.ValidateNoteID(id); !result.IsValid {
		err := result.GetFirstError()
		err.Log()
		return nil, err
	}

	if result := validator.ValidateNoteContent(content); !result.IsValid {
		err := result.GetFirstError()
		err.Log()
		return nil, err
	}

	// Update note with retry logic
	retryHandler := errors.NewRetryHandler(3)
	var note *models.Note

	err := retryHandler.Execute(func() error {
		var err error
		note, err = s.store.UpdateNote(id, content, key)
		if err != nil {
			if err.Error() == "note not found" {
				return errors.ErrNoteNotFound.WithContext("noteId", id)
			}
			return errors.Wrap(err, errors.ErrTypeFileSystem, "NOTE_UPDATE_FAILED",
				"failed to update note").
				WithUserMessage("Unable to save changes. Please try again").
				WithRetryable(true).
				WithContext("noteId", id)
		}
		return nil
	})

	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			appErr.Log()
			return nil, appErr
		}
		return nil, err
	}

	log.Printf("Note updated successfully: %s", note.ID)
	return note, nil
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

// ReencryptAllNotes - disabled in simplified mode
func (s *NoteService) ReencryptAllNotes(oldPassword, newPassword, notesPath string, authManager *auth.Manager) error {
	return fmt.Errorf("password change not supported in simplified mode")
}
