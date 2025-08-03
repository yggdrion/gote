package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"

	"gote/pkg/errors"

	"golang.org/x/crypto/pbkdf2"
)

// KeyDerivationMethod represents the method used for key derivation
type KeyDerivationMethod string

const (
	// Legacy SHA-256 method for backward compatibility
	MethodSHA256 KeyDerivationMethod = "sha256"
	// PBKDF2 method for enhanced security
	MethodPBKDF2 KeyDerivationMethod = "pbkdf2"
)

// KeyDerivationConfig holds configuration for key derivation
type KeyDerivationConfig struct {
	Method     KeyDerivationMethod `json:"method"`
	Salt       string              `json:"salt,omitempty"`       // Base64 encoded salt for PBKDF2
	Iterations int                 `json:"iterations,omitempty"` // Iterations for PBKDF2
	KeyLength  int                 `json:"keyLength,omitempty"`  // Key length for PBKDF2
}

// Default PBKDF2 configuration
const (
	DefaultPBKDF2Iterations = 100000 // OWASP recommended minimum
	DefaultKeyLength        = 32     // 256 bits
	SaltLength              = 32     // 256 bits
)

// SecureKeyDeriver provides enhanced key derivation with backward compatibility
type SecureKeyDeriver struct{}

// NewSecureKeyDeriver creates a new secure key deriver
func NewSecureKeyDeriver() *SecureKeyDeriver {
	return &SecureKeyDeriver{}
}

// DeriveKeySecure derives a key using PBKDF2 with proper salt
func (d *SecureKeyDeriver) DeriveKeySecure(password string) ([]byte, *KeyDerivationConfig, error) {
	// Generate random salt
	salt := make([]byte, SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return nil, nil, errors.Wrap(err, errors.ErrTypeCrypto, "SALT_GENERATION_FAILED",
			"failed to generate salt").
			WithUserMessage("Unable to generate secure encryption key")
	}

	// Derive key using PBKDF2
	key := pbkdf2.Key([]byte(password), salt, DefaultPBKDF2Iterations, DefaultKeyLength, sha256.New)

	config := &KeyDerivationConfig{
		Method:     MethodPBKDF2,
		Salt:       base64.StdEncoding.EncodeToString(salt),
		Iterations: DefaultPBKDF2Iterations,
		KeyLength:  DefaultKeyLength,
	}

	return key, config, nil
}

// DeriveKeyWithConfig derives a key using the provided configuration
func (d *SecureKeyDeriver) DeriveKeyWithConfig(password string, config *KeyDerivationConfig) ([]byte, error) {
	switch config.Method {
	case MethodPBKDF2:
		salt, err := base64.StdEncoding.DecodeString(config.Salt)
		if err != nil {
			return nil, errors.Wrap(err, errors.ErrTypeCrypto, "SALT_DECODE_FAILED",
				"failed to decode salt").
				WithUserMessage("Invalid encryption configuration")
		}

		key := pbkdf2.Key([]byte(password), salt, config.Iterations, config.KeyLength, sha256.New)
		return key, nil

	case MethodSHA256:
		// Legacy method for backward compatibility
		hash := sha256.Sum256([]byte(password))
		return hash[:], nil

	default:
		return nil, errors.New(errors.ErrTypeCrypto, "UNSUPPORTED_METHOD",
			"unsupported key derivation method").
			WithUserMessage("Unsupported encryption method").
			WithContext("method", string(config.Method))
	}
}

// DetectKeyDerivationMethod detects the key derivation method from existing data
func (d *SecureKeyDeriver) DetectKeyDerivationMethod(configPath string) (*KeyDerivationConfig, error) {
	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// No config file - assume legacy SHA-256
		return &KeyDerivationConfig{
			Method: MethodSHA256,
		}, nil
	}

	// Read and parse config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrTypeFileSystem, "CONFIG_READ_FAILED",
			"failed to read key derivation config").
			WithUserMessage("Unable to read encryption configuration")
	}

	var config KeyDerivationConfig
	if err := json.Unmarshal(data, &config); err != nil {
		// Invalid JSON - assume legacy
		return &KeyDerivationConfig{
			Method: MethodSHA256,
		}, nil
	}

	return &config, nil
}

// SaveKeyDerivationConfig saves the key derivation configuration
func (d *SecureKeyDeriver) SaveKeyDerivationConfig(config *KeyDerivationConfig, configPath string) error {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return errors.Wrap(err, errors.ErrTypeFileSystem, "DIR_CREATE_FAILED",
			"failed to create config directory").
			WithUserMessage("Unable to create configuration directory")
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return errors.Wrap(err, errors.ErrTypeConfig, "CONFIG_MARSHAL_FAILED",
			"failed to marshal config").
			WithUserMessage("Unable to format configuration")
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return errors.Wrap(err, errors.ErrTypeFileSystem, "CONFIG_WRITE_FAILED",
			"failed to write config").
			WithUserMessage("Unable to save encryption configuration")
	}

	return nil
}

// MigrateFromLegacy migrates from legacy SHA-256 to PBKDF2
func (d *SecureKeyDeriver) MigrateFromLegacy(password string, configPath string) ([]byte, error) {
	// Generate new PBKDF2 key and config
	newKey, config, err := d.DeriveKeySecure(password)
	if err != nil {
		return nil, err
	}

	// Save new configuration
	if err := d.SaveKeyDerivationConfig(config, configPath); err != nil {
		return nil, err
	}

	return newKey, nil
}

// DeriveKeyEnhanced - Enhanced key derivation function that maintains backward compatibility
func DeriveKeyEnhanced(password string, configPath string) ([]byte, error) {
	deriver := NewSecureKeyDeriver()

	// Detect current method
	config, err := deriver.DetectKeyDerivationMethod(configPath)
	if err != nil {
		return nil, err
	}

	// Derive key with existing method
	return deriver.DeriveKeyWithConfig(password, config)
}

// Note: DeriveKey function remains in crypto.go for backward compatibility
