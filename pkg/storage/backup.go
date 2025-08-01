package storage

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// BackupNotes creates a zip archive of all notes in the notes directory.
func BackupNotes(notesDir string, _ string) (string, error) {
	// Ensure notes directory exists
	if err := os.MkdirAll(notesDir, 0755); err != nil {
		return "", err
	}
	timestamp := time.Now().Format("20060102-1504")
	zipPath := filepath.Join(notesDir, "backup-"+timestamp+".zip")

	// Remove old zip if exists
	if _, err := os.Stat(zipPath); err == nil {
		if err := os.Remove(zipPath); err != nil {
			return "", err
		}
	}

	zipFile, err := os.Create(zipPath)
	if err != nil {
		return "", err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	noteFiles, err := filepath.Glob(filepath.Join(notesDir, "*.json"))
	if err != nil {
		return "", err
	}
	notesFolder := "notes/"
	for _, file := range noteFiles {
		f, err := os.Open(file)
		if err != nil {
			continue // skip unreadable files
		}
		defer f.Close()
		w, err := zipWriter.Create(notesFolder + filepath.Base(file))
		if err != nil {
			continue
		}
		if _, err := io.Copy(w, f); err != nil {
			continue
		}
	}
	fmt.Printf("[DEBUG] Backup zip created at: %s\n", zipPath)
	return zipPath, nil
}
