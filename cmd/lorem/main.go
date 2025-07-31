package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"gote/pkg/auth"
	"gote/pkg/config"
	"gote/pkg/crypto"
	"gote/pkg/storage"
)

// loremIpsum returns a markdown string with lorem ipsum content
func loremIpsum() string {
	return `# Lorem Ipsum

Lorem ipsum dolor sit amet, **consectetur** adipiscing elit.

- Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.
- Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.

> Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur.

` +
		"```go\n" +
		"// Example code block\n" +
		"fmt.Println(\"Hello, world!\")\n" +
		"```\n"
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	fmt.Print("Enter password: ")
	reader := bufio.NewReader(os.Stdin)
	pw, _ := reader.ReadString('\n')
	pw = strings.TrimSpace(pw)

	authManager := auth.NewManager(cfg.PasswordHashPath)
	if !authManager.VerifyPassword(pw) {
		fmt.Fprintln(os.Stderr, "Invalid password")
		os.Exit(1)
	}

	key := crypto.DeriveKey(pw)
	store := storage.NewNoteStore(cfg.NotesPath)
	if err := store.LoadNotes(key); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load notes: %v\n", err)
		os.Exit(1)
	}

	note, err := store.CreateNote(loremIpsum(), key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create note: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated note with ID: %s\n", note.ID)
}
