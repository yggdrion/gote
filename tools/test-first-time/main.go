package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gote/pkg/auth"
)

func main() {
	fmt.Println("🧪 Cross-Platform First-Time Setup Test")
	fmt.Println("======================================")

	// Simulate a new device setup
	testNotesDir := "C:\\Users\\rapha\\sync\\gote"
	testPasswordHashPath := ".\\test_password_hash"

	// Clean up any existing test files
	os.Remove(testPasswordHashPath)

	fmt.Printf("Notes directory: %s\n", testNotesDir)
	fmt.Printf("Test password hash path: %s\n", testPasswordHashPath)

	// Check if cross-platform config exists
	configPath := filepath.Join(testNotesDir, ".gote_config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Printf("❌ Cross-platform config not found: %s\n", configPath)
		fmt.Println("   Make sure the migration tool has been run!")
		os.Exit(1)
	}

	fmt.Printf("✅ Cross-platform config found: %s\n", configPath)

	// Create auth manager with notes directory
	authManager := auth.NewManagerWithNotesDir(testPasswordHashPath, testNotesDir)

	// Test IsFirstTimeSetup - should return false because cross-platform config exists
	isFirstTime := authManager.IsFirstTimeSetup()
	fmt.Printf("IsFirstTimeSetup(): %t\n", isFirstTime)

	if isFirstTime {
		fmt.Printf("❌ FAIL: Should not be first time setup when cross-platform config exists\n")
		os.Exit(1)
	}

	fmt.Printf("✅ PASS: Correctly detected existing cross-platform config\n")

	// Get password securely
	fmt.Print("\nEnter your Gote password to test verification: ")
	var password string
	fmt.Scanln(&password)

	// Test password verification
	fmt.Printf("Testing password verification...\n")
	isValid := authManager.VerifyPassword(password)

	if !isValid {
		fmt.Printf("❌ FAIL: Password verification failed\n")
		fmt.Printf("   This could mean:\n")
		fmt.Printf("   1. Incorrect password\n")
		fmt.Printf("   2. Cross-platform config has wrong salt\n")
		fmt.Printf("   3. Password verification logic needs fixing\n")
		os.Exit(1)
	}

	fmt.Printf("✅ PASS: Password verification successful\n")

	// Test key derivation
	fmt.Printf("Testing encryption key derivation...\n")
	key, err := authManager.DeriveEncryptionKey(password)
	if err != nil {
		fmt.Printf("❌ FAIL: Key derivation failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ PASS: Key derivation successful (key length: %d bytes)\n", len(key))

	// Check if local password hash was created
	if _, err := os.Stat(testPasswordHashPath); os.IsNotExist(err) {
		fmt.Printf("⚠️  Local password hash not created yet\n")
		fmt.Printf("   This will be created during SyncFromCrossPlatform\n")
	} else {
		fmt.Printf("✅ Local password hash file exists\n")
	}

	// Clean up
	os.Remove(testPasswordHashPath)

	fmt.Println("\n🎉 Cross-platform first-time setup test completed successfully!")
	fmt.Println("   Your authentication system should work on new devices.")
}
