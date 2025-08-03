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

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: verify.exe <data-directory>")
		fmt.Println("Example: verify.exe \"C:\\Users\\rapha\\sync\\gote\"")
		os.Exit(1)
	}

	dataDir := os.Args[1]

	// Check if directory exists
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		log.Fatalf("Data directory does not exist: %s", dataDir)
	}

	// Get password
	fmt.Print("Enter your password: ")
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		log.Fatalf("Failed to read password: %v", err)
	}
	fmt.Println() // New line after password input
	password := string(passwordBytes)

	// Use enhanced key derivation
	configPath := filepath.Join(dataDir, ".keyconfig.json")
	key, err := crypto.DeriveKeyEnhanced(password, configPath)
	if err != nil {
		log.Fatalf("Failed to derive key: %v", err)
	}

	// Find all note files
	noteFiles, err := filepath.Glob(filepath.Join(dataDir, "*.json"))
	if err != nil {
		log.Fatalf("Failed to find note files: %v", err)
	}

	// Filter out the config file
	var actualNoteFiles []string
	for _, file := range noteFiles {
		if filepath.Base(file) != ".keyconfig.json" {
			actualNoteFiles = append(actualNoteFiles, file)
		}
	}

	if len(actualNoteFiles) == 0 {
		fmt.Println("No note files found in directory")
		return
	}

	fmt.Printf("Found %d note files\n", len(actualNoteFiles))

	successCount := 0
	failCount := 0

	// Try to decrypt each note
	for _, noteFile := range actualNoteFiles {
		data, err := os.ReadFile(noteFile)
		if err != nil {
			fmt.Printf("❌ Failed to read %s: %v\n", filepath.Base(noteFile), err)
			failCount++
			continue
		}

		var encryptedNote models.EncryptedNote
		if err := json.Unmarshal(data, &encryptedNote); err != nil {
			fmt.Printf("❌ Failed to parse %s: %v\n", filepath.Base(noteFile), err)
			failCount++
			continue
		}

		// Try to decrypt
		decryptedContent, err := crypto.Decrypt(encryptedNote.EncryptedData, key)
		if err != nil {
			fmt.Printf("❌ Failed to decrypt %s: %v\n", filepath.Base(noteFile), err)
			failCount++
			continue
		}

		// Check if content makes sense (not empty)
		if len(decryptedContent) == 0 {
			fmt.Printf("❌ Empty content in %s\n", filepath.Base(noteFile))
			failCount++
			continue
		}

		fmt.Printf("✅ Successfully decrypted %s (content length: %d)\n", filepath.Base(noteFile), len(decryptedContent))
		successCount++
	}

	fmt.Printf("\n=== SUMMARY ===\n")
	fmt.Printf("✅ Successfully decrypted: %d notes\n", successCount)
	fmt.Printf("❌ Failed to decrypt: %d notes\n", failCount)

	if failCount == 0 {
		fmt.Println("\n🎉 All notes can be decrypted successfully!")
		fmt.Println("The issue might be with the application not loading the notes directory correctly.")
		fmt.Printf("Make sure your app is configured to use: %s\n", dataDir)
	} else {
		fmt.Println("\n⚠️ Some notes could not be decrypted.")
		fmt.Println("This might indicate a password issue or migration problem.")
	}
}
