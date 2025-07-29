package handlers

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"regexp"
	"strings"

	"gote/pkg/models"
	"gote/pkg/storage"
)

// WebHandlers contains handlers for web interface
type WebHandlers struct {
	store       *storage.NoteStore
	authManager AuthManager
}

// AuthManager interface for authentication operations
type AuthManager interface {
	IsAuthenticated(r *http.Request) *models.Session
	IsFirstTimeSetup() bool
}

// NewWebHandlers creates a new web handlers instance
func NewWebHandlers(store *storage.NoteStore, authManager AuthManager) *WebHandlers {
	return &WebHandlers{
		store:       store,
		authManager: authManager,
	}
}

// IndexHandler serves the main page
func (h *WebHandlers) IndexHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	var notes []*models.Note

	if query != "" {
		notes = h.store.SearchNotes(query)
	} else {
		notes = h.store.GetAllNotes()
	}

	funcMap := template.FuncMap{
		"formatContent": func(s string) template.HTML {
			if s == "" {
				return template.HTML("Empty note...")
			}

			// Escape HTML first
			s = template.HTMLEscapeString(s)

			// Apply basic markdown formatting

			// Code blocks (must be processed before line breaks)
			codeBlockRegex := regexp.MustCompile("(?s)```([\\s\\S]*?)```")
			codeBlocks := codeBlockRegex.FindAllString(s, -1)
			codeBlockPlaceholders := make([]string, len(codeBlocks))

			// Replace code blocks with placeholders to protect them from other formatting
			for i, block := range codeBlocks {
				placeholder := fmt.Sprintf("__CODEBLOCK_%d__", i)
				codeBlockPlaceholders[i] = placeholder
				content := codeBlockRegex.FindStringSubmatch(block)
				if len(content) > 1 {
					// Store the processed code block
					codeBlockPlaceholders[i] = "<pre><code>" + strings.TrimSpace(content[1]) + "</code></pre>"
				}
				s = strings.Replace(s, block, fmt.Sprintf("__CODEBLOCK_%d__", i), 1)
			}

			// Convert line breaks
			s = strings.ReplaceAll(s, "\n", "<br>")

			// Bold - fix the regex to be more specific
			boldRegex := regexp.MustCompile(`\*\*([^*]+)\*\*`)
			s = boldRegex.ReplaceAllString(s, "<strong>$1</strong>")

			// Italic - make sure it doesn't conflict with bold
			italicRegex := regexp.MustCompile(`\*([^*\n]+)\*`)
			s = italicRegex.ReplaceAllString(s, "<em>$1</em>")

			// Inline code (process after code blocks to avoid conflicts)
			inlineCodeRegex := regexp.MustCompile("`([^`\n]+)`")
			s = inlineCodeRegex.ReplaceAllString(s, "<code>$1</code>")

			// Simple heading support
			headingRegex := regexp.MustCompile(`(?m)^# (.+)$`)
			s = headingRegex.ReplaceAllString(s, "<strong style='font-size:1.1em;color:#333;'>$1</strong>")

			// Restore code blocks
			for i, processedBlock := range codeBlockPlaceholders {
				s = strings.Replace(s, fmt.Sprintf("__CODEBLOCK_%d__", i), processedBlock, 1)
			}

			return template.HTML(s)
		},
	}

	tmpl, err := template.New("index.html").Funcs(funcMap).ParseFiles("./static/index.html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		log.Printf("Template parsing error: %v", err)
		return
	}

	data := struct {
		Notes []*models.Note
		Query string
	}{
		Notes: notes,
		Query: query,
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Template execution error", http.StatusInternalServerError)
		log.Printf("Template execution error: %v", err)
	}
}
