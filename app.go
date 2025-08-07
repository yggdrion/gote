package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"gote/pkg/auth"
	"gote/pkg/config"
	"gote/pkg/models"
	"gote/pkg/services"
	"gote/pkg/storage"
	"gote/pkg/types"
)

// App struct
type App struct {
	ctx            context.Context
	authManager    *auth.Manager
	store          *storage.NoteStore
	imageStore     *storage.ImageStore
	config         *config.Config
	currentKey     []byte
	currentSession string // Track current session ID

	// Service layer - simplified architecture
	noteService *services.NoteService
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Check if this is first-time setup (no config file exists)
	configExists := a.IsConfigured()

	if configExists {
		// Load existing configuration
		cfg, err := config.Load()
		if err != nil {
			log.Printf("Failed to load configuration, using defaults: %v", err)
			cfg = &config.Config{
				NotesPath:        config.GetDefaultDataPath(),
				PasswordHashPath: config.GetDefaultPasswordHashPath(),
			}
		}

		// Initialize components
		a.authManager = auth.NewManagerWithNotesDir(cfg.PasswordHashPath, cfg.NotesPath)
		a.store = storage.NewNoteStore(cfg.NotesPath)
		a.imageStore = storage.NewImageStore(cfg.NotesPath)
		a.config = cfg

		// Initialize services - simplified
		a.noteService = services.NewNoteService(a.store)

		// Start background session cleanup
		go a.startSessionCleanup()

		log.Printf("Note app initialized:")
		log.Printf("  Configuration file: %s", config.GetConfigFilePath())
		log.Printf("  Password hash file: %s", cfg.PasswordHashPath)
		log.Printf("  Notes directory: %s", cfg.NotesPath)
	} else {
		log.Printf("First-time setup required - no configuration file found")
		// Initialize with default config for now, will be replaced during setup
		a.config = &config.Config{
			NotesPath:        config.GetDefaultDataPath(),
			PasswordHashPath: config.GetDefaultPasswordHashPath(),
		}
	}
}

// Authentication methods
func (a *App) IsPasswordSet() bool {
	return !a.authManager.IsFirstTimeSetup()
}

// IsConfigured checks if the configuration file exists (not first-time setup)
func (a *App) IsConfigured() bool {
	configFile := config.GetConfigFilePath()
	_, err := os.Stat(configFile)
	return err == nil
}

// CompleteInitialSetup handles the first-time setup process
func (a *App) CompleteInitialSetup(notesPath, passwordHashPath, password, confirmPassword string) error {
	// Basic validation
	if len(password) < 6 {
		return fmt.Errorf("password must be at least 6 characters long")
	}

	if password != confirmPassword {
		return fmt.Errorf("passwords do not match")
	}

	// Use defaults if paths are empty
	if notesPath == "" {
		notesPath = config.GetDefaultDataPath()
	}
	if passwordHashPath == "" {
		passwordHashPath = config.GetDefaultPasswordHashPath()
	}

	// Create directories
	if err := os.MkdirAll(notesPath, 0755); err != nil {
		return fmt.Errorf("failed to create notes directory: %v", err)
	}

	passwordDir := filepath.Dir(passwordHashPath)
	if err := os.MkdirAll(passwordDir, 0755); err != nil {
		return fmt.Errorf("failed to create password directory: %v", err)
	}

	// Create and save configuration
	a.config = &config.Config{
		NotesPath:        notesPath,
		PasswordHashPath: passwordHashPath,
	}

	if err := a.config.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %v", err)
	}

	// Initialize components with new configuration
	a.authManager = auth.NewManagerWithNotesDir(a.config.PasswordHashPath, a.config.NotesPath)
	a.store = storage.NewNoteStore(a.config.NotesPath)
	a.imageStore = storage.NewImageStore(a.config.NotesPath)

	// Set the initial password
	if err := a.authManager.StorePasswordHash(password); err != nil {
		return fmt.Errorf("failed to store password: %v", err)
	}

	// Generate encryption key and initialize
	key, err := a.authManager.DeriveEncryptionKey(password)
	if err != nil {
		return fmt.Errorf("failed to derive encryption key: %v", err)
	}
	a.currentKey = key
	a.store.LoadNotes(a.currentKey)
	a.imageStore.SetKey(a.currentKey)

	log.Printf("Initial setup completed:")
	log.Printf("  Configuration file: %s", config.GetConfigFilePath())
	log.Printf("  Password hash file: %s", a.config.PasswordHashPath)
	log.Printf("  Notes directory: %s", a.config.NotesPath)

	return nil
}

func (a *App) SetPassword(password string) error {
	// Store password hash
	err := a.authManager.StorePasswordHash(password)
	if err != nil {
		return fmt.Errorf("failed to store password: %v", err)
	}

	// Derive encryption key
	key, err := a.authManager.DeriveEncryptionKey(password)
	if err != nil {
		return fmt.Errorf("failed to derive encryption key: %v", err)
	}

	a.currentKey = key
	// Load existing notes with the new key
	if a.noteService != nil {
		a.noteService.LoadNotes(a.currentKey)
	} else {
		a.store.LoadNotes(a.currentKey)
	}
	a.imageStore.SetKey(a.currentKey)
	return nil
}

func (a *App) VerifyPassword(password string) bool {
	// Verify password - this will automatically handle cross-platform setup if needed
	if !a.authManager.VerifyPassword(password) {
		return false
	}

	// Derive encryption key
	key, err := a.authManager.DeriveEncryptionKey(password)
	if err != nil {
		log.Printf("Failed to derive encryption key: %v", err)
		return false
	}

	a.currentKey = key

	// Create a new session
	sessionID := a.authManager.CreateSession(key)
	a.currentSession = sessionID

	// Load notes with the key
	if a.noteService != nil {
		a.noteService.LoadNotes(a.currentKey)
	} else {
		a.store.LoadNotes(a.currentKey)
	}
	a.imageStore.SetKey(a.currentKey)
	return true
}

// Note management methods
func (a *App) GetAllNotes() []types.WailsNote {
	var notes []*models.Note
	if a.noteService != nil {
		notes = a.noteService.GetAllNotes()
	} else {
		notes = a.store.GetAllNotes()
	}
	return types.ConvertToWailsNotes(notes)
}

func (a *App) GetNote(id string) (types.WailsNote, error) {
	var note *models.Note
	var err error

	if a.noteService != nil {
		note, err = a.noteService.GetNote(id)
	} else {
		note, err = a.store.GetNote(id)
	}

	if err != nil {
		return types.WailsNote{}, err
	}
	return types.ConvertToWailsNote(note), nil
}

func (a *App) CreateNote(content string) (types.WailsNote, error) {
	if err := a.requireAuth(); err != nil {
		return types.WailsNote{}, err
	}

	note, err := a.noteService.CreateNote(content, a.currentKey)
	if err != nil {
		return types.WailsNote{}, err
	}
	return types.ConvertToWailsNote(note), nil
}

// CreateNoteWithCategory creates a new note with a specific category
func (a *App) CreateNoteWithCategory(content string, category string) (types.WailsNote, error) {
	if err := a.requireAuth(); err != nil {
		return types.WailsNote{}, err
	}

	// Convert string to NoteCategory
	var noteCategory models.NoteCategory
	switch category {
	case "private":
		noteCategory = models.CategoryPrivate
	case "work":
		noteCategory = models.CategoryWork
	case "trash":
		noteCategory = models.CategoryTrash
	default:
		noteCategory = models.CategoryPrivate
	}

	note, err := a.noteService.CreateNoteWithCategory(content, noteCategory, a.currentKey)
	if err != nil {
		return types.WailsNote{}, err
	}
	return types.ConvertToWailsNote(note), nil
}

func (a *App) UpdateNote(id, content string) (types.WailsNote, error) {
	if err := a.requireAuth(); err != nil {
		return types.WailsNote{}, err
	}

	note, err := a.noteService.UpdateNote(id, content, a.currentKey)
	if err != nil {
		return types.WailsNote{}, err
	}
	return types.ConvertToWailsNote(note), nil
}

// UpdateNoteCategory updates the category of a note
func (a *App) UpdateNoteCategory(id string, category string) (types.WailsNote, error) {
	if err := a.requireAuth(); err != nil {
		return types.WailsNote{}, err
	}

	// Convert string to NoteCategory
	var noteCategory models.NoteCategory
	switch category {
	case "private":
		noteCategory = models.CategoryPrivate
	case "work":
		noteCategory = models.CategoryWork
	case "trash":
		noteCategory = models.CategoryTrash
	default:
		return types.WailsNote{}, fmt.Errorf("invalid category: %s", category)
	}

	note, err := a.noteService.UpdateNoteCategory(id, noteCategory, a.currentKey)
	if err != nil {
		return types.WailsNote{}, err
	}
	return types.ConvertToWailsNote(note), nil
}

func (a *App) DeleteNote(id string) error {
	// First, get the note content to extract image IDs before deletion
	var noteContent string
	if note, err := a.noteService.GetNote(id); err == nil && note != nil {
		noteContent = note.Content
	}

	// Extract image IDs from the note content
	imageIDs := a.extractImageIDsFromContent(noteContent)

	// Delete the note (this will move to trash or permanently delete)
	err := a.noteService.DeleteNote(id, a.currentKey)
	if err != nil {
		return err
	}

	// If the note was permanently deleted (from trash), clean up orphaned images
	// Check if note still exists - if not, it was permanently deleted
	if _, err := a.noteService.GetNote(id); err != nil {
		// Note was permanently deleted, clean up images
		for _, imageID := range imageIDs {
			if !a.isImageReferencedByOtherNotes(imageID, id) {
				// Image is not referenced by any other note, safe to delete
				if deleteErr := a.imageStore.DeleteImage(imageID); deleteErr != nil {
					log.Printf("Warning: Failed to delete orphaned image %s: %v", imageID, deleteErr)
					// Don't fail the note deletion if image cleanup fails
				} else {
					log.Printf("Cleaned up orphaned image: %s", imageID)
				}
			}
		}
	}

	return nil
}

func (a *App) SearchNotes(query string) []types.WailsNote {
	var notes []*models.Note
	if a.noteService != nil {
		notes = a.noteService.SearchNotes(query)
	} else {
		notes = a.store.SearchNotes(query)
	}
	return types.ConvertToWailsNotes(notes)
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
	a.authManager = auth.NewManagerWithNotesDir(a.config.PasswordHashPath, a.config.NotesPath)
	a.store = storage.NewNoteStore(a.config.NotesPath)
	a.imageStore = storage.NewImageStore(a.config.NotesPath)

	log.Printf("Settings updated:")
	log.Printf("  Notes directory: %s", a.config.NotesPath)
	log.Printf("  Password hash file: %s", a.config.PasswordHashPath)
	log.Printf("User logged out - re-authentication required")

	return nil
}

func (a *App) ChangePassword(oldPassword, newPassword string) error {
	return fmt.Errorf("password change not supported in simplified mode. Please backup your notes, delete data, and set up fresh with new password")
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
	// Delete the session from auth manager
	if a.currentSession != "" {
		a.authManager.DeleteSession(a.currentSession)
		a.currentSession = ""
	}

	// Clear the current key to end the session
	a.currentKey = nil
	return nil
}

// ValidateCurrentSession checks if the current session is still valid
func (a *App) ValidateCurrentSession() bool {
	if a.currentSession == "" {
		return false
	}

	valid := a.authManager.ValidateSession(a.currentSession)
	if !valid {
		// Session expired, clear local state
		a.currentSession = ""
		a.currentKey = nil
	}
	return valid
}

// IsAuthenticated returns true if user has a valid session
func (a *App) IsAuthenticated() bool {
	return a.ValidateCurrentSession()
}

// requireAuth checks if user is authenticated and returns error if not
func (a *App) requireAuth() error {
	if !a.IsAuthenticated() {
		return fmt.Errorf("session expired - re-authentication required")
	}
	return nil
}

// startSessionCleanup runs a background goroutine to clean up expired sessions
func (a *App) startSessionCleanup() {
	ticker := time.NewTicker(5 * time.Minute) // Clean up every 5 minutes
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if a.authManager != nil {
				a.authManager.CleanupExpiredSessions()
			}
		case <-a.ctx.Done():
			return
		}
	}
}

// CreateBackup creates a zip backup of all notes
func (a *App) CreateBackup() (string, error) {
	if err := a.requireAuth(); err != nil {
		return "", err
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

// Error handling helpers for frontend

// HandleError converts application errors to frontend-friendly format
func (a *App) HandleError(err error) map[string]interface{} {
	return map[string]interface{}{
		"error": map[string]interface{}{
			"message":     err.Error(),
			"userMessage": err.Error(),
			"code":        "GENERIC_ERROR",
		},
	}
}

// ValidateSetupInputs validates initial setup inputs
func (a *App) ValidateSetupInputs(notesPath, passwordHashPath, password, confirmPassword string) map[string]interface{} {
	// Basic validation
	if len(password) < 6 {
		return map[string]interface{}{
			"valid": false,
			"error": map[string]interface{}{
				"message":     "Password must be at least 6 characters long",
				"userMessage": "Password must be at least 6 characters long",
				"code":        "PASSWORD_TOO_SHORT",
			},
		}
	}

	if password != confirmPassword {
		return map[string]interface{}{
			"valid": false,
			"error": map[string]interface{}{
				"message":     "Passwords do not match",
				"userMessage": "Passwords do not match",
				"code":        "PASSWORD_MISMATCH",
			},
		}
	}

	return map[string]interface{}{
		"valid": true,
	}
}

// Security information methods

// GetSecurityInfo returns information about current security configuration
func (a *App) GetSecurityInfo() map[string]interface{} {
	return map[string]interface{}{
		"method":          "PBKDF2",
		"secure":          true,
		"iterations":      100000,
		"key_length":      32,
		"salt_length":     32,
		"recommendations": []string{"Using OWASP-compliant PBKDF2 with salt"},
	}
}

// IsUsingSecureMethod checks if enhanced security is enabled
func (a *App) IsUsingSecureMethod() bool {
	return true // Always using PBKDF2 in simplified mode
}

// Performance monitoring methods

// GetPerformanceStats returns performance statistics
func (a *App) GetPerformanceStats() map[string]interface{} {
	stats := map[string]interface{}{
		"notes_count":       len(a.GetAllNotes()),
		"has_service_layer": a.noteService != nil,
		"security_method":   "PBKDF2",
	}

	// Add basic performance information
	if a.config != nil {
		stats["notes_path"] = a.config.NotesPath
		stats["authenticated"] = a.currentKey != nil
	}

	return stats
}

// Image-related methods

// SaveImageFromClipboard saves an image from clipboard data
func (a *App) SaveImageFromClipboard(imageData string, contentType string) (*models.Image, error) {
	if err := a.requireAuth(); err != nil {
		return nil, err
	}

	// Decode base64 image data
	data, err := base64.StdEncoding.DecodeString(imageData)
	if err != nil {
		return nil, fmt.Errorf("invalid image data: %v", err)
	}

	// Generate filename based on timestamp
	filename := fmt.Sprintf("clipboard_%d", time.Now().Unix())
	if contentType == "image/png" {
		filename += ".png"
	} else if contentType == "image/jpeg" {
		filename += ".jpg"
	} else {
		filename += ".img"
	}

	return a.imageStore.StoreImage(data, contentType, filename)
}

// GetImage retrieves an image by ID and returns base64 encoded data
func (a *App) GetImage(imageID string) (map[string]interface{}, error) {
	if err := a.requireAuth(); err != nil {
		return nil, err
	}

	imageData, image, err := a.imageStore.GetImage(imageID)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":           image.ID,
		"filename":     image.Filename,
		"content_type": image.ContentType,
		"size":         image.Size,
		"created_at":   image.CreatedAt,
		"data":         base64.StdEncoding.EncodeToString(imageData),
	}, nil
}

// DeleteImage removes an image from storage
func (a *App) DeleteImage(imageID string) error {
	if err := a.requireAuth(); err != nil {
		return err
	}

	return a.imageStore.DeleteImage(imageID)
}

// ListImages returns a list of all stored images
func (a *App) ListImages() ([]*models.Image, error) {
	if err := a.requireAuth(); err != nil {
		return nil, err
	}

	return a.imageStore.ListImages()
}

// GetImageAsDataURL returns an image as a data URL for embedding in HTML
func (a *App) GetImageAsDataURL(imageID string) (string, error) {
	if err := a.requireAuth(); err != nil {
		return "", err
	}

	imageData, image, err := a.imageStore.GetImage(imageID)
	if err != nil {
		return "", err
	}

	base64Data := base64.StdEncoding.EncodeToString(imageData)
	return fmt.Sprintf("data:%s;base64,%s", image.ContentType, base64Data), nil
}

// extractImageIDsFromContent extracts image IDs from note content
func (a *App) extractImageIDsFromContent(content string) []string {
	var imageIDs []string

	// Regular expression to match ![alt](image:imageId) pattern
	re := regexp.MustCompile(`!\[[^\]]*\]\(image:([^)]+)\)`)
	matches := re.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 1 {
			imageIDs = append(imageIDs, match[1])
		}
	}

	return imageIDs
}

// isImageReferencedByOtherNotes checks if an image is referenced by notes other than the excluded note
func (a *App) isImageReferencedByOtherNotes(imageID, excludeNoteID string) bool {
	var allNotes []*models.Note

	if a.noteService != nil {
		allNotes = a.noteService.GetAllNotes()
	} else {
		allNotes = a.store.GetAllNotes()
	}

	for _, note := range allNotes {
		if note.ID == excludeNoteID {
			continue // Skip the note being deleted
		}

		imageIDs := a.extractImageIDsFromContent(note.Content)
		for _, id := range imageIDs {
			if id == imageID {
				return true // Image is referenced by another note
			}
		}
	}

	return false
}

// CleanupOrphanedImages removes images that are not referenced by any notes
func (a *App) CleanupOrphanedImages() (int, error) {
	if a.currentKey == nil {
		return 0, fmt.Errorf("not authenticated")
	}

	// Get all stored images
	allImages, err := a.imageStore.ListImages()
	if err != nil {
		return 0, fmt.Errorf("failed to list images: %v", err)
	}

	// Get all notes
	var allNotes []*models.Note
	if a.noteService != nil {
		allNotes = a.noteService.GetAllNotes()
	} else {
		allNotes = a.store.GetAllNotes()
	}

	// Create a set of all referenced image IDs
	referencedImages := make(map[string]bool)
	for _, note := range allNotes {
		imageIDs := a.extractImageIDsFromContent(note.Content)
		for _, imageID := range imageIDs {
			referencedImages[imageID] = true
		}
	}

	// Delete orphaned images
	cleanedUp := 0
	for _, image := range allImages {
		if !referencedImages[image.ID] {
			if err := a.imageStore.DeleteImage(image.ID); err != nil {
				log.Printf("Warning: Failed to delete orphaned image %s: %v", image.ID, err)
			} else {
				log.Printf("Cleaned up orphaned image: %s (%s)", image.ID, image.Filename)
				cleanedUp++
			}
		}
	}

	return cleanedUp, nil
}

// GetImageStats returns statistics about image usage
func (a *App) GetImageStats() (map[string]interface{}, error) {
	if a.currentKey == nil {
		return nil, fmt.Errorf("not authenticated")
	}

	// Get all stored images
	allImages, err := a.imageStore.ListImages()
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %v", err)
	}

	// Get all notes
	var allNotes []*models.Note
	if a.noteService != nil {
		allNotes = a.noteService.GetAllNotes()
	} else {
		allNotes = a.store.GetAllNotes()
	}

	// Create a set of all referenced image IDs
	referencedImages := make(map[string]bool)
	totalReferences := 0
	for _, note := range allNotes {
		imageIDs := a.extractImageIDsFromContent(note.Content)
		totalReferences += len(imageIDs)
		for _, imageID := range imageIDs {
			referencedImages[imageID] = true
		}
	}

	// Calculate total size
	var totalSize int64
	orphanedCount := 0
	for _, image := range allImages {
		totalSize += image.Size
		if !referencedImages[image.ID] {
			orphanedCount++
		}
	}

	return map[string]interface{}{
		"total_images":      len(allImages),
		"referenced_images": len(referencedImages),
		"orphaned_images":   orphanedCount,
		"total_references":  totalReferences,
		"total_size_bytes":  totalSize,
		"total_size_mb":     float64(totalSize) / (1024 * 1024),
	}, nil
}

// GetNotesByCategory returns notes filtered by category
func (a *App) GetNotesByCategory(category string) []types.WailsNote {
	if a.noteService == nil {
		return []types.WailsNote{}
	}

	// Convert string to NoteCategory
	var noteCategory models.NoteCategory
	switch category {
	case "private":
		noteCategory = models.CategoryPrivate
	case "work":
		noteCategory = models.CategoryWork
	case "trash":
		noteCategory = models.CategoryTrash
	default:
		return []types.WailsNote{}
	}

	notes := a.noteService.GetNotesByCategory(noteCategory)
	return types.ConvertToWailsNotes(notes)
} // MoveToTrash moves a note to trash category
func (a *App) MoveToTrash(id string) (types.WailsNote, error) {
	if a.currentKey == nil {
		return types.WailsNote{}, fmt.Errorf("not authenticated")
	}

	note, err := a.noteService.MoveToTrash(id, a.currentKey)
	if err != nil {
		return types.WailsNote{}, err
	}
	return types.ConvertToWailsNote(note), nil
}

// PermanentlyDeleteNote permanently deletes a note (only works for trash items)
func (a *App) PermanentlyDeleteNote(id string) error {
	return a.noteService.PermanentlyDeleteNote(id)
}

// RestoreFromTrash restores a note from trash to its original category
func (a *App) RestoreFromTrash(id string) (types.WailsNote, error) {
	if a.currentKey == nil {
		return types.WailsNote{}, fmt.Errorf("not authenticated")
	}

	note, err := a.noteService.RestoreFromTrash(id, a.currentKey)
	if err != nil {
		return types.WailsNote{}, err
	}
	return types.ConvertToWailsNote(note), nil
}
