package main

import (
	"fmt"
	"gote/pkg/config"
)

func main() {
	fmt.Printf("Config file path: %s\n", config.GetConfigFilePath())

	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	fmt.Printf("Notes Path: %s\n", cfg.NotesPath)
	fmt.Printf("Password Hash Path: %s\n", cfg.PasswordHashPath)
}
