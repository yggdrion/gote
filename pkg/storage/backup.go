package storage

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// BackupNotes creates a zip archive of all notes in the notes directory.
func BackupNotes(notesDir string, _ string) (string, error) {
	// Ensure notes directory exists
	if err := os.MkdirAll(notesDir, 0755); err != nil {
		return "", err
	}
	// Create backups subdirectory under notesDir
	backupsDir := filepath.Join(notesDir, "backups")
	if err := os.MkdirAll(backupsDir, 0755); err != nil {
		return "", err
	}

	timestamp := time.Now().Format("20060102-1504")
	zipPath := filepath.Join(backupsDir, "backup-"+timestamp+".zip")

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

	folderName := "backup-" + timestamp + "/"

	// Resolve backupsDir absolute path for safety checks
	absBackupsDir, _ := filepath.Abs(backupsDir)

	// Helper to add a single file with a relative path under the backup folder
	addFile := func(absPath, rel string) error {
		// Never include anything from the backups directory
		if absPath != "" {
			if absAbsPath, err := filepath.Abs(absPath); err == nil {
				if relToBackups, err := filepath.Rel(absBackupsDir, absAbsPath); err == nil {
					if relToBackups == "." || (relToBackups != "" && !strings.HasPrefix(relToBackups, "..")) {
						// absPath is inside backupsDir; skip silently
						return nil
					}
				}
			}
		}
		f, err := os.Open(absPath)
		if err != nil {
			return err
		}
		defer func() {
			if cerr := f.Close(); cerr != nil {
				fmt.Printf("[ERROR] f.Close: %v\n", cerr)
			}
		}()
		w, err := zipWriter.Create(folderName + rel)
		if err != nil {
			return err
		}
		_, err = io.Copy(w, f)
		return err
	}

	// Include note JSON files at root of notesDir (exclude backups directory and zips)
	noteFiles, err := filepath.Glob(filepath.Join(notesDir, "*.json"))
	if err != nil {
		return "", err
	}
	for _, file := range noteFiles {
		// Skip temporary or backup zips
		base := filepath.Base(file)
		if strings.HasPrefix(base, "backup-") && strings.HasSuffix(base, ".zip") {
			continue
		}
		_ = addFile(file, base)
	}

	// Include images directory, if present
	imagesDir := filepath.Join(notesDir, "images")
	if fi, err := os.Stat(imagesDir); err == nil && fi.IsDir() {
		filepath.Walk(imagesDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				return nil
			}
			relPath, relErr := filepath.Rel(notesDir, path)
			if relErr != nil {
				return nil
			}
			_ = addFile(path, relPath)
			return nil
		})
	}

	// Include cross-platform config file if exists
	configPath := filepath.Join(notesDir, ".gote_config.json")
	if _, err := os.Stat(configPath); err == nil {
		_ = addFile(configPath, ".gote_config.json")
	}

	return zipPath, nil
}
