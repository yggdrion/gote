package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gote/pkg/auth"
	"gote/pkg/config"
	"gote/pkg/crypto"
	"gote/pkg/models"
	"gote/pkg/storage"
)

// App struct
type App struct {
	ctx         context.Context
	authManager *auth.Manager
	store       *storage.NoteStore
	config      *config.Config
	currentKey  []byte
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Printf("Failed to load configuration, using defaults: %v", err)
		cfg = &config.Config{
			NotesPath:        "./data/notes",
			PasswordHashPath: "./data/password_hash",
		}
	}

	// Ensure data directory exists
	if err := os.MkdirAll("./data", 0755); err != nil {
		log.Printf("Failed to create data directory: %v", err)
	}

	// Initialize components
	a.authManager = auth.NewManager(cfg.PasswordHashPath)
	a.store = storage.NewNoteStore(cfg.NotesPath)
	a.config = cfg

	log.Printf("Note app initialized:")
	log.Printf("  Notes directory: %s", cfg.NotesPath)
	log.Printf("  Password hash file: %s", cfg.PasswordHashPath)
}

// Authentication methods
func (a *App) IsPasswordSet() bool {
	_, err := os.Stat(a.config.PasswordHashPath)
	return err == nil
}

func (a *App) SetPassword(password string) error {
	err := a.authManager.StorePasswordHash(password)
	if err == nil {
		// Generate encryption key from password using proper key derivation
		a.currentKey = crypto.DeriveKey(password)
		// Load existing notes with the new key
		a.store.LoadNotes(a.currentKey)
	}
	return err
}

func (a *App) VerifyPassword(password string) bool {
	if a.authManager.VerifyPassword(password) {
		a.currentKey = crypto.DeriveKey(password)
		// Load notes with the key
		a.store.LoadNotes(a.currentKey)
		return true
	}
	return false
}

// Note management methods
func (a *App) GetAllNotes() []WailsNote {
	notes := a.store.GetAllNotes()
	return ConvertToWailsNotes(notes)
}

func (a *App) GetNote(id string) (WailsNote, error) {
	note, err := a.store.GetNote(id)
	if err != nil {
		return WailsNote{}, err
	}
	return ConvertToWailsNote(note), nil
}

func (a *App) CreateNote(content string) (WailsNote, error) {
	if a.currentKey == nil {
		return WailsNote{}, fmt.Errorf("not authenticated")
	}
	note, err := a.store.CreateNote(content, a.currentKey)
	if err != nil {
		return WailsNote{}, err
	}
	return ConvertToWailsNote(note), nil
}

func (a *App) UpdateNote(id, content string) (WailsNote, error) {
	if a.currentKey == nil {
		return WailsNote{}, fmt.Errorf("not authenticated")
	}
	note, err := a.store.UpdateNote(id, content, a.currentKey)
	if err != nil {
		return WailsNote{}, err
	}
	return ConvertToWailsNote(note), nil
}

func (a *App) DeleteNote(id string) error {
	return a.store.DeleteNote(id)
}

func (a *App) SearchNotes(query string) []WailsNote {
	notes := a.store.SearchNotes(query)
	return ConvertToWailsNotes(notes)
}

func (a *App) SyncFromDisk() error {
	if a.currentKey == nil {
		return fmt.Errorf("not authenticated")
	}
	return a.store.LoadNotes(a.currentKey)
}

// Settings methods
func (a *App) GetSettings() map[string]interface{} {
	return map[string]interface{}{
		"notesPath":        a.config.NotesPath,
		"passwordHashPath": a.config.PasswordHashPath,
	}
}

func (a *App) UpdateSettings(notesPath, passwordHashPath string) error {
	// Validate paths
	if notesPath == "" {
		notesPath = config.GetDefaultDataPath()
	}
	if passwordHashPath == "" {
		passwordHashPath = config.GetDefaultPasswordHashPath()
	}

	// Create directories if they don't exist
	if err := os.MkdirAll(notesPath, 0755); err != nil {
		return fmt.Errorf("failed to create notes directory: %v", err)
	}

	passwordDir := filepath.Dir(passwordHashPath)
	if err := os.MkdirAll(passwordDir, 0755); err != nil {
		return fmt.Errorf("failed to create password hash directory: %v", err)
	}

	// Update configuration
	a.config.NotesPath = notesPath
	a.config.PasswordHashPath = passwordHashPath

	// Save configuration to file
	if err := a.config.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %v", err)
	}

	// Clear current session and unload notes for security
	// This ensures the user must re-authenticate with the new configuration
	a.currentKey = nil

	// Update components with new paths
	a.authManager = auth.NewManager(a.config.PasswordHashPath)
	a.store = storage.NewNoteStore(a.config.NotesPath)

	log.Printf("Settings updated:")
	log.Printf("  Notes directory: %s", a.config.NotesPath)
	log.Printf("  Password hash file: %s", a.config.PasswordHashPath)
	log.Printf("User logged out - re-authentication required")

	return nil
}

func (a *App) ChangePassword(oldPassword, newPassword string) error {
	if !a.authManager.VerifyPassword(oldPassword) {
		return fmt.Errorf("invalid current password")
	}

	// Derive old and new keys
	oldKey := crypto.DeriveKey(oldPassword)
	newKey := crypto.DeriveKey(newPassword)

	// Re-encrypt all notes from disk
	noteFiles, err := filepath.Glob(filepath.Join(a.config.NotesPath, "*.json"))
	if err != nil {
		return fmt.Errorf("failed to list note files: %v", err)
	}

	var corruptedNotes []string
	for _, file := range noteFiles {
		data, err := os.ReadFile(file)
		if err != nil {
			corruptedNotes = append(corruptedNotes, filepath.Base(file))
			log.Printf("Error reading file %s: %v", file, err)
			continue
		}

		var encryptedNote models.EncryptedNote
		if err := json.Unmarshal(data, &encryptedNote); err != nil {
			corruptedNotes = append(corruptedNotes, filepath.Base(file))
			log.Printf("Error unmarshalling file %s: %v", file, err)
			continue
		}

		// Decrypt with old key
		decryptedContent, err := crypto.Decrypt(encryptedNote.EncryptedData, oldKey)
		if err != nil {
			corruptedNotes = append(corruptedNotes, encryptedNote.ID)
			log.Printf("Error decrypting note %s: %v", encryptedNote.ID, err)
			continue
		}

		// Create note with decrypted content
		note := &models.Note{
			ID:        encryptedNote.ID,
			Content:   decryptedContent,
			CreatedAt: encryptedNote.CreatedAt,
			UpdatedAt: encryptedNote.UpdatedAt,
		}

		// Save with new key using SaveNoteDirect
		if err := a.store.SaveNoteDirect(note, newKey); err != nil {
			return fmt.Errorf("failed to save note %s: %v", note.ID, err)
		}
	}

	// Store new password hash only after all notes are successfully re-encrypted
	if err := a.authManager.StorePasswordHash(newPassword); err != nil {
		return fmt.Errorf("failed to update password hash: %v", err)
	}

	// Clear the current session - user will need to log in again with new password
	a.currentKey = nil

	if len(corruptedNotes) > 0 {
		log.Printf("Password changed successfully. %d corrupted notes were skipped: %v", len(corruptedNotes), corruptedNotes)
	}

	return nil
}

func (a *App) ResetApplication() error {
	// Clear the current key to prevent any operations
	a.currentKey = nil

	// Remove password hash file only - keep the encrypted notes
	if err := a.authManager.RemovePasswordHash(); err != nil {
		return fmt.Errorf("failed to remove password hash: %v", err)
	}

	return nil
}

// Logout clears the current session without removing password hash
func (a *App) Logout() error {
	// Clear the current key to end the session
	a.currentKey = nil
	return nil
}

// CreateBackup creates a zip backup of all notes
func (a *App) CreateBackup() (string, error) {
	if a.currentKey == nil {
		return "", fmt.Errorf("not authenticated")
	}

	// Use the storage backup function
	backupPath, err := storage.BackupNotes(a.config.NotesPath, "")
	if err != nil {
		return "", fmt.Errorf("failed to create backup: %v", err)
	}

	return backupPath, nil
}

// Greet returns a greeting for the given name (keeping for compatibility)
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}
