package types

import (
	"gote/pkg/models"
	"time"
)

// WailsNote represents a note structure optimized for Wails bindings
type WailsNote struct {
	ID        string `json:"id"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"` // Use string representation for better Wails compatibility
	UpdatedAt string `json:"updated_at"` // Use string representation for better Wails compatibility
}

// ConvertToWailsNote converts a models.Note to WailsNote with proper time formatting
func ConvertToWailsNote(note *models.Note) WailsNote {
	if note == nil {
		return WailsNote{}
	}

	return WailsNote{
		ID:        note.ID,
		Content:   note.Content,
		CreatedAt: note.CreatedAt.Format(time.RFC3339),
		UpdatedAt: note.UpdatedAt.Format(time.RFC3339),
	}
}

// ConvertToWailsNotes converts a slice of models.Note to slice of WailsNote
func ConvertToWailsNotes(notes []*models.Note) []WailsNote {
	if notes == nil {
		return []WailsNote{}
	}

	wailsNotes := make([]WailsNote, len(notes))
	for i, note := range notes {
		wailsNotes[i] = ConvertToWailsNote(note)
	}
	return wailsNotes
}
