package models

import "time"

// Note represents a decrypted note in memory
type Note struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	Images    []Image   `json:"images,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Image represents an embedded image in a note
type Image struct {
	ID          string    `json:"id"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	Size        int64     `json:"size"`
	CreatedAt   time.Time `json:"created_at"`
}

// EncryptedNote represents an encrypted note for storage
type EncryptedNote struct {
	ID            string    `json:"id"`
	EncryptedData string    `json:"encrypted_data"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
