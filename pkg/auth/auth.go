package auth

import (
	"crypto/sha256"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gote/pkg/models"
	"gote/pkg/utils"
)

const SessionTimeout = 30 * time.Minute

// Manager handles authentication and session management
type Manager struct {
	sessions         map[string]*models.Session
	sessionsMutex    sync.RWMutex
	passwordHashPath string
}

// NewManager creates a new authentication manager
func NewManager(passwordHashPath string) *Manager {
	return &Manager{
		sessions:         make(map[string]*models.Session),
		passwordHashPath: passwordHashPath,
	}
}

// IsFirstTimeSetup checks if this is the first time setup (no password hash exists)
func (m *Manager) IsFirstTimeSetup() bool {
	_, err := os.Stat(m.passwordHashPath)
	return os.IsNotExist(err)
}

// StorePasswordHash stores a hash of the password for verification
func (m *Manager) StorePasswordHash(password string) error {
	verificationHash := sha256.Sum256([]byte(password + "verification"))

	// Ensure password hash directory exists
	hashDir := filepath.Dir(m.passwordHashPath)
	if err := os.MkdirAll(hashDir, 0755); err != nil {
		return err
	}

	return os.WriteFile(m.passwordHashPath, verificationHash[:], 0600)
}

// VerifyPassword verifies the provided password against the stored hash
func (m *Manager) VerifyPassword(password string) bool {
	if m.IsFirstTimeSetup() {
		return false
	}

	storedHash, err := os.ReadFile(m.passwordHashPath)
	if err != nil {
		return false
	}

	verificationHash := sha256.Sum256([]byte(password + "verification"))
	return string(storedHash) == string(verificationHash[:])
}

// CreateSession creates a new session for an authenticated user
func (m *Manager) CreateSession(key []byte) string {
	sessionID := utils.GenerateSessionID()

	m.sessionsMutex.Lock()
	m.sessions[sessionID] = &models.Session{
		Key:       key,
		ExpiresAt: time.Now().Add(SessionTimeout),
	}
	m.sessionsMutex.Unlock()

	return sessionID
}

// GetSession retrieves and validates a session
func (m *Manager) GetSession(r *http.Request) *models.Session {
	cookie, err := r.Cookie("session")
	if err != nil {
		return nil
	}

	m.sessionsMutex.RLock()
	session, exists := m.sessions[cookie.Value]
	m.sessionsMutex.RUnlock()

	if !exists || time.Now().After(session.ExpiresAt) {
		// Clean up expired session
		if exists {
			m.sessionsMutex.Lock()
			delete(m.sessions, cookie.Value)
			m.sessionsMutex.Unlock()
		}
		return nil
	}

	// Extend session
	m.sessionsMutex.Lock()
	session.ExpiresAt = time.Now().Add(SessionTimeout)
	m.sessionsMutex.Unlock()

	return session
}

// DeleteSession removes a session (logout)
func (m *Manager) DeleteSession(sessionID string) {
	m.sessionsMutex.Lock()
	delete(m.sessions, sessionID)
	m.sessionsMutex.Unlock()
}

// IsAuthenticated checks if the request has a valid session
func (m *Manager) IsAuthenticated(r *http.Request) *models.Session {
	return m.GetSession(r)
}

// RemovePasswordHash deletes the password hash file
func (m *Manager) RemovePasswordHash() error {
	if _, err := os.Stat(m.passwordHashPath); os.IsNotExist(err) {
		// File doesn't exist, nothing to remove
		return nil
	}
	return os.Remove(m.passwordHashPath)
}
