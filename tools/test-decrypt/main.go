package main

import (
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
	if len(os.Args) != 2 {
		fmt.Println("Usage: test-decrypt <notes-directory>")
		fmt.Println("Example: test-decrypt \"C:\\Users\\rapha\\sync\\gote\"")
		os.Exit(1)
	}

	notesDir := os.Args[1]

	// Get password securely
	fmt.Print("Enter your Gote password: ")
	var password string
	fmt.Scanln(&password)

	fmt.Println("\n🧪 Cross-Platform Decryption Test")
	fmt.Println("==================================")
	fmt.Printf("Notes directory: %s\n", notesDir)

	// Test 1: Check if cross-platform config exists
	configPath := filepath.Join(notesDir, ".gote_config.json")
	fmt.Printf("Checking for config: %s\n", configPath)

	data, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Printf("❌ Could not read config file: %v\n", err)
		os.Exit(1)
	}

	var config CrossPlatformConfig
	if err := json.Unmarshal(data, &config); err != nil {
		fmt.Printf("❌ Could not parse config file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Config loaded successfully\n")
	fmt.Printf("   Salt: %s\n", config.Salt[:20]+"...")
	fmt.Printf("   Created: %s\n", config.CreatedAt)

	// Test 2: Decode salt and derive key
	salt, err := base64.StdEncoding.DecodeString(config.Salt)
	if err != nil {
		fmt.Printf("❌ Could not decode salt: %v\n", err)
		os.Exit(1)
	}

	key := crypto.DeriveKey(password, salt)
	fmt.Printf("✅ Encryption key derived successfully\n")
	fmt.Printf("   Key length: %d bytes\n", len(key))

	// Test 3: Try to decrypt a note file
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
	fmt.Printf("🧪 Testing decryption with: %s\n", filepath.Base(testFile))

	noteData, err := os.ReadFile(testFile)
	if err != nil {
		fmt.Printf("❌ Could not read note file: %v\n", err)
		os.Exit(1)
	}

	// Try to parse as encrypted note
	var encryptedNote struct {
		ID            string `json:"id"`
		EncryptedData string `json:"encrypted_data"` // snake_case format
		CreatedAt     string `json:"created_at"`
		UpdatedAt     string `json:"updated_at"`
	}

	if err := json.Unmarshal(noteData, &encryptedNote); err != nil {
		fmt.Printf("❌ Could not parse encrypted note: %v\n", err)
		os.Exit(1)
	}

	// Try to decrypt
	decryptedContent, err := crypto.Decrypt(encryptedNote.EncryptedData, key)
	if err != nil {
		fmt.Printf("❌ Decryption failed: %v\n", err)
		fmt.Printf("   This indicates the key derivation might be different\n")
		os.Exit(1)
	}

	fmt.Printf("✅ Decryption successful!\n")
	fmt.Printf("   Content preview: %s\n", truncateString(decryptedContent, 50))

	fmt.Println("\n🎉 Cross-platform decryption is working correctly!")
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
