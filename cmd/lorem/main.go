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

// loremIpsum returns a markdown string with highly randomized and varied content
func loremIpsum() string {
	f := faker.New()
	variant := f.IntBetween(1, 4)

	switch variant {
	case 1:
		// Full markdown: title, paragraphs, bullets, quote, code
		paragraphs := []string{}
		for i := 0; i < f.IntBetween(1, 3); i++ {
			paragraphs = append(paragraphs, f.Lorem().Paragraph(f.IntBetween(1, 3)))
		}
		bullets := []string{}
		for i := 0; i < f.IntBetween(2, 5); i++ {
			bullets = append(bullets, f.Lorem().Sentence(f.IntBetween(6, 14)))
		}
		quote := f.Lorem().Sentence(f.IntBetween(20, 50))
		codeLines := []string{"// Example code block"}
		for i := 0; i < f.IntBetween(1, 3); i++ {
			codeLines = append(codeLines, f.Lorem().Word()+" := \""+f.Lorem().Word()+"\"")
		}
		return "# " + f.Lorem().Word() + " " + f.Lorem().Word() + "\n\n" +
			strings.Join(paragraphs, "\n\n") + "\n\n" +
			"- " + strings.Join(bullets, "\n- ") + "\n\n" +
			"> " + quote + "\n\n" +
			"```go\n" +
			strings.Join(codeLines, "\n") + "\n" +
			"```\n"
	case 2:
		// Only code block
		codeLines := []string{"// Only code block"}
		for i := 0; i < f.IntBetween(2, 6); i++ {
			codeLines = append(codeLines, f.Lorem().Word()+" := \""+f.Lorem().Word()+"\"")
		}
		return "```go\n" + strings.Join(codeLines, "\n") + "\n```\n"
	case 3:
		// Only bullets
		bullets := []string{}
		for i := 0; i < f.IntBetween(3, 8); i++ {
			bullets = append(bullets, f.Lorem().Sentence(f.IntBetween(4, 12)))
		}
		return "- " + strings.Join(bullets, "\n- ") + "\n"
	case 4:
		// Only quote
		return "> " + f.Lorem().Sentence(f.IntBetween(30, 60)) + "\n"
	default:
		return f.Lorem().Paragraph(1) + "\n"
	}
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
