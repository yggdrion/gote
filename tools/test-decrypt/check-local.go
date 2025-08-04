package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
)

type PasswordData struct {
	Hash string `json:"hash"`
	Salt string `json:"salt"`
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: check-local-config <password-hash-path>")
		os.Exit(1)
	}

	hashPath := os.Args[1]
	fmt.Printf("Checking local password hash at: %s\n", hashPath)

	data, err := os.ReadFile(hashPath)
	if err != nil {
		fmt.Printf("Error reading password hash: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Password hash file size: %d bytes\n", len(data))
	fmt.Printf("Password hash content:\n%s\n", string(data))

	var passwordData PasswordData
	if err := json.Unmarshal(data, &passwordData); err != nil {
		fmt.Printf("Error parsing password hash: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Parsed password data:\n")
	fmt.Printf("  Hash: %s...\n", passwordData.Hash[:20])
	fmt.Printf("  Salt: %s\n", passwordData.Salt)

	// Decode salt to verify it's valid
	salt, err := base64.StdEncoding.DecodeString(passwordData.Salt)
	if err != nil {
		fmt.Printf("Error decoding salt: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Salt decoded successfully: %d bytes\n", len(salt))
	fmt.Printf("Salt preview: %s...\n", passwordData.Salt[:20])
}
