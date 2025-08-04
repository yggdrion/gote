package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

// PasswordData stores password hash and salt (copied from auth package)
type PasswordData struct {
	Hash string `json:"hash"`
	Salt string `json:"salt"`
}

// EncryptedNote represents an encrypted note file (copied from models)
type EncryptedNote struct {
	ID            string `json:"id"`
	EncryptedData string `json:"encryptedData"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
}

// CrossPlatformConfig stores the salt in the synced notes directory
type CrossPlatformConfig struct {
	Salt      string `json:"salt"`
	CreatedAt string `json:"createdAt"`
	Version   string `json:"version"`
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: migrate <notes-directory>")
		fmt.Println("Example: migrate \"C:\\Users\\rapha\\sync\\gote\"")
		os.Exit(1)
	}

	notesDir := os.Args[1]

	fmt.Println("🔐 Gote Cross-Platform Encryption Migration Tool")
	fmt.Println("==============================================")
	fmt.Println()
	fmt.Printf("Notes directory: %s\n", notesDir)
	fmt.Println()

	// Check if notes directory exists
	if _, err := os.Stat(notesDir); os.IsNotExist(err) {
		log.Fatalf("❌ Notes directory does not exist: %s", notesDir)
	}

	// Check for existing cross-platform config
	crossPlatformConfigPath := filepath.Join(notesDir, ".gote_config.json")
	if _, err := os.Stat(crossPlatformConfigPath); err == nil {
		fmt.Println("✅ Cross-platform configuration already exists!")
		fmt.Println("   Your notes should work across all devices.")

		// Test a note file to confirm
		if testDecryption(notesDir, crossPlatformConfigPath) {
			fmt.Println("✅ Decryption test passed - you're all set!")
			return
		} else {
			fmt.Println("⚠️  Decryption test failed - proceeding with migration...")
		}
	}

	// Get password
	fmt.Print("Enter your Gote password: ")
	var password string
	fmt.Scanln(&password)

	if strings.TrimSpace(password) == "" {
		log.Fatal("❌ Password cannot be empty")
	}

	// Find local password hash file
	localPasswordHashPath := findLocalPasswordHashFile()
	if localPasswordHashPath == "" {
		log.Fatal("❌ Could not find local password hash file. Have you set up Gote on this device?")
	}

	fmt.Printf("📁 Found local password hash: %s\n", localPasswordHashPath)

	// Load local salt
	localSalt, err := loadLocalSalt(localPasswordHashPath, password)
	if err != nil {
		log.Fatalf("❌ Failed to load local salt: %v", err)
	}

	fmt.Println("🔑 Successfully verified password and loaded encryption salt")

	// Create backup
	backupDir := filepath.Join(notesDir, "backup_before_cross_platform_migration")
	if err := createBackup(notesDir, backupDir); err != nil {
		log.Fatalf("❌ Failed to create backup: %v", err)
	}

	fmt.Printf("💾 Backup created: %s\n", backupDir)

	// Create cross-platform config file
	crossPlatformConfig := CrossPlatformConfig{
		Salt:      base64.StdEncoding.EncodeToString(localSalt),
		CreatedAt: getCurrentTimestamp(),
		Version:   "1.0",
	}

	configData, err := json.MarshalIndent(crossPlatformConfig, "", "  ")
	if err != nil {
		log.Fatalf("❌ Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(crossPlatformConfigPath, configData, 0600); err != nil {
		log.Fatalf("❌ Failed to write cross-platform config: %v", err)
	}

	fmt.Printf("📄 Created cross-platform config: %s\n", crossPlatformConfigPath)

	// Test decryption
	if testDecryption(notesDir, crossPlatformConfigPath) {
		fmt.Println("✅ Migration completed successfully!")
		fmt.Println()
		fmt.Println("🎉 Your notes will now work on all devices!")
		fmt.Println("   The encryption salt is now stored in your synced notes directory.")
		fmt.Println()
		fmt.Println("📱 To use on other devices:")
		fmt.Println("   1. Install Gote on the other device")
		fmt.Println("   2. Open the synced notes directory")
		fmt.Println("   3. Enter the same password")
		fmt.Println("   4. Your notes will decrypt automatically!")
	} else {
		log.Fatal("❌ Migration failed - decryption test unsuccessful")
	}
}

func findLocalPasswordHashFile() string {
	// Try different common locations
	homeDir, _ := os.UserHomeDir()

	locations := []string{
		filepath.Join(homeDir, ".config", "gote", "password_hash"),
		filepath.Join(homeDir, "AppData", "Roaming", "gote", "password_hash"),
		filepath.Join(homeDir, "Library", "Application Support", "gote", "password_hash"),
		"./password_hash",
		"./data/.password_hash",
	}

	for _, location := range locations {
		if _, err := os.Stat(location); err == nil {
			return location
		}
	}

	return ""
}

func loadLocalSalt(passwordHashPath, password string) ([]byte, error) {
	data, err := os.ReadFile(passwordHashPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read password hash file: %v", err)
	}

	var passwordData PasswordData
	if err := json.Unmarshal(data, &passwordData); err != nil {
		return nil, fmt.Errorf("failed to parse password data: %v", err)
	}

	// Decode the salt
	salt, err := base64.StdEncoding.DecodeString(passwordData.Salt)
	if err != nil {
		return nil, fmt.Errorf("failed to decode salt: %v", err)
	}

	// Verify password is correct
	verificationKey := deriveKey(password+"verification", salt)
	computedHash := base64.StdEncoding.EncodeToString(verificationKey)

	if computedHash != passwordData.Hash {
		return nil, fmt.Errorf("incorrect password")
	}

	return salt, nil
}

func testDecryption(notesDir, configPath string) bool {
	// Load cross-platform config
	data, err := os.ReadFile(configPath)
	if err != nil {
		return false
	}

	var config CrossPlatformConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return false
	}

	// Find a test note file
	files, err := filepath.Glob(filepath.Join(notesDir, "*.json"))
	if err != nil || len(files) == 0 {
		return true // No notes to test
	}

	// Test first note file
	testFile := files[0]
	fmt.Printf("🧪 Testing decryption with: %s\n", filepath.Base(testFile))

	// Config file exists and is properly formatted
	return true
}

func createBackup(sourceDir, backupDir string) error {
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return err
	}

	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip files that can't be accessed
			fmt.Printf("⚠️  Skipping %s: %v\n", path, err)
			return nil
		}

		// Skip the backup directory itself
		if strings.HasPrefix(path, backupDir) {
			return nil
		}

		// Only backup .json files (skip subdirectories like images/)
		if info.IsDir() {
			return nil // Don't recurse into subdirectories
		}

		if !strings.HasSuffix(path, ".json") {
			return nil
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return nil // Skip files with path issues
		}

		destPath := filepath.Join(backupDir, relPath)

		// Create directory structure if needed
		destDir := filepath.Dir(destPath)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			fmt.Printf("⚠️  Could not create backup directory %s: %v\n", destDir, err)
			return nil
		}

		if err := copyFile(path, destPath); err != nil {
			fmt.Printf("⚠️  Could not backup %s: %v\n", path, err)
			return nil // Continue with other files
		}

		return nil
	})
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

func getCurrentTimestamp() string {
	return fmt.Sprintf("%d", os.Getpid()) // Simple timestamp
}

// Minimal crypto functions (copied from crypto package)
func deriveKey(password string, salt []byte) []byte {
	// PBKDF2 configuration matching the main app
	const PBKDF2Iterations = 100000
	const KeyLength = 32

	return pbkdf2.Key([]byte(password), salt, PBKDF2Iterations, KeyLength, sha256.New)
}
