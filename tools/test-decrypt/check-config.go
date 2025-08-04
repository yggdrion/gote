package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type CrossPlatformConfigCheck struct {
	Salt      string `json:"salt"`
	CreatedAt string `json:"createdAt"`
	Version   string `json:"version"`
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: check-config <notes-directory>")
		os.Exit(1)
	}

	notesDir := os.Args[1]
	configPath := filepath.Join(notesDir, ".gote_config.json")

	fmt.Printf("Checking cross-platform config at: %s\n", configPath)

	data, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Printf("Error reading config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Config file size: %d bytes\n", len(data))
	fmt.Printf("Config content:\n%s\n", string(data))

	var config CrossPlatformConfigCheck
	if err := json.Unmarshal(data, &config); err != nil {
		fmt.Printf("Error parsing config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Parsed config:\n")
	fmt.Printf("  Salt: %s\n", config.Salt)
	fmt.Printf("  Created: %s\n", config.CreatedAt)
	fmt.Printf("  Version: %s\n", config.Version)

	// Decode salt to verify it's valid
	salt, err := base64.StdEncoding.DecodeString(config.Salt)
	if err != nil {
		fmt.Printf("Error decoding salt: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Salt decoded successfully: %d bytes\n", len(salt))
	fmt.Printf("Salt preview: %s...\n", config.Salt[:20])
}
