package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"

	"gote/pkg/crypto"
	"gote/pkg/errors"
)

// SecurePasswordConfig holds configuration for secure password storage
type SecurePasswordConfig struct {
	Method        string                      `json:"method"`                  // "legacy" or "pbkdf2"
	KeyDerivation *crypto.KeyDerivationConfig `json:"keyDerivation,omitempty"` // PBKDF2 config
	PasswordHash  string                      `json:"passwordHash"`            // Base64 encoded hash
}

// SecureManager provides enhanced authentication with backward compatibility
type SecureManager struct {
	*Manager   // Embed existing manager for backward compatibility
	configPath string
	deriver    *crypto.SecureKeyDeriver
}

// NewSecureManager creates a new secure authentication manager
func NewSecureManager(passwordHashPath string) *SecureManager {
	configPath := passwordHashPath + ".config"
	return &SecureManager{
		Manager:    NewManager(passwordHashPath),
		configPath: configPath,
		deriver:    crypto.NewSecureKeyDeriver(),
	}
}

// DetectPasswordMethod detects whether legacy or secure password storage is used
func (sm *SecureManager) DetectPasswordMethod() (string, error) {
	// Check if secure config exists
	if _, err := os.Stat(sm.configPath); err == nil {
		return "pbkdf2", nil
	}

	// Check if legacy password hash exists
	if _, err := os.Stat(sm.passwordHashPath); err == nil {
		return "legacy", nil
	}

	// No password set yet
	return "none", nil
}

// StorePasswordHashSecure stores a password using PBKDF2 with proper salt
func (sm *SecureManager) StorePasswordHashSecure(password string) error {
	// Generate PBKDF2 key and config
	derivedKey, keyConfig, err := sm.deriver.DeriveKeySecure(password)
	if err != nil {
		return err
	}

	// Create verification hash from the derived key
	verificationHash := sha256.Sum256(append(derivedKey, []byte("verification")...))

	// Create secure password config
	config := &SecurePasswordConfig{
		Method:        "pbkdf2",
		KeyDerivation: keyConfig,
		PasswordHash:  base64.StdEncoding.EncodeToString(verificationHash[:]),
	}

	// Ensure config directory exists
	configDir := filepath.Dir(sm.configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return errors.Wrap(err, errors.ErrTypeFileSystem, "DIR_CREATE_FAILED",
			"failed to create config directory").
			WithUserMessage("Unable to create password configuration directory")
	}

	// Save secure config
	configData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return errors.Wrap(err, errors.ErrTypeConfig, "CONFIG_MARSHAL_FAILED",
			"failed to marshal password config").
			WithUserMessage("Unable to format password configuration")
	}

	if err := os.WriteFile(sm.configPath, configData, 0600); err != nil {
		return errors.Wrap(err, errors.ErrTypeFileSystem, "CONFIG_WRITE_FAILED",
			"failed to write password config").
			WithUserMessage("Unable to save password configuration")
	}

	// Remove legacy password hash file if it exists
	if _, err := os.Stat(sm.passwordHashPath); err == nil {
		os.Remove(sm.passwordHashPath)
	}

	return nil
}

// VerifyPasswordSecure verifies a password using the appropriate method
func (sm *SecureManager) VerifyPasswordSecure(password string) ([]byte, bool) {
	method, err := sm.DetectPasswordMethod()
	if err != nil {
		return nil, false
	}

	switch method {
	case "pbkdf2":
		return sm.verifyPBKDF2Password(password)
	case "legacy":
		return sm.verifyLegacyPassword(password)
	default:
		return nil, false
	}
}

// verifyPBKDF2Password verifies password using PBKDF2
func (sm *SecureManager) verifyPBKDF2Password(password string) ([]byte, bool) {
	// Read secure config
	configData, err := os.ReadFile(sm.configPath)
	if err != nil {
		return nil, false
	}

	var config SecurePasswordConfig
	if err := json.Unmarshal(configData, &config); err != nil {
		return nil, false
	}

	// Derive key using stored configuration
	derivedKey, err := sm.deriver.DeriveKeyWithConfig(password, config.KeyDerivation)
	if err != nil {
		return nil, false
	}

	// Create verification hash and compare
	verificationHash := sha256.Sum256(append(derivedKey, []byte("verification")...))
	expectedHash, err := base64.StdEncoding.DecodeString(config.PasswordHash)
	if err != nil {
		return nil, false
	}

	// Constant time comparison for security
	if len(verificationHash) != len(expectedHash) {
		return nil, false
	}

	match := true
	for i := 0; i < len(verificationHash); i++ {
		if verificationHash[i] != expectedHash[i] {
			match = false
		}
	}

	if match {
		return derivedKey, true
	}
	return nil, false
}

// verifyLegacyPassword verifies password using legacy SHA-256
func (sm *SecureManager) verifyLegacyPassword(password string) ([]byte, bool) {
	// Use legacy verification
	if sm.Manager.VerifyPassword(password) {
		// Return legacy key
		return crypto.DeriveKey(password), true
	}
	return nil, false
}

// MigrateToSecure migrates from legacy password storage to secure PBKDF2
func (sm *SecureManager) MigrateToSecure(password string) error {
	// First verify the password works with legacy method
	if !sm.Manager.VerifyPassword(password) {
		return errors.ErrInvalidPassword
	}

	// Store using secure method
	if err := sm.StorePasswordHashSecure(password); err != nil {
		return err
	}

	return nil
}

// GetEncryptionKey gets the appropriate encryption key for the password
func (sm *SecureManager) GetEncryptionKey(password string) ([]byte, bool) {
	return sm.VerifyPasswordSecure(password)
}

// IsSecureMethod checks if the current password storage uses secure method
func (sm *SecureManager) IsSecureMethod() bool {
	method, err := sm.DetectPasswordMethod()
	if err != nil {
		return false
	}
	return method == "pbkdf2"
}

// RemovePasswordHashSecure removes both legacy and secure password storage
func (sm *SecureManager) RemovePasswordHashSecure() error {
	var lastErr error

	// Remove legacy password hash
	if err := sm.Manager.RemovePasswordHash(); err != nil {
		lastErr = err
	}

	// Remove secure config
	if err := os.Remove(sm.configPath); err != nil && !os.IsNotExist(err) {
		lastErr = err
	}

	if lastErr != nil {
		return errors.Wrap(lastErr, errors.ErrTypeFileSystem, "PASSWORD_REMOVE_FAILED",
			"failed to remove password files").
			WithUserMessage("Unable to remove password configuration")
	}

	return nil
}
