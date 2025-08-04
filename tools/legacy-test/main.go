package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gote/pkg/crypto"
)

// CrossPlatformConfig for testing
type CrossPlatformConfig struct {
	Salt      string `json:"salt"`
	CreatedAt string `json:"createdAt"`
	Version   string `json:"version"`
}

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: legacy-test <notes-directory> <password>")
		fmt.Println("Example: legacy-test \"C:\\Users\\rapha\\sync\\gote\" \"your-password\"")
		os.Exit(1)
	}

	notesDir := os.Args[1]
	password := os.Args[2]

	fmt.Println("🔍 Legacy Encryption Detection Tool")
	fmt.Println("===================================")
	fmt.Printf("Notes directory: %s\n", notesDir)

	// Find a test note file
	files, err := filepath.Glob(filepath.Join(notesDir, "*.json"))
	if err != nil {
		fmt.Printf("❌ Could not scan for note files: %v\n", err)
		os.Exit(1)
	}

	var noteFiles []string
	for _, file := range files {
		if filepath.Base(file) != ".gote_config.json" {
			noteFiles = append(noteFiles, file)
		}
	}

	if len(noteFiles) == 0 {
		fmt.Printf("⚠️  No note files found for testing\n")
		return
	}

	testFile := noteFiles[0]
	fmt.Printf("🧪 Testing with: %s\n", filepath.Base(testFile))

	noteData, err := os.ReadFile(testFile)
	if err != nil {
		fmt.Printf("❌ Could not read note file: %v\n", err)
		os.Exit(1)
	}

	// Parse encrypted note
	var encryptedNote struct {
		ID            string `json:"id"`
		EncryptedData string `json:"encrypted_data"`
		CreatedAt     string `json:"created_at"`
		UpdatedAt     string `json:"updated_at"`
	}

	if err := json.Unmarshal(noteData, &encryptedNote); err != nil {
		fmt.Printf("❌ Could not parse encrypted note: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Note file parsed successfully\n")
	fmt.Printf("   Encrypted data length: %d characters\n", len(encryptedNote.EncryptedData))

	// Test 1: Try with PBKDF2 + cross-platform salt
	fmt.Println("\n🔐 Test 1: PBKDF2 with cross-platform salt")
	configPath := filepath.Join(notesDir, ".gote_config.json")
	if data, err := os.ReadFile(configPath); err == nil {
		var config CrossPlatformConfig
		if err := json.Unmarshal(data, &config); err == nil {
			if salt, err := base64.StdEncoding.DecodeString(config.Salt); err == nil {
				key := crypto.DeriveKey(password, salt)
				if content, err := crypto.Decrypt(encryptedNote.EncryptedData, key); err == nil {
					fmt.Printf("✅ SUCCESS: PBKDF2 + cross-platform salt works!\n")
					fmt.Printf("   Content preview: %s\n", truncateString(content, 50))
					return
				} else {
					fmt.Printf("❌ Failed: %v\n", err)
				}
			}
		}
	}

	// Test 2: Try with legacy SHA-256 method (no salt)
	fmt.Println("\n🔐 Test 2: Legacy SHA-256 (no salt)")
	legacyKey := sha256.Sum256([]byte(password))
	if content, err := crypto.Decrypt(encryptedNote.EncryptedData, legacyKey[:]); err == nil {
		fmt.Printf("✅ SUCCESS: Legacy SHA-256 method works!\n")
		fmt.Printf("   Content preview: %s\n", truncateString(content, 50))
		fmt.Printf("   -> Your notes use the OLD encryption method\n")
		fmt.Printf("   -> You need to run the full migration tool to upgrade them\n")
		return
	} else {
		fmt.Printf("❌ Failed: %v\n", err)
	}

	// Test 3: Try PBKDF2 with local salt (if different from cross-platform)
	fmt.Println("\n🔐 Test 3: PBKDF2 with possible local salt")
	// This would require reading the local password hash file
	fmt.Printf("⚠️  Could check local password hash file for different salt\n")

	fmt.Println("\n❌ All decryption methods failed!")
	fmt.Println("Possible causes:")
	fmt.Println("1. Wrong password")
	fmt.Println("2. Notes encrypted with different salt")
	fmt.Println("3. Notes corrupted during sync")
	fmt.Println("4. Different encryption method not tested")
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
