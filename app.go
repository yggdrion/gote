package main

import (
	"context"
	"fmt"
	"log"
	"os"

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
func (a *App) GetAllNotes() []*models.Note {
	return a.store.GetAllNotes()
}

func (a *App) GetNote(id string) (*models.Note, error) {
	return a.store.GetNote(id)
}

func (a *App) CreateNote(content string) (*models.Note, error) {
	if a.currentKey == nil {
		return nil, fmt.Errorf("not authenticated")
	}
	return a.store.CreateNote(content, a.currentKey)
}

func (a *App) UpdateNote(id, content string) (*models.Note, error) {
	if a.currentKey == nil {
		return nil, fmt.Errorf("not authenticated")
	}
	return a.store.UpdateNote(id, content, a.currentKey)
}

func (a *App) DeleteNote(id string) error {
	return a.store.DeleteNote(id)
}

func (a *App) SearchNotes(query string) []*models.Note {
	return a.store.SearchNotes(query)
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

func (a *App) ChangePassword(oldPassword, newPassword string) error {
	if !a.authManager.VerifyPassword(oldPassword) {
		return fmt.Errorf("invalid current password")
	}
	err := a.authManager.StorePasswordHash(newPassword)
	if err == nil {
		a.currentKey = crypto.DeriveKey(newPassword)
	}
	return err
}

// Greet returns a greeting for the given name (keeping for compatibility)
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}
