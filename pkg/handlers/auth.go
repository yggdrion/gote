package handlers

import (
	"html/template"
	"log"
	"net/http"
	"path/filepath"

	"gote/pkg/crypto"
)

// AuthHandlers contains authentication-related handlers
type AuthHandlers struct {
	authManager AuthManagerFull
	store       NoteStore
}

// AuthManagerFull interface with full authentication methods
type AuthManagerFull interface {
	AuthManager
	IsFirstTimeSetup() bool
	StorePasswordHash(password string) error
	VerifyPassword(password string) bool
	CreateSession(key []byte) string
	DeleteSession(sessionID string)
}

// NoteStore interface for note operations
type NoteStore interface {
	LoadNotes(key []byte) error
	GetDataDir() string
}

// NewAuthHandlers creates new auth handlers
func NewAuthHandlers(authManager AuthManagerFull, store NoteStore) *AuthHandlers {
	return &AuthHandlers{
		authManager: authManager,
		store:       store,
	}
}

// LoginHandler serves the login page
func (h *AuthHandlers) LoginHandler(w http.ResponseWriter, r *http.Request) {
	isFirstTime := h.authManager.IsFirstTimeSetup()

	tmpl, err := template.ParseFiles("./static/login.html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		log.Printf("Template parsing error: %v", err)
		return
	}

	data := struct {
		Error       string
		IsFirstTime bool
	}{
		Error:       r.URL.Query().Get("error"),
		IsFirstTime: isFirstTime,
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Template execution error", http.StatusInternalServerError)
		log.Printf("Template execution error: %v", err)
	}
}

// AuthHandler handles authentication requests
func (h *AuthHandlers) AuthHandler(w http.ResponseWriter, r *http.Request) {
	password := r.FormValue("password")
	if password == "" {
		http.Redirect(w, r, "/login?error=Password required", http.StatusSeeOther)
		return
	}

	// Handle first-time setup
	if h.authManager.IsFirstTimeSetup() {
		confirmPassword := r.FormValue("confirm_password")
		if confirmPassword == "" {
			http.Redirect(w, r, "/login?error=Please confirm your password", http.StatusSeeOther)
			return
		}

		if password != confirmPassword {
			http.Redirect(w, r, "/login?error=Passwords do not match", http.StatusSeeOther)
			return
		}

		if len(password) < 6 {
			http.Redirect(w, r, "/login?error=Password must be at least 6 characters", http.StatusSeeOther)
			return
		}

		// Store the password hash
		if err := h.authManager.StorePasswordHash(password); err != nil {
			http.Redirect(w, r, "/login?error=Failed to create password", http.StatusSeeOther)
			return
		}
	} else {
		// Verify existing password
		if !h.authManager.VerifyPassword(password) {
			http.Redirect(w, r, "/login?error=Invalid password", http.StatusSeeOther)
			return
		}
	}

	// Use enhanced key derivation that supports both legacy and PBKDF2 methods
	configPath := filepath.Join(h.store.GetDataDir(), ".keyconfig.json")
	key, err := crypto.DeriveKeyEnhanced(password, configPath)
	if err != nil {
		log.Printf("Error deriving key: %v", err)
		http.Redirect(w, r, "/login?error=Authentication failed", http.StatusSeeOther)
		return
	}

	// Try to load notes with this password
	if err := h.store.LoadNotes(key); err != nil {
		// For existing setup, this should not fail if password is correct
		if !h.authManager.IsFirstTimeSetup() {
			http.Redirect(w, r, "/login?error=Failed to decrypt notes", http.StatusSeeOther)
			return
		}
		// For first-time setup, it's expected that there are no notes to load
	}

	// Create session
	sessionID := h.authManager.CreateSession(key)

	// Set session cookie
	cookie := &http.Cookie{
		Name:     "session",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// LogoutHandler handles logout requests
func (h *AuthHandlers) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err == nil {
		h.authManager.DeleteSession(cookie.Value)
	}

	// Clear session cookie
	cookie = &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	}
	http.SetCookie(w, cookie)

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
