package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"syscall"

	"gote/pkg/crypto"
	"gote/pkg/models"

	"golang.org/x/term"
)

// MigrationTool helps migrate notes from legacy encryption to PBKDF2
type MigrationTool struct {
	dataDir string
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run migrate.go <data-directory>")
		fmt.Println("Example: go run migrate.go C:\\Users\\rapha\\sync\\gote")
		os.Exit(1)
	}

	dataDir := os.Args[1]
	
	// Check if directory exists
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		log.Fatalf("Data directory does not exist: %s", dataDir)
	}

	tool := &MigrationTool{dataDir: dataDir}
	if err := tool.migrate(); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	fmt.Println("Migration completed successfully!")
}

func (m *MigrationTool) migrate() error {
	fmt.Printf("Starting migration for directory: %s\n", m.dataDir)
	
	// Get password from user
	fmt.Print("Enter your password: ")
	password, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return fmt.Errorf("failed to read password: %v", err)
	}
	fmt.Println() // New line after password input

	configPath := filepath.Join(m.dataDir, ".keyconfig.json")
	
	// Check if already using PBKDF2
	deriver := crypto.NewSecureKeyDeriver()
	config, err := deriver.DetectKeyDerivationMethod(configPath)
	if err != nil {
		return fmt.Errorf("failed to detect key derivation method: %v", err)
	}

	if config.Method == crypto.MethodPBKDF2 {
		fmt.Println("Notes are already using PBKDF2 encryption.")
		return m.validateExistingNotes(string(password), config, deriver)
	}

	fmt.Println("Detected legacy SHA-256 encryption. Starting migration...")
	
	// Get list of note files
	noteFiles, err := filepath.Glob(filepath.Join(m.dataDir, "*.json"))
	if err != nil {
		return fmt.Errorf("failed to list note files: %v", err)
	}

	if len(noteFiles) == 0 {
		fmt.Println("No notes found to migrate.")
		return nil
	}

	fmt.Printf("Found %d notes to migrate.\n", len(noteFiles))

	// Derive legacy key
	legacyKey := crypto.DeriveKey(string(password))
	
	// Test decryption with legacy key
	fmt.Println("Testing legacy key with first note...")
	if !m.testDecryption(noteFiles[0], legacyKey) {
		return fmt.Errorf("failed to decrypt notes with provided password")
	}

	// Generate new PBKDF2 key
	fmt.Println("Generating new PBKDF2 key...")
	newKey, newConfig, err := deriver.DeriveKeySecure(string(password))
	if err != nil {
		return fmt.Errorf("failed to generate PBKDF2 key: %v", err)
	}

	// Create backup directory
	backupDir := filepath.Join(m.dataDir, "backup_before_migration")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %v", err)
	}

	// Migrate each note
	fmt.Println("Migrating notes...")
	for i, noteFile := range noteFiles {
		fmt.Printf("Migrating note %d/%d: %s\n", i+1, len(noteFiles), filepath.Base(noteFile))
		
		if err := m.migrateNote(noteFile, legacyKey, newKey, backupDir); err != nil {
			return fmt.Errorf("failed to migrate note %s: %v", noteFile, err)
		}
	}

	// Save new configuration
	fmt.Println("Saving new encryption configuration...")
	if err := deriver.SaveKeyDerivationConfig(newConfig, configPath); err != nil {
		return fmt.Errorf("failed to save new configuration: %v", err)
	}

	fmt.Printf("Migration completed! Backup created in: %s\n", backupDir)
	return nil
}

func (m *MigrationTool) testDecryption(noteFile string, key []byte) bool {
	data, err := os.ReadFile(noteFile)
	if err != nil {
		return false
	}

	var encryptedNote models.EncryptedNote
	if err := json.Unmarshal(data, &encryptedNote); err != nil {
		return false
	}

	_, err = crypto.Decrypt(encryptedNote.EncryptedData, key)
	return err == nil
}

func (m *MigrationTool) migrateNote(noteFile string, oldKey, newKey []byte, backupDir string) error {
	// Read encrypted note
	data, err := os.ReadFile(noteFile)
	if err != nil {
		return fmt.Errorf("failed to read note file: %v", err)
	}

	// Create backup
	backupFile := filepath.Join(backupDir, filepath.Base(noteFile))
	if err := os.WriteFile(backupFile, data, 0600); err != nil {
		return fmt.Errorf("failed to create backup: %v", err)
	}

	var encryptedNote models.EncryptedNote
	if err := json.Unmarshal(data, &encryptedNote); err != nil {
		return fmt.Errorf("failed to unmarshal encrypted note: %v", err)
	}

	// Decrypt with old key
	decryptedContent, err := crypto.Decrypt(encryptedNote.EncryptedData, oldKey)
	if err != nil {
		return fmt.Errorf("failed to decrypt with old key: %v", err)
	}

	// Re-encrypt with new key
	newEncryptedContent, err := crypto.Encrypt(decryptedContent, newKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt with new key: %v", err)
	}

	// Update encrypted note
	encryptedNote.EncryptedData = newEncryptedContent

	// Save updated note
	newData, err := json.MarshalIndent(encryptedNote, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal updated note: %v", err)
	}

	if err := os.WriteFile(noteFile, newData, 0600); err != nil {
		return fmt.Errorf("failed to save updated note: %v", err)
	}

	return nil
}

func (m *MigrationTool) validateExistingNotes(password string, config *crypto.KeyDerivationConfig, deriver *crypto.SecureKeyDeriver) error {
	fmt.Println("Validating existing notes...")
	
	// Derive key with current config
	key, err := deriver.DeriveKeyWithConfig(password, config)
	if err != nil {
		return fmt.Errorf("failed to derive key: %v", err)
	}

	// Get list of note files
	noteFiles, err := filepath.Glob(filepath.Join(m.dataDir, "*.json"))
	if err != nil {
		return fmt.Errorf("failed to list note files: %v", err)
	}

	failedNotes := []string{}
	for _, noteFile := range noteFiles {
		if !m.testDecryption(noteFile, key) {
			failedNotes = append(failedNotes, filepath.Base(noteFile))
		}
	}

	if len(failedNotes) > 0 {
		fmt.Printf("WARNING: %d notes failed validation:\n", len(failedNotes))
		for _, note := range failedNotes {
			fmt.Printf("  - %s\n", note)
		}
		fmt.Println("These notes may have been encrypted with a different password.")
		return fmt.Errorf("validation failed for %d notes", len(failedNotes))
	}

	fmt.Printf("All %d notes validated successfully!\n", len(noteFiles))
	return nil
}
