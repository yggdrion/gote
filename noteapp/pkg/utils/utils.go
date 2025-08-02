package utils

import (
	"crypto/rand"
	"encoding/base64"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// IsValidShortHashFilename checks if the filename matches the expected short hash pattern
func IsValidShortHashFilename(filename string) bool {
	// Remove .json extension if present
	filename = strings.TrimSuffix(filename, ".json")

	// Check if it's exactly 8 characters and all hexadecimal
	if len(filename) != 8 {
		return false
	}

	// Check if all characters are valid hexadecimal
	matched, err := regexp.MatchString("^[0-9a-fA-F]{8}$", filename)
	if err != nil {
		return false
	}

	return matched
}

// GenerateSessionID generates a secure random session ID
func GenerateSessionID() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		// If random read fails, return an empty string for safety
		return ""
	}
	return base64.URLEncoding.EncodeToString(bytes)
}

// GenerateShortUUID generates a short UUID (8 characters) for file names
func GenerateShortUUID() string {
	fullUUID := uuid.New().String()
	// Take first 8 characters for a short but still unique identifier
	return strings.ReplaceAll(fullUUID[:8], "-", "")
}
