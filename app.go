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
	"gote/pkg/errors"
	"gote/pkg/models"
	"gote/pkg/services"
	"gote/pkg/storage"
	"gote/pkg/types"
)

// App struct
type App struct {
	ctx         context.Context
	authManager *auth.Manager
	store       *storage.NoteStore
	imageStore  *storage.ImageStore
	config      *config.Config
	currentKey  []byte

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

// CompleteInitialSetup handles the first-time setup process with enhanced validation
func (a *App) CompleteInitialSetup(notesPath, passwordHashPath, password, confirmPassword string) error {
	// Create validator for input validation
	validator := errors.NewValidator()

	// Validate password
	if result := validator.ValidatePassword(password); !result.IsValid {
		err := result.GetFirstError()
		err.Log()
		return err
	}

	// Validate password match
	if result := validator.ValidatePasswordMatch(password, confirmPassword); !result.IsValid {
		err := result.GetFirstError()
		err.Log()
		return err
	}

	// Use defaults if paths are empty
	if notesPath == "" {
		notesPath = config.GetDefaultDataPath()
	}
	if passwordHashPath == "" {
		passwordHashPath = config.GetDefaultPasswordHashPath()
	}

	// Validate directory paths
	if result := validator.ValidateDirectoryPath(notesPath); !result.IsValid {
		err := result.GetFirstError()
		err.Log()
		return err
	}

	if result := validator.ValidateDirectoryPath(filepath.Dir(passwordHashPath)); !result.IsValid {
		err := result.GetFirstError()
		err.Log()
		return err
	}

	// Create directories with retry logic
	retryHandler := errors.NewRetryHandler(3)

	err := retryHandler.Execute(func() error {
		if err := os.MkdirAll(notesPath, 0755); err != nil {
			return errors.Wrap(err, errors.ErrTypeFileSystem, "DIR_CREATE_FAILED",
				"failed to create notes directory").
				WithUserMessage("Unable to create notes directory. Check permissions").
				WithRetryable(true)
		}

		passwordDir := filepath.Dir(passwordHashPath)
		if err := os.MkdirAll(passwordDir, 0755); err != nil {
			return errors.Wrap(err, errors.ErrTypeFileSystem, "DIR_CREATE_FAILED",
				"failed to create password hash directory").
				WithUserMessage("Unable to create password directory. Check permissions").
				WithRetryable(true)
		}
		return nil
	})

	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			appErr.Log()
			return appErr
		}
		return err
	}

	// Create and save configuration
	a.config = &config.Config{
		NotesPath:        notesPath,
		PasswordHashPath: passwordHashPath,
	}

	if err := a.config.Save(); err != nil {
		appErr := errors.Wrap(err, errors.ErrTypeConfig, "CONFIG_SAVE_FAILED",
			"failed to save configuration").
			WithUserMessage("Unable to save settings. Check permissions")
		appErr.Log()
		return appErr
	}

	// Initialize components with new configuration
	a.authManager = auth.NewManagerWithNotesDir(a.config.PasswordHashPath, a.config.NotesPath)
	a.store = storage.NewNoteStore(a.config.NotesPath)
	a.imageStore = storage.NewImageStore(a.config.NotesPath)

	// Set the initial password with retry logic
	err = retryHandler.Execute(func() error {
		if err := a.authManager.StorePasswordHash(password); err != nil {
			return errors.Wrap(err, errors.ErrTypeAuth, "PASSWORD_STORE_FAILED",
				"failed to store password").
				WithUserMessage("Unable to save password. Please try again").
				WithRetryable(true)
		}
		return nil
	})

	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			appErr.Log()
			return appErr
		}
		return err
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
	// Check if this is a new device with cross-platform config
	hasLocalPassword := !a.authManager.IsFirstTimeSetup()

	// Verify password
	if !a.authManager.VerifyPassword(password) {
		return false
	}

	// If no local password hash exists but cross-platform config does,
	// sync the cross-platform config to local storage
	if !hasLocalPassword {
		if err := a.authManager.SyncFromCrossPlatform(password); err != nil {
			log.Printf("Warning: Could not sync from cross-platform config: %v", err)
			// Continue anyway - the verification already passed
		} else {
			log.Printf("Successfully synced authentication from cross-platform config")
		}
	}

	// Derive encryption key
	key, err := a.authManager.DeriveEncryptionKey(password)
	if err != nil {
		log.Printf("Failed to derive encryption key: %v", err)
		return false
	}

	a.currentKey = key
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
	if a.currentKey == nil {
		err := errors.ErrNotAuthenticated
		err.Log()
		return types.WailsNote{}, err
	}

	var note *models.Note
	var err error

	if a.noteService != nil {
		note, err = a.noteService.CreateNote(content, a.currentKey)
	} else {
		// Fallback with basic validation
		validator := errors.NewValidator()
		if result := validator.ValidateNoteContent(content); !result.IsValid {
			appErr := result.GetFirstError()
			appErr.Log()
			return types.WailsNote{}, appErr
		}

		note, err = a.store.CreateNote(content, a.currentKey)
		if err != nil {
			appErr := errors.Wrap(err, errors.ErrTypeFileSystem, "NOTE_CREATE_FAILED",
				"failed to create note").
				WithUserMessage("Unable to save the note. Please try again")
			appErr.Log()
			return types.WailsNote{}, appErr
		}
	}

	if err != nil {
		return types.WailsNote{}, err
	}
	return types.ConvertToWailsNote(note), nil
}

func (a *App) UpdateNote(id, content string) (types.WailsNote, error) {
	if a.currentKey == nil {
		err := errors.ErrNotAuthenticated
		err.Log()
		return types.WailsNote{}, err
	}

	var note *models.Note
	var err error

	if a.noteService != nil {
		note, err = a.noteService.UpdateNote(id, content, a.currentKey)
	} else {
		// Fallback with basic validation
		validator := errors.NewValidator()
		if result := validator.ValidateNoteID(id); !result.IsValid {
			appErr := result.GetFirstError()
			appErr.Log()
			return types.WailsNote{}, appErr
		}

		if result := validator.ValidateNoteContent(content); !result.IsValid {
			appErr := result.GetFirstError()
			appErr.Log()
			return types.WailsNote{}, appErr
		}

		note, err = a.store.UpdateNote(id, content, a.currentKey)
		if err != nil {
			if err.Error() == "note not found" {
				appErr := errors.ErrNoteNotFound.WithContext("noteId", id)
				appErr.Log()
				return types.WailsNote{}, appErr
			}

			appErr := errors.Wrap(err, errors.ErrTypeFileSystem, "NOTE_UPDATE_FAILED",
				"failed to update note").
				WithUserMessage("Unable to save changes. Please try again").
				WithContext("noteId", id)
			appErr.Log()
			return types.WailsNote{}, appErr
		}
	}

	if err != nil {
		return types.WailsNote{}, err
	}
	return types.ConvertToWailsNote(note), nil
}

func (a *App) DeleteNote(id string) error {
	// First, get the note content to extract image IDs before deletion
	var noteContent string
	if a.noteService != nil {
		if note, err := a.noteService.GetNote(id); err == nil && note != nil {
			noteContent = note.Content
		}
	} else {
		if note, err := a.store.GetNote(id); err == nil && note != nil {
			noteContent = note.Content
		}
	}

	// Extract image IDs from the note content
	imageIDs := a.extractImageIDsFromContent(noteContent)

	// Delete the note
	var err error
	if a.noteService != nil {
		err = a.noteService.DeleteNote(id)
	} else {
		err = a.store.DeleteNote(id)
	}

	if err != nil {
		return err
	}

	// Clean up orphaned images
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

func (a *App) SyncFromDisk() error {
	if a.currentKey == nil {
		return fmt.Errorf("not authenticated")
	}

	if a.noteService != nil {
		return a.noteService.SyncFromDisk()
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

// Error handling helpers for frontend

// HandleError converts application errors to frontend-friendly format
func (a *App) HandleError(err error) map[string]interface{} {
	errorHandler := errors.NewErrorHandler()
	frontendErr := errorHandler.HandleError(err, nil)

	return map[string]interface{}{
		"error": frontendErr,
	}
}

// ValidateSetupInputs validates initial setup inputs
func (a *App) ValidateSetupInputs(notesPath, passwordHashPath, password, confirmPassword string) map[string]interface{} {
	validator := errors.NewValidator()
	var validationErrors []*errors.AppError

	// Validate password
	if result := validator.ValidatePassword(password); !result.IsValid {
		validationErrors = append(validationErrors, result.Errors...)
	}

	// Validate password match
	if result := validator.ValidatePasswordMatch(password, confirmPassword); !result.IsValid {
		validationErrors = append(validationErrors, result.Errors...)
	}

	// Validate paths if provided
	if notesPath != "" {
		if result := validator.ValidateDirectoryPath(notesPath); !result.IsValid {
			validationErrors = append(validationErrors, result.Errors...)
		}
	}

	if passwordHashPath != "" {
		if result := validator.ValidateDirectoryPath(filepath.Dir(passwordHashPath)); !result.IsValid {
			validationErrors = append(validationErrors, result.Errors...)
		}
	}

	if len(validationErrors) > 0 {
		// Return the first validation error
		errorHandler := errors.NewErrorHandler()
		frontendErr := errorHandler.HandleError(validationErrors[0], nil)

		return map[string]interface{}{
			"valid": false,
			"error": frontendErr,
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
	if a.currentKey == nil {
		return nil, fmt.Errorf("not authenticated")
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
	if a.currentKey == nil {
		return nil, fmt.Errorf("not authenticated")
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
	if a.currentKey == nil {
		return fmt.Errorf("not authenticated")
	}

	return a.imageStore.DeleteImage(imageID)
}

// ListImages returns a list of all stored images
func (a *App) ListImages() ([]*models.Image, error) {
	if a.currentKey == nil {
		return nil, fmt.Errorf("not authenticated")
	}

	return a.imageStore.ListImages()
}

// GetImageAsDataURL returns an image as a data URL for embedding in HTML
func (a *App) GetImageAsDataURL(imageID string) (string, error) {
	if a.currentKey == nil {
		return "", fmt.Errorf("not authenticated")
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
