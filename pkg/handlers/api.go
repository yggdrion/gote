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
	json.NewEncoder(w).Encode(notes)
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
	json.NewEncoder(w).Encode(note)
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
	json.NewEncoder(w).Encode(note)
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
	json.NewEncoder(w).Encode(note)
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
	json.NewEncoder(w).Encode(notes)
}

// GetSettingsHandler returns current configuration
func (h *APIHandlers) GetSettingsHandler(w http.ResponseWriter, r *http.Request) {
	session := h.authManager.IsAuthenticated(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Debug: Log the config being sent
	fmt.Printf("[DEBUG] Sending config: NotesPath=%s, PasswordHashPath=%s\n", h.config.NotesPath, h.config.PasswordHashPath)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.config)
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
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Settings saved successfully",
	})
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
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Successfully synced from disk",
	})
}

// ChangePasswordHandler changes the user's password and re-encrypts all notes
func (h *APIHandlers) ChangePasswordHandler(w http.ResponseWriter, r *http.Request) {
	// Authenticate user session
	session := h.authManager.IsAuthenticated(r)
	fmt.Printf("[DEBUG] Session: %+v\n", session)
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
	fmt.Printf("[DEBUG] ChangePassword request: old=%q new=%q\n", req.OldPassword, req.NewPassword)

	// Verify old password
	verified := h.authManager.VerifyPassword(req.OldPassword)
	fmt.Printf("[DEBUG] VerifyPassword result: %v\n", verified)
	if !verified {
		http.Error(w, "Old password is incorrect", http.StatusUnauthorized)
		return
	}

	// Derive old and new keys
	oldKey := crypto.DeriveKey(req.OldPassword)
	newKey := crypto.DeriveKey(req.NewPassword)

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
			h.store.MoveNoteToCorrupted(strings.TrimSuffix(filepath.Base(file), ".json"))
			continue
		}
		var encryptedNote models.EncryptedNote
		if err := json.Unmarshal(data, &encryptedNote); err != nil {
			corruptedNotes = append(corruptedNotes, filepath.Base(file))
			h.store.MoveNoteToCorrupted(strings.TrimSuffix(filepath.Base(file), ".json"))
			continue
		}
		decryptedContent, err := crypto.Decrypt(encryptedNote.EncryptedData, oldKey)
		if err != nil {
			corruptedNotes = append(corruptedNotes, encryptedNote.ID)
			h.store.MoveNoteToCorrupted(encryptedNote.ID)
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
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":         true,
			"message":         fmt.Sprintf("Password changed and notes re-encrypted successfully. %d corrupted note(s) were moved to the 'corrupted' folder.", len(corruptedNotes)),
			"corrupted_notes": corruptedNotes,
		})
	} else {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Password changed and notes re-encrypted successfully.",
		})
	}
}
