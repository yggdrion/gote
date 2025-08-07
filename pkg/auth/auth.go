package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// CrossPlatformConfig stores the salt in the synced notes directory for cross-platform compatibility
type CrossPlatformConfig struct {
	Salt      string `json:"salt"`
	CreatedAt string `json:"createdAt"`
	Version   string `json:"version"`
}

// Manager handles authentication and session management
type Manager struct {
	sessions         map[string]*models.Session
	sessionsMutex    sync.RWMutex
	passwordHashPath string
	currentSalt      []byte // Store the current salt for key derivation
	notesDir         string // Store notes directory for cross-platform config
}

// NewManager creates a new authentication manager
func NewManager(passwordHashPath string) *Manager {
	return &Manager{
		sessions:         make(map[string]*models.Session),
		passwordHashPath: passwordHashPath,
	}
}

// NewManagerWithNotesDir creates a new authentication manager with notes directory for cross-platform support
func NewManagerWithNotesDir(passwordHashPath, notesDir string) *Manager {
	return &Manager{
		sessions:         make(map[string]*models.Session),
		passwordHashPath: passwordHashPath,
		notesDir:         notesDir,
	}
}

// IsFirstTimeSetup checks if this is the first time setup (no password hash exists AND no cross-platform config exists)
func (m *Manager) IsFirstTimeSetup() bool {
	// Check if local password hash exists
	_, err := os.Stat(m.passwordHashPath)
	localExists := !os.IsNotExist(err)

	// If local exists, not first time
	if localExists {
		return false
	}

	// Check if cross-platform config exists (if notes directory is set)
	if m.notesDir != "" {
		configPath := filepath.Join(m.notesDir, ".gote_config.json")
		_, err := os.Stat(configPath)
		crossPlatformExists := !os.IsNotExist(err)

		// If cross-platform config exists, not first time setup - just need to sync locally
		if crossPlatformExists {
			return false
		}
	}

	// Neither local nor cross-platform config exists - truly first time
	return true
} // StorePasswordHash stores a hash of the password with salt for verification
func (m *Manager) StorePasswordHash(password string) error {
	// Check if cross-platform config already exists
	if m.notesDir != "" {
		configPath := filepath.Join(m.notesDir, ".gote_config.json")
		if _, err := os.Stat(configPath); err == nil {
			if salt, err := m.loadCrossPlatformSalt(); err == nil {
				// Use the existing cross-platform salt
				m.currentSalt = salt

				// Create verification hash using the existing salt
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

				// Save local password hash with existing salt
				return os.WriteFile(m.passwordHashPath, data, 0600)
			}
		}
	}

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

	// Save local password hash
	if err := os.WriteFile(m.passwordHashPath, data, 0600); err != nil {
		return err
	}

	// Save cross-platform config if notes directory is set
	if m.notesDir != "" {
		if err := m.saveCrossPlatformSalt(salt); err != nil {
			// Log warning but don't fail - local storage still works
			fmt.Printf("Warning: Could not save cross-platform config: %v\n", err)
		}
	}

	return nil
}

// VerifyPassword verifies the provided password against the stored hash
func (m *Manager) VerifyPassword(password string) bool {
	if m.IsFirstTimeSetup() {
		return false
	}

	// Try local password hash first
	data, err := os.ReadFile(m.passwordHashPath)
	if err == nil {
		var passwordData PasswordData
		if err := json.Unmarshal(data, &passwordData); err == nil {
			// Decode the stored salt
			salt, err := base64.StdEncoding.DecodeString(passwordData.Salt)
			if err == nil {
				// Store the salt for key derivation
				m.currentSalt = salt

				// Create verification hash using the same salt
				verificationKey := crypto.DeriveKey(password+"verification", salt)
				computedHash := base64.StdEncoding.EncodeToString(verificationKey)

				if computedHash == passwordData.Hash {
					return true
				}
			}
		}
	}

	// If local verification failed, try cross-platform setup
	// There's only one password for all notes across devices
	if m.notesDir != "" {
		salt, err := m.loadCrossPlatformSalt()
		if err == nil {
			// We have cross-platform salt - verify password with this salt
			// and create local password hash if verification succeeds
			m.currentSalt = salt

			// Verify the password can decrypt existing notes (if any exist)
			if m.verifyPasswordWithCrossPlatformData(password, salt) {
				// Password is correct - create local password hash for faster future logins
				if err := m.createLocalPasswordHashFromCrossPlatform(password, salt); err != nil {
					// Log warning but don't fail - cross-platform verification already passed
					fmt.Printf("Warning: Could not create local password hash: %v\n", err)
				}
				return true
			}
		}
	}

	return false
} // CreateSession creates a new session for an authenticated user
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
func (m *Manager) GetSession(sessionID string) (*models.Session, bool) {
	m.sessionsMutex.RLock()
	session, exists := m.sessions[sessionID]
	m.sessionsMutex.RUnlock()

	if !exists {
		return nil, false
	}

	// Check if session has expired
	if time.Now().After(session.ExpiresAt) {
		// Session expired, clean it up
		m.DeleteSession(sessionID)
		return nil, false
	}

	return session, true
}

// ValidateSession checks if a session is valid and updates expiry
func (m *Manager) ValidateSession(sessionID string) bool {
	m.sessionsMutex.Lock()
	defer m.sessionsMutex.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return false
	}

	// Check if session has expired
	if time.Now().After(session.ExpiresAt) {
		// Session expired, clean it up
		delete(m.sessions, sessionID)
		return false
	}

	// Update expiry time (extend session)
	session.ExpiresAt = time.Now().Add(SessionTimeout)
	return true
}

// CleanupExpiredSessions removes all expired sessions
func (m *Manager) CleanupExpiredSessions() {
	m.sessionsMutex.Lock()
	defer m.sessionsMutex.Unlock()

	now := time.Now()
	for sessionID, session := range m.sessions {
		if now.After(session.ExpiresAt) {
			delete(m.sessions, sessionID)
		}
	}
}

// DeleteSession removes a session (logout)
func (m *Manager) DeleteSession(sessionID string) {
	m.sessionsMutex.Lock()
	delete(m.sessions, sessionID)
	m.sessionsMutex.Unlock()
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
		// Try loading salt from cross-platform config first (for multi-device support)
		if m.notesDir != "" {
			salt, err := m.loadCrossPlatformSalt()
			if err == nil {
				m.currentSalt = salt
				return crypto.DeriveKey(password, m.currentSalt), nil
			}
		}

		// Load salt from local password file
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

		// Create cross-platform config if it doesn't exist and notes directory is set
		if m.notesDir != "" {
			configPath := filepath.Join(m.notesDir, ".gote_config.json")
			if _, err := os.Stat(configPath); os.IsNotExist(err) {
				if err := m.saveCrossPlatformSalt(salt); err != nil {
					// Log warning but don't fail
					fmt.Printf("Warning: Could not create cross-platform config: %v\n", err)
				}
			}
		}
	}

	return crypto.DeriveKey(password, m.currentSalt), nil
}

// loadCrossPlatformSalt loads salt from the notes directory for cross-platform compatibility
func (m *Manager) loadCrossPlatformSalt() ([]byte, error) {
	if m.notesDir == "" {
		return nil, fmt.Errorf("notes directory not set")
	}

	configPath := filepath.Join(m.notesDir, ".gote_config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("cross-platform config not found: %v", err)
	}

	var config CrossPlatformConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse cross-platform config: %v", err)
	}

	salt, err := base64.StdEncoding.DecodeString(config.Salt)
	if err != nil {
		return nil, fmt.Errorf("failed to decode salt: %v", err)
	}

	return salt, nil
}

// saveCrossPlatformSalt saves salt to the notes directory for cross-platform compatibility
func (m *Manager) saveCrossPlatformSalt(salt []byte) error {
	if m.notesDir == "" {
		return fmt.Errorf("notes directory not set")
	}

	config := CrossPlatformConfig{
		Salt:      base64.StdEncoding.EncodeToString(salt),
		CreatedAt: time.Now().Format(time.RFC3339),
		Version:   "1.0",
	}

	configPath := filepath.Join(m.notesDir, ".gote_config.json")
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	return os.WriteFile(configPath, data, 0600)
}

// SyncFromCrossPlatform creates a local password hash from cross-platform config
// This is used when setting up Gote on a new device that has access to synced notes
func (m *Manager) SyncFromCrossPlatform(password string) error {
	if m.notesDir == "" {
		return fmt.Errorf("notes directory not set")
	}

	// Load salt from cross-platform config
	salt, err := m.loadCrossPlatformSalt()
	if err != nil {
		return fmt.Errorf("failed to load cross-platform salt: %v", err)
	}

	// Store the salt for key derivation
	m.currentSalt = salt

	// Create local password hash using the shared salt
	return m.createLocalPasswordHashFromCrossPlatform(password, salt)
}

// verifyPasswordWithCrossPlatformData verifies a password by testing if it can decrypt existing notes
func (m *Manager) verifyPasswordWithCrossPlatformData(password string, salt []byte) bool {
	if m.notesDir == "" {
		return false
	}

	// Derive encryption key from password and salt
	key := crypto.DeriveKey(password, salt)

	// Look for encrypted note files in the notes directory
	files, err := os.ReadDir(m.notesDir)
	if err != nil {
		return false
	}

	// Try to decrypt at least one .json file to verify the password
	jsonFilesFound := 0
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") && file.Name() != ".gote_config.json" {
			jsonFilesFound++
			filePath := filepath.Join(m.notesDir, file.Name())

			// Try to read and decrypt the file
			data, err := os.ReadFile(filePath)
			if err != nil {
				continue
			}

			// Try to parse as encrypted note
			var encryptedNote struct {
				EncryptedData string `json:"encryptedData"`
			}

			if err := json.Unmarshal(data, &encryptedNote); err != nil {
				continue
			}

			// Attempt to decrypt - if this succeeds, the password is correct
			_, err = crypto.Decrypt(encryptedNote.EncryptedData, key)
			if err == nil {
				// Successfully decrypted at least one note - password is valid
				return true
			}
		}
	}

	// No files could be decrypted - this means either:
	// 1. Wrong password (most likely)
	// 2. No encrypted notes exist yet (unlikely in cross-platform setup)
	// For security, we should NEVER allow a password that can't decrypt existing data
	// if such data exists. Only allow if there are truly NO encrypted note files.
	if jsonFilesFound > 0 {
		// Found encrypted notes but none could be decrypted - definitely wrong password
		return false
	}

	// Only if there are truly no encrypted notes, check if this might be a fresh setup
	configPath := filepath.Join(m.notesDir, ".gote_config.json")
	if _, err := os.Stat(configPath); err == nil {
		// Cross-platform config exists but no encrypted notes - this might be a fresh setup
		// We'll allow the password in this specific case only
		return true
	}

	return false
}

// createLocalPasswordHashFromCrossPlatform creates a local password hash using the cross-platform salt
func (m *Manager) createLocalPasswordHashFromCrossPlatform(password string, salt []byte) error {
	// Create verification hash using the cross-platform salt
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

	// Save local password hash
	return os.WriteFile(m.passwordHashPath, data, 0600)
}
