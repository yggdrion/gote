package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gote/pkg/crypto"
	"gote/pkg/models"
	"gote/pkg/utils"
)

const SessionTimeout = 30 * time.Minute

// PasswordData stores password hash and salt
type PasswordData struct {
	Hash string `json:"hash"`
	Salt string `json:"salt"`
}

// Manager handles authentication and session management
type Manager struct {
	sessions         map[string]*models.Session
	sessionsMutex    sync.RWMutex
	passwordHashPath string
	currentSalt      []byte // Store the current salt for key derivation
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

// StorePasswordHash stores a hash of the password with salt for verification
func (m *Manager) StorePasswordHash(password string) error {
	// Generate salt for password verification
	salt, err := crypto.GenerateSalt()
	if err != nil {
		return fmt.Errorf("failed to generate salt: %v", err)
	}

	// Store the salt for key derivation
	m.currentSalt = salt

	// Create verification hash using PBKDF2
	verificationKey := crypto.DeriveKey(password+"verification", salt)

	passwordData := PasswordData{
		Hash: base64.StdEncoding.EncodeToString(verificationKey),
		Salt: base64.StdEncoding.EncodeToString(salt),
	}

	// Ensure password hash directory exists
	hashDir := filepath.Dir(m.passwordHashPath)
	if err := os.MkdirAll(hashDir, 0755); err != nil {
		return err
	}

	data, err := json.Marshal(passwordData)
	if err != nil {
		return fmt.Errorf("failed to marshal password data: %v", err)
	}

	return os.WriteFile(m.passwordHashPath, data, 0600)
}

// VerifyPassword verifies the provided password against the stored hash
func (m *Manager) VerifyPassword(password string) bool {
	if m.IsFirstTimeSetup() {
		return false
	}

	data, err := os.ReadFile(m.passwordHashPath)
	if err != nil {
		return false
	}

	var passwordData PasswordData
	if err := json.Unmarshal(data, &passwordData); err != nil {
		return false
	}

	// Decode the stored salt
	salt, err := base64.StdEncoding.DecodeString(passwordData.Salt)
	if err != nil {
		return false
	}

	// Store the salt for key derivation
	m.currentSalt = salt

	// Create verification hash using the same salt
	verificationKey := crypto.DeriveKey(password+"verification", salt)
	computedHash := base64.StdEncoding.EncodeToString(verificationKey)

	return computedHash == passwordData.Hash
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

// DeriveEncryptionKey derives the encryption key from password using the stored salt
func (m *Manager) DeriveEncryptionKey(password string) ([]byte, error) {
	if m.currentSalt == nil {
		// Load salt from password file if not in memory
		if m.IsFirstTimeSetup() {
			return nil, fmt.Errorf("no password set up")
		}

		data, err := os.ReadFile(m.passwordHashPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read password file: %v", err)
		}

		var passwordData PasswordData
		if err := json.Unmarshal(data, &passwordData); err != nil {
			return nil, fmt.Errorf("failed to parse password data: %v", err)
		}

		salt, err := base64.StdEncoding.DecodeString(passwordData.Salt)
		if err != nil {
			return nil, fmt.Errorf("failed to decode salt: %v", err)
		}

		m.currentSalt = salt
	}

	return crypto.DeriveKey(password, m.currentSalt), nil
}
