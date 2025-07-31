package main

import (
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/jaswdr/faker"
	"golang.org/x/term"

	"gote/pkg/auth"
	"gote/pkg/config"
	"gote/pkg/crypto"
	"gote/pkg/storage"
)

// loremIpsum returns a markdown string with generated lorem ipsum content
func loremIpsum() string {
	f := faker.New()
	paragraph := f.Lorem().Paragraph(2)
	bullet1 := f.Lorem().Sentence(8)
	bullet2 := f.Lorem().Sentence(10)
	quote := f.Lorem().Sentence(40)
	code := f.Lorem().Word() + " := \"" + f.Lorem().Word() + "\""

	return "# Lorem Ipsum\n\n" +
		paragraph + "\n\n" +
		"- " + bullet1 + "\n" +
		"- " + bullet2 + "\n\n" +
		"> " + quote + "\n\n" +
		"```go\n" +
		"// Example code block\n" +
		code + "\n" +
		"```\n"
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	fmt.Print("Enter password: ")
	bytePw, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to read password")
		os.Exit(1)
	}
	pw := strings.TrimSpace(string(bytePw))

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
