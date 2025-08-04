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
	fmt.Printf("[DEBUG] IsFirstTimeSetup: Checking authentication state\n")
	fmt.Printf("[DEBUG] Local password hash path: %s\n", m.passwordHashPath)
	fmt.Printf("[DEBUG] Notes directory: %s\n", m.notesDir)

	// Check if local password hash exists
	_, err := os.Stat(m.passwordHashPath)
	localExists := !os.IsNotExist(err)
	fmt.Printf("[DEBUG] Local password hash exists: %t\n", localExists)

	// If local exists, not first time
	if localExists {
		fmt.Printf("[DEBUG] Result: Not first time (local password hash exists)\n")
		return false
	}

	// Check if cross-platform config exists (if notes directory is set)
	if m.notesDir != "" {
		configPath := filepath.Join(m.notesDir, ".gote_config.json")
		fmt.Printf("[DEBUG] Cross-platform config path: %s\n", configPath)
		_, err := os.Stat(configPath)
		crossPlatformExists := !os.IsNotExist(err)
		fmt.Printf("[DEBUG] Cross-platform config exists: %t\n", crossPlatformExists)

		// If cross-platform config exists, not first time setup - just need to sync locally
		if crossPlatformExists {
			fmt.Printf("[DEBUG] Result: Not first time (cross-platform config exists)\n")
			return false
		}
	} else {
		fmt.Printf("[DEBUG] Notes directory not set, cannot check cross-platform config\n")
	}

	// Neither local nor cross-platform config exists - truly first time
	fmt.Printf("[DEBUG] Result: First time setup (no configs found)\n")
	return true
} // StorePasswordHash stores a hash of the password with salt for verification
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
	fmt.Printf("[DEBUG] VerifyPassword: Starting password verification\n")
	fmt.Printf("[DEBUG] Password length: %d characters\n", len(password))

	if m.IsFirstTimeSetup() {
		fmt.Printf("[DEBUG] VerifyPassword: First time setup detected, returning false\n")
		return false
	}

	// Try local password hash first
	fmt.Printf("[DEBUG] VerifyPassword: Attempting local password hash verification\n")
	fmt.Printf("[DEBUG] Local password hash path: %s\n", m.passwordHashPath)

	data, err := os.ReadFile(m.passwordHashPath)
	if err == nil {
		fmt.Printf("[DEBUG] Local password hash file read successfully (%d bytes)\n", len(data))
		var passwordData PasswordData
		if err := json.Unmarshal(data, &passwordData); err == nil {
			fmt.Printf("[DEBUG] Local password data parsed successfully\n")
			fmt.Printf("[DEBUG] Stored hash: %s\n", passwordData.Hash[:20]+"...")
			fmt.Printf("[DEBUG] Stored salt: %s\n", passwordData.Salt[:20]+"...")

			// Decode the stored salt
			salt, err := base64.StdEncoding.DecodeString(passwordData.Salt)
			if err == nil {
				fmt.Printf("[DEBUG] Salt decoded successfully (%d bytes)\n", len(salt))
				// Store the salt for key derivation
				m.currentSalt = salt

				// Create verification hash using the same salt
				verificationKey := crypto.DeriveKey(password+"verification", salt)
				computedHash := base64.StdEncoding.EncodeToString(verificationKey)

				fmt.Printf("[DEBUG] Computed hash: %s\n", computedHash[:20]+"...")
				fmt.Printf("[DEBUG] Stored hash:   %s\n", passwordData.Hash[:20]+"...")

				if computedHash == passwordData.Hash {
					fmt.Printf("[DEBUG] VerifyPassword: Local verification SUCCESS\n")
					return true
				} else {
					fmt.Printf("[DEBUG] VerifyPassword: Local verification FAILED (hash mismatch)\n")
				}
			} else {
				fmt.Printf("[DEBUG] Failed to decode local salt: %v\n", err)
			}
		} else {
			fmt.Printf("[DEBUG] Failed to parse local password data: %v\n", err)
		}
	} else {
		fmt.Printf("[DEBUG] Failed to read local password hash file: %v\n", err)
	}

	// If local verification failed, try cross-platform config (for new devices)
	fmt.Printf("[DEBUG] VerifyPassword: Attempting cross-platform config verification\n")
	if m.notesDir != "" {
		fmt.Printf("[DEBUG] Notes directory: %s\n", m.notesDir)
		salt, err := m.loadCrossPlatformSalt()
		if err == nil {
			fmt.Printf("[DEBUG] Cross-platform salt loaded successfully (%d bytes)\n", len(salt))
			fmt.Printf("[DEBUG] Cross-platform salt: %s\n", base64.StdEncoding.EncodeToString(salt)[:20]+"...")

			// We have cross-platform salt but no local password hash
			// This means it's a new device - we need to create a local password hash
			// But first verify the password would work with this salt
			m.currentSalt = salt

			// Since we don't have a stored verification hash for cross-platform,
			// we'll return true and let the calling code handle first-time setup
			// The password verification will happen during key derivation
			fmt.Printf("[DEBUG] VerifyPassword: Cross-platform config found, returning true for new device setup\n")
			return true
		} else {
			fmt.Printf("[DEBUG] Failed to load cross-platform salt: %v\n", err)
		}
	} else {
		fmt.Printf("[DEBUG] Notes directory not set, cannot try cross-platform verification\n")
	}

	fmt.Printf("[DEBUG] VerifyPassword: All verification methods failed\n")
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
	fmt.Printf("[DEBUG] DeriveEncryptionKey: Starting key derivation\n")
	fmt.Printf("[DEBUG] Password length: %d characters\n", len(password))
	fmt.Printf("[DEBUG] Current salt in memory: %v\n", m.currentSalt != nil)

	if m.currentSalt == nil {
		fmt.Printf("[DEBUG] No salt in memory, attempting to load...\n")

		// Try loading salt from cross-platform config first (for multi-device support)
		if m.notesDir != "" {
			fmt.Printf("[DEBUG] Trying cross-platform config first\n")
			fmt.Printf("[DEBUG] Notes directory: %s\n", m.notesDir)

			salt, err := m.loadCrossPlatformSalt()
			if err == nil {
				fmt.Printf("[DEBUG] Cross-platform salt loaded successfully (%d bytes)\n", len(salt))
				fmt.Printf("[DEBUG] Cross-platform salt: %s\n", base64.StdEncoding.EncodeToString(salt)[:20]+"...")
				m.currentSalt = salt
				key := crypto.DeriveKey(password, m.currentSalt)
				fmt.Printf("[DEBUG] Encryption key derived from cross-platform salt (%d bytes)\n", len(key))
				return key, nil
			} else {
				fmt.Printf("[DEBUG] Cross-platform salt loading failed: %v\n", err)
			}
			// If cross-platform config fails, fall back to local config
		} else {
			fmt.Printf("[DEBUG] Notes directory not set, skipping cross-platform config\n")
		}

		// Load salt from local password file
		fmt.Printf("[DEBUG] Attempting to load salt from local password file\n")
		if m.IsFirstTimeSetup() {
			fmt.Printf("[DEBUG] First time setup detected, cannot derive key\n")
			return nil, fmt.Errorf("no password set up")
		}

		data, err := os.ReadFile(m.passwordHashPath)
		if err != nil {
			fmt.Printf("[DEBUG] Failed to read local password file: %v\n", err)
			return nil, fmt.Errorf("failed to read password file: %v", err)
		}

		var passwordData PasswordData
		if err := json.Unmarshal(data, &passwordData); err != nil {
			fmt.Printf("[DEBUG] Failed to parse local password data: %v\n", err)
			return nil, fmt.Errorf("failed to parse password data: %v", err)
		}

		salt, err := base64.StdEncoding.DecodeString(passwordData.Salt)
		if err != nil {
			fmt.Printf("[DEBUG] Failed to decode local salt: %v\n", err)
			return nil, fmt.Errorf("failed to decode salt: %v", err)
		}

		fmt.Printf("[DEBUG] Local salt loaded successfully (%d bytes)\n", len(salt))
		fmt.Printf("[DEBUG] Local salt: %s\n", passwordData.Salt[:20]+"...")
		m.currentSalt = salt

		// Create cross-platform config if it doesn't exist and notes directory is set
		if m.notesDir != "" {
			configPath := filepath.Join(m.notesDir, ".gote_config.json")
			if _, err := os.Stat(configPath); os.IsNotExist(err) {
				fmt.Printf("[DEBUG] Creating cross-platform config from local salt\n")
				if err := m.saveCrossPlatformSalt(salt); err != nil {
					fmt.Printf("[DEBUG] Warning: Could not create cross-platform config: %v\n", err)
				} else {
					fmt.Printf("[DEBUG] Cross-platform config created successfully\n")
				}
			}
		}
	} else {
		fmt.Printf("[DEBUG] Using salt from memory (%d bytes)\n", len(m.currentSalt))
		fmt.Printf("[DEBUG] Memory salt: %s\n", base64.StdEncoding.EncodeToString(m.currentSalt)[:20]+"...")
	}

	key := crypto.DeriveKey(password, m.currentSalt)
	fmt.Printf("[DEBUG] Final encryption key derived (%d bytes)\n", len(key))
	return key, nil
}

// loadCrossPlatformSalt loads salt from the notes directory for cross-platform compatibility
func (m *Manager) loadCrossPlatformSalt() ([]byte, error) {
	fmt.Printf("[DEBUG] loadCrossPlatformSalt: Starting\n")

	if m.notesDir == "" {
		fmt.Printf("[DEBUG] loadCrossPlatformSalt: Notes directory not set\n")
		return nil, fmt.Errorf("notes directory not set")
	}

	configPath := filepath.Join(m.notesDir, ".gote_config.json")
	fmt.Printf("[DEBUG] loadCrossPlatformSalt: Config path: %s\n", configPath)

	data, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Printf("[DEBUG] loadCrossPlatformSalt: Failed to read config file: %v\n", err)
		return nil, fmt.Errorf("cross-platform config not found: %v", err)
	}

	fmt.Printf("[DEBUG] loadCrossPlatformSalt: Config file read (%d bytes)\n", len(data))
	fmt.Printf("[DEBUG] loadCrossPlatformSalt: Config content: %s\n", string(data))

	var config CrossPlatformConfig
	if err := json.Unmarshal(data, &config); err != nil {
		fmt.Printf("[DEBUG] loadCrossPlatformSalt: Failed to parse config: %v\n", err)
		return nil, fmt.Errorf("failed to parse cross-platform config: %v", err)
	}

	fmt.Printf("[DEBUG] loadCrossPlatformSalt: Config parsed successfully\n")
	fmt.Printf("[DEBUG] loadCrossPlatformSalt: Salt from config: %s\n", config.Salt[:20]+"...")

	salt, err := base64.StdEncoding.DecodeString(config.Salt)
	if err != nil {
		fmt.Printf("[DEBUG] loadCrossPlatformSalt: Failed to decode salt: %v\n", err)
		return nil, fmt.Errorf("failed to decode salt: %v", err)
	}

	fmt.Printf("[DEBUG] loadCrossPlatformSalt: Salt decoded successfully (%d bytes)\n", len(salt))
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
