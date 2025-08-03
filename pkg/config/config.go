package config

import (
	"encoding/json"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
)

// Config holds application configuration
type Config struct {
	NotesPath        string `json:"notesPath"`
	PasswordHashPath string `json:"passwordHashPath"`
}

// GetDefaultDataPath returns the default path for storing notes
func GetDefaultDataPath() string {
	currentUser, err := user.Current()
	if err != nil {
		return "./data"
	}

	var documentsDir string
	if runtime.GOOS == "windows" {
		documentsDir = filepath.Join(currentUser.HomeDir, "Documents")
	} else {
		documentsDir = filepath.Join(currentUser.HomeDir, "Documents")
	}

	// Create the default gote notes directory in Documents
	defaultPath := filepath.Join(documentsDir, "Gote", "Notes")

	// Ensure the directory exists
	if err := os.MkdirAll(defaultPath, 0755); err != nil {
		// Fall back to relative path if we can't create in Documents
		return "./data"
	}

	return defaultPath
}

// GetDefaultPasswordHashPath returns the default path for password hash storage
func GetDefaultPasswordHashPath() string {
	currentUser, err := user.Current()
	if err != nil {
		return filepath.Join("./data", ".password_hash")
	}

	// Use .config/gote directory for all platforms
	configDir := filepath.Join(currentUser.HomeDir, ".config")
	configPath := filepath.Join(configDir, "gote")

	if err := os.MkdirAll(configPath, 0755); err != nil {
		return filepath.Join("./data", ".password_hash")
	}

	return filepath.Join(configPath, "password_hash")
}

// GetConfigFilePath returns the path where the config file should be stored
func GetConfigFilePath() string {
	currentUser, err := user.Current()
	if err != nil {
		return "./config.json"
	}

	// Use .config/gote directory for all platforms
	configDir := filepath.Join(currentUser.HomeDir, ".config")
	configPath := filepath.Join(configDir, "gote")

	if err := os.MkdirAll(configPath, 0755); err != nil {
		return "./config.json"
	}

	return filepath.Join(configPath, "config")
}

// Load loads configuration from file, using defaults if file doesn't exist
func Load() (*Config, error) {
	config := &Config{
		NotesPath:        GetDefaultDataPath(),
		PasswordHashPath: GetDefaultPasswordHashPath(),
	}

	configFile := GetConfigFilePath()
	if data, err := os.ReadFile(configFile); err == nil {
		if err := json.Unmarshal(data, config); err != nil {
			return nil, err
		}
	}

	// Ensure directories exist
	if err := os.MkdirAll(config.NotesPath, 0755); err != nil {
		return nil, err
	}

	passwordDir := filepath.Dir(config.PasswordHashPath)
	if err := os.MkdirAll(passwordDir, 0755); err != nil {
		return nil, err
	}

	return config, nil
}

// Save saves the configuration to file
func (c *Config) Save() error {
	configFile := GetConfigFilePath()

	configDir := filepath.Dir(configFile)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configFile, data, 0644)
}
