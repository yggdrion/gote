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
	defer func() {
		if cerr := zipFile.Close(); cerr != nil {
			fmt.Printf("[ERROR] zipFile.Close: %v\n", cerr)
		}
	}()

	zipWriter := zip.NewWriter(zipFile)
	defer func() {
		if cerr := zipWriter.Close(); cerr != nil {
			fmt.Printf("[ERROR] zipWriter.Close: %v\n", cerr)
		}
	}()

	noteFiles, err := filepath.Glob(filepath.Join(notesDir, "*.json"))
	if err != nil {
		return "", err
	}
	folderName := "backup-" + timestamp + "/"
	for _, file := range noteFiles {
		f, err := os.Open(file)
		if err != nil {
			continue // skip unreadable files
		}
		defer func(f *os.File) {
			if cerr := f.Close(); cerr != nil {
				fmt.Printf("[ERROR] f.Close: %v\n", cerr)
			}
		}(f)
		w, err := zipWriter.Create(folderName + filepath.Base(file))
		if err != nil {
			continue
		}
		if _, err := io.Copy(w, f); err != nil {
			continue
		}
	}
	return zipPath, nil
}
