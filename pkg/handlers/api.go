package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"

	"gote/pkg/config"
	"gote/pkg/storage"
)

// APIHandlers contains API endpoint handlers
type APIHandlers struct {
	store       *storage.NoteStore
	authManager AuthManager
	config      *config.Config
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
