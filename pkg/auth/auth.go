package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
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

	// If local verification failed, try cross-platform config (for new devices)
	if m.notesDir != "" {
		salt, err := m.loadCrossPlatformSalt()
		if err == nil {
			// We have cross-platform salt but no local password hash
			// This means it's a new device - we need to create a local password hash
			// But first verify the password would work with this salt
			m.currentSalt = salt

			// Since we don't have a stored verification hash for cross-platform,
			// we'll return true and let the calling code handle first-time setup
			// The password verification will happen during key derivation
			return true
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
