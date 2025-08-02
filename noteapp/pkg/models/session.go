package models

import "time"

// Session represents a user session with encryption key
type Session struct {
	Key       []byte    `json:"-"` // Don't serialize the key
	ExpiresAt time.Time `json:"expires_at"`
}
