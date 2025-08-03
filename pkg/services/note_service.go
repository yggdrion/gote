package services

import (
	"encoding/json"
	"fmt"
	"gote/pkg/auth"
	"gote/pkg/crypto"
	"gote/pkg/errors"
	"gote/pkg/models"
	"gote/pkg/storage"
	"log"
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

// ReencryptAllNotes re-encrypts all notes with a new password
func (s *NoteService) ReencryptAllNotes(oldPassword, newPassword, notesPath string, authManager *auth.Manager) error {
	configPath := filepath.Join(s.store.GetDataDir(), ".keyconfig.json")

	oldKey, err := crypto.DeriveKeyEnhanced(oldPassword, configPath)
	if err != nil {
		return fmt.Errorf("failed to derive old key: %v", err)
	}

	newKey, err := crypto.DeriveKeyEnhanced(newPassword, configPath)
	if err != nil {
		return fmt.Errorf("failed to derive new key: %v", err)
	}

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
