package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"

	"gote/pkg/config"
	"gote/pkg/crypto"
	"gote/pkg/models"
	"gote/pkg/storage"
)

// APIHandlers contains API endpoint handlers
type APIHandlers struct {
	store       *storage.NoteStore
	authManager AuthManager
	config      *config.Config
}

// AuthManager interface for dependency injection
// Add methods used in APIHandlers
// (This should match the methods on *auth.Manager)
type AuthManager interface {
	IsAuthenticated(r *http.Request) *models.Session
	VerifyPassword(password string) bool
	StorePasswordHash(password string) error
}

// NewAPIHandlers creates a new API handlers instance
func NewAPIHandlers(store *storage.NoteStore, authManager AuthManager, config *config.Config) *APIHandlers {
	return &APIHandlers{
		store:       store,
		authManager: authManager,
		config:      config,
	}
}

// GetNotesHandler returns all notes as JSON
func (h *APIHandlers) GetNotesHandler(w http.ResponseWriter, r *http.Request) {
	notes := h.store.GetAllNotes()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(notes); err != nil {
		fmt.Printf("[ERROR] encoding notes: %v\n", err)
	}
}

// CreateNoteHandler creates a new note
func (h *APIHandlers) CreateNoteHandler(w http.ResponseWriter, r *http.Request) {
	session := h.authManager.IsAuthenticated(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Content string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	note, err := h.store.CreateNote(req.Content, session.Key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(note); err != nil {
		fmt.Printf("[ERROR] encoding note: %v\n", err)
	}
}

// GetNoteHandler returns a specific note by ID
func (h *APIHandlers) GetNoteHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Invalid note ID", http.StatusBadRequest)
		return
	}

	note, err := h.store.GetNote(id)
	if err != nil {
		http.Error(w, "Note not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(note); err != nil {
		fmt.Printf("[ERROR] encoding note: %v\n", err)
	}
}

// UpdateNoteHandler updates an existing note
func (h *APIHandlers) UpdateNoteHandler(w http.ResponseWriter, r *http.Request) {
	session := h.authManager.IsAuthenticated(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Invalid note ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Content string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	note, err := h.store.UpdateNote(id, req.Content, session.Key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(note); err != nil {
		fmt.Printf("[ERROR] encoding note: %v\n", err)
	}
}

// DeleteNoteHandler deletes a note by ID
func (h *APIHandlers) DeleteNoteHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Invalid note ID", http.StatusBadRequest)
		return
	}

	if err := h.store.DeleteNote(id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// SearchHandler searches notes by query
func (h *APIHandlers) SearchHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Missing query parameter", http.StatusBadRequest)
		return
	}

	notes := h.store.SearchNotes(query)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(notes); err != nil {
		fmt.Printf("[ERROR] encoding notes: %v\n", err)
	}
}

// GetSettingsHandler returns current configuration
func (h *APIHandlers) GetSettingsHandler(w http.ResponseWriter, r *http.Request) {
	session := h.authManager.IsAuthenticated(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(h.config); err != nil {
		fmt.Printf("[ERROR] encoding config: %v\n", err)
	}
}

// SettingsHandler updates configuration
func (h *APIHandlers) SettingsHandler(w http.ResponseWriter, r *http.Request) {
	session := h.authManager.IsAuthenticated(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req config.Config
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate and set default paths if empty
	if req.NotesPath == "" {
		req.NotesPath = config.GetDefaultDataPath()
	}
	if req.PasswordHashPath == "" {
		req.PasswordHashPath = config.GetDefaultPasswordHashPath()
	}

	// Ensure directories exist before saving config
	if err := os.MkdirAll(req.NotesPath, 0755); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create notes directory: %v", err), http.StatusBadRequest)
		return
	}

	passwordDir := filepath.Dir(req.PasswordHashPath)
	if err := os.MkdirAll(passwordDir, 0755); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create password hash directory: %v", err), http.StatusBadRequest)
		return
	}

	// Update global config
	h.config.NotesPath = req.NotesPath
	h.config.PasswordHashPath = req.PasswordHashPath

	// Save config to file
	if err := h.config.Save(); err != nil {
		http.Error(w, "Failed to save configuration", http.StatusInternalServerError)
		return
	}

	// Note: In a real implementation, you might want to restart the store
	// with the new path, but that would require more complex coordination

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Settings saved successfully",
	}); err != nil {
		fmt.Printf("[ERROR] encoding settings response: %v\n", err)
	}
}

// SyncHandler forces a sync from disk
func (h *APIHandlers) SyncHandler(w http.ResponseWriter, r *http.Request) {
	session := h.authManager.IsAuthenticated(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.store.RefreshFromDisk(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to sync from disk: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Successfully synced from disk",
	}); err != nil {
		fmt.Printf("[ERROR] encoding sync response: %v\n", err)
	}
}

// ChangePasswordHandler changes the user's password and re-encrypts all notes
func (h *APIHandlers) ChangePasswordHandler(w http.ResponseWriter, r *http.Request) {
	// Authenticate user session
	session := h.authManager.IsAuthenticated(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Verify old password
	verified := h.authManager.VerifyPassword(req.OldPassword)
	if !verified {
		http.Error(w, "Old password is incorrect", http.StatusUnauthorized)
		return
	}

	// Derive old and new keys using enhanced method
	configPath := filepath.Join(h.store.GetDataDir(), ".keyconfig.json")
	oldKey, err := crypto.DeriveKeyEnhanced(req.OldPassword, configPath)
	if err != nil {
		http.Error(w, "Failed to derive old key: "+err.Error(), http.StatusInternalServerError)
		return
	}

	newKey, err := crypto.DeriveKeyEnhanced(req.NewPassword, configPath)
	if err != nil {
		http.Error(w, "Failed to derive new key: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Backup notes before changing password
	// backupPath, err := storage.BackupNotes(h.config.NotesPath, "")
	// if err != nil {
	// 	http.Error(w, "Failed to create backup: "+err.Error(), http.StatusInternalServerError)
	// 	return
	// }

	// Re-encrypt all notes from disk
	noteFiles, err := filepath.Glob(filepath.Join(h.config.NotesPath, "*.json"))
	if err != nil {
		http.Error(w, "Failed to list note files", http.StatusInternalServerError)
		return
	}
	var corruptedNotes []string
	for _, file := range noteFiles {
		data, err := os.ReadFile(file)
		if err != nil {
			corruptedNotes = append(corruptedNotes, filepath.Base(file))
			if err2 := h.store.MoveNoteToCorrupted(strings.TrimSuffix(filepath.Base(file), ".json")); err2 != nil {
				fmt.Printf("[ERROR] moving note to corrupted: %v\n", err2)
			}
			fmt.Printf("[ERROR] moving note to corrupted: %v\n", err)
			continue
		}
		var encryptedNote models.EncryptedNote
		if err := json.Unmarshal(data, &encryptedNote); err != nil {
			corruptedNotes = append(corruptedNotes, filepath.Base(file))
			if err2 := h.store.MoveNoteToCorrupted(strings.TrimSuffix(filepath.Base(file), ".json")); err2 != nil {
				fmt.Printf("[ERROR] moving note to corrupted: %v\n", err2)
			}
			fmt.Printf("[ERROR] moving note to corrupted: %v\n", err)
			continue
		}
		decryptedContent, err := crypto.Decrypt(encryptedNote.EncryptedData, oldKey)
		if err != nil {
			corruptedNotes = append(corruptedNotes, encryptedNote.ID)
			if err2 := h.store.MoveNoteToCorrupted(encryptedNote.ID); err2 != nil {
				fmt.Printf("[ERROR] moving note to corrupted: %v\n", err2)
			}
			fmt.Printf("[ERROR] moving note to corrupted: %v\n", err)
			continue
		}
		note := &models.Note{
			ID:        encryptedNote.ID,
			Content:   decryptedContent, // <-- use plaintext here
			CreatedAt: encryptedNote.CreatedAt,
			UpdatedAt: encryptedNote.UpdatedAt,
		}
		if err := h.store.SaveNoteDirect(note, newKey); err != nil {
			http.Error(w, "Failed to save note: "+note.ID, http.StatusInternalServerError)
			return
		}
	}

	// Store new password hash
	if err := h.authManager.StorePasswordHash(req.NewPassword); err != nil {
		http.Error(w, "Failed to update password hash", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if len(corruptedNotes) > 0 {
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"success":         true,
			"message":         fmt.Sprintf("Password changed and notes re-encrypted successfully. %d corrupted note(s) were moved to the 'corrupted' folder.", len(corruptedNotes)),
			"corrupted_notes": corruptedNotes,
		}); err != nil {
			fmt.Printf("[ERROR] encoding password change response: %v\n", err)
		}
	} else {
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Password changed and notes re-encrypted successfully.",
		}); err != nil {
			fmt.Printf("[ERROR] encoding password change response: %v\n", err)
		}
	}
}

// BackupHandler triggers a manual backup of notes
func (h *APIHandlers) BackupHandler(w http.ResponseWriter, r *http.Request) {
	backupPath, err := storage.BackupNotes(h.config.NotesPath, "")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to create backup: " + err.Error(),
		})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Backup created successfully.",
		"path":    backupPath,
	})
}
