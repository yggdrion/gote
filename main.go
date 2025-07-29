package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

// Session management
type Session struct {
	key       []byte
	expiresAt time.Time
}

// Configuration management
type Config struct {
	NotesPath        string `json:"notesPath"`
	PasswordHashPath string `json:"passwordHashPath"`
}

var currentConfig *Config

var sessions = make(map[string]*Session)
var sessionTimeout = 30 * time.Minute

// Encryption functions
func encrypt(plaintext string, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func decrypt(ciphertext string, key []byte) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext_bytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext_bytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

func deriveKey(password string) []byte {
	hash := sha256.Sum256([]byte(password))
	return hash[:]
}

// Cross-platform path helpers
func getDefaultDataPath() string {
	// Default to "./data" in current directory
	return "./data"
}

func getDefaultPasswordHashPath() string {
	// Get user's home directory
	currentUser, err := user.Current()
	if err != nil {
		// Fallback to current directory if home not available
		return filepath.Join("./data", ".password_hash")
	}

	var configDir string
	if runtime.GOOS == "windows" {
		// On Windows, use %APPDATA%
		configDir = os.Getenv("APPDATA")
		if configDir == "" {
			configDir = currentUser.HomeDir
		}
	} else {
		// On Linux/Unix, use $HOME/.config
		configDir = os.Getenv("XDG_CONFIG_HOME")
		if configDir == "" {
			configDir = filepath.Join(currentUser.HomeDir, ".config")
		}
	}

	// Create the config directory if it doesn't exist
	configPath := filepath.Join(configDir, "gote")
	if err := os.MkdirAll(configPath, 0755); err != nil {
		// If we can't create the config directory, fallback to data directory
		return filepath.Join("./data", ".password_hash")
	}

	return filepath.Join(configPath, "gote_password_hash")
}

func getConfigFilePath() string {
	// Get user's home directory for config file
	currentUser, err := user.Current()
	if err != nil {
		// Fallback to current directory
		return "./config.json"
	}

	var configDir string
	if runtime.GOOS == "windows" {
		configDir = os.Getenv("APPDATA")
		if configDir == "" {
			configDir = currentUser.HomeDir
		}
	} else {
		configDir = os.Getenv("XDG_CONFIG_HOME")
		if configDir == "" {
			configDir = filepath.Join(currentUser.HomeDir, ".config")
		}
	}

	configPath := filepath.Join(configDir, "gote")
	if err := os.MkdirAll(configPath, 0755); err != nil {
		return "./config.json"
	}

	return filepath.Join(configPath, "config.json")
}

// Configuration functions
func loadConfig() *Config {
	config := &Config{
		NotesPath:        getDefaultDataPath(),
		PasswordHashPath: getDefaultPasswordHashPath(),
	}

	configFile := getConfigFilePath()
	if data, err := os.ReadFile(configFile); err == nil {
		json.Unmarshal(data, config)
	}

	// Ensure the data directory exists
	if err := os.MkdirAll(config.NotesPath, 0755); err != nil {
		log.Printf("Warning: Could not create data directory %s: %v", config.NotesPath, err)
	}

	// Ensure the password hash directory exists
	passwordDir := filepath.Dir(config.PasswordHashPath)
	if err := os.MkdirAll(passwordDir, 0755); err != nil {
		log.Printf("Warning: Could not create password hash directory %s: %v", passwordDir, err)
	}

	return config
}

func saveConfig(config *Config) error {
	configFile := getConfigFilePath()

	// Ensure the config directory exists
	configDir := filepath.Dir(configFile)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configFile, data, 0644)
}

func getPasswordHashFile() string {
	if currentConfig != nil {
		return currentConfig.PasswordHashPath
	}
	return filepath.Join("./data", ".password_hash")
}

func isFirstTimeSetup() bool {
	_, err := os.Stat(getPasswordHashFile())
	return os.IsNotExist(err)
}

func storePasswordHash(password string) error {
	// Create a double hash - one for verification, one for encryption key
	verificationHash := sha256.Sum256([]byte(password + "verification"))
	hashFile := getPasswordHashFile()

	// Ensure password hash directory exists
	hashDir := filepath.Dir(hashFile)
	if err := os.MkdirAll(hashDir, 0755); err != nil {
		return fmt.Errorf("failed to create password hash directory: %v", err)
	}

	return os.WriteFile(hashFile, verificationHash[:], 0600)
}

func verifyPassword(password string) bool {
	if isFirstTimeSetup() {
		return false
	}

	storedHash, err := os.ReadFile(getPasswordHashFile())
	if err != nil {
		return false
	}

	verificationHash := sha256.Sum256([]byte(password + "verification"))
	return string(storedHash) == string(verificationHash[:])
}

func generateSessionID() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)
}

// generateShortUUID generates a short UUID (8 characters) for file names
func generateShortUUID() string {
	fullUUID := uuid.New().String()
	// Take first 8 characters for a short but still unique identifier
	return strings.ReplaceAll(fullUUID[:8], "-", "")
}

func isAuthenticated(r *http.Request) *Session {
	cookie, err := r.Cookie("session")
	if err != nil {
		return nil
	}

	session, exists := sessions[cookie.Value]
	if !exists || time.Now().After(session.expiresAt) {
		// Clean up expired session
		if exists {
			delete(sessions, cookie.Value)
		}
		return nil
	}

	// Extend session
	session.expiresAt = time.Now().Add(sessionTimeout)
	return session
}

// Encrypted Note structure for storage
type EncryptedNote struct {
	ID            string    `json:"id"`
	EncryptedData string    `json:"encrypted_data"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Note struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type NoteStore struct {
	dataDir      string
	notes        map[string]*Note
	mutex        sync.RWMutex
	watcher      *fsnotify.Watcher
	key          []byte
	lastSync     time.Time
	fileModTimes map[string]time.Time
}

func NewNoteStore(dataDir string) *NoteStore {
	store := &NoteStore{
		dataDir:      dataDir,
		notes:        make(map[string]*Note),
		fileModTimes: make(map[string]time.Time),
	}

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatal("Failed to create data directory:", err)
	}

	// Initialize file system watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("Warning: Could not create file watcher: %v", err)
	} else {
		store.watcher = watcher

		// Add data directory to watcher
		if err := watcher.Add(dataDir); err != nil {
			log.Printf("Warning: Could not watch data directory: %v", err)
		}
	}

	// Notes will be loaded when user authenticates
	return store
}

// startWatching starts the file system watcher goroutine
func (s *NoteStore) startWatching() {
	if s.watcher == nil {
		return
	}

	go func() {
		for {
			select {
			case event, ok := <-s.watcher.Events:
				if !ok {
					return
				}

				// Only process .json files
				if !strings.HasSuffix(event.Name, ".json") {
					continue
				}

				log.Printf("File event: %s %s", event.Op, event.Name)

				switch {
				case event.Op&fsnotify.Create == fsnotify.Create:
					s.handleFileCreate(event.Name)
				case event.Op&fsnotify.Write == fsnotify.Write:
					s.handleFileWrite(event.Name)
				case event.Op&fsnotify.Remove == fsnotify.Remove:
					s.handleFileRemove(event.Name)
				case event.Op&fsnotify.Rename == fsnotify.Rename:
					s.handleFileRemove(event.Name)
				}

			case err, ok := <-s.watcher.Errors:
				if !ok {
					return
				}
				log.Printf("Watcher error: %v", err)
			}
		}
	}()
}

// handleFileCreate handles new file creation
func (s *NoteStore) handleFileCreate(filePath string) {
	s.handleFileWrite(filePath)
}

// handleFileWrite handles file modifications
func (s *NoteStore) handleFileWrite(filePath string) {
	if s.key == nil {
		return // Not authenticated yet
	}

	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		log.Printf("Error getting file info for %s: %v", filePath, err)
		return
	}

	// Check if this is a change we need to process
	s.mutex.Lock()
	lastModTime, exists := s.fileModTimes[filePath]
	currentModTime := fileInfo.ModTime()

	// If we already have this modification time, skip (probably our own write)
	if exists && !currentModTime.After(lastModTime) {
		s.mutex.Unlock()
		return
	}

	s.fileModTimes[filePath] = currentModTime
	s.mutex.Unlock()

	// Load the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("Error reading changed file %s: %v", filePath, err)
		return
	}

	var encryptedNote EncryptedNote
	if err := json.Unmarshal(data, &encryptedNote); err != nil {
		log.Printf("Error unmarshalling changed file %s: %v", filePath, err)
		return
	}

	// Decrypt the note content
	decryptedContent, err := decrypt(encryptedNote.EncryptedData, s.key)
	if err != nil {
		log.Printf("Error decrypting changed file %s: %v", filePath, err)
		return
	}

	note := &Note{
		ID:        encryptedNote.ID,
		Content:   decryptedContent,
		CreatedAt: encryptedNote.CreatedAt,
		UpdatedAt: encryptedNote.UpdatedAt,
	}

	s.mutex.Lock()
	existingNote, exists := s.notes[note.ID]

	// Only update if the external file is newer than what we have in memory
	if !exists || note.UpdatedAt.After(existingNote.UpdatedAt) {
		s.notes[note.ID] = note
		log.Printf("Updated note %s from external file change", note.ID)
	} else {
		log.Printf("Skipped updating note %s - in-memory version is newer", note.ID)
	}
	s.mutex.Unlock()
}

// handleFileRemove handles file deletion
func (s *NoteStore) handleFileRemove(filePath string) {
	// Extract note ID from filename
	filename := filepath.Base(filePath)
	if !strings.HasSuffix(filename, ".json") {
		return
	}

	noteID := strings.TrimSuffix(filename, ".json")

	s.mutex.Lock()
	delete(s.notes, noteID)
	delete(s.fileModTimes, filePath)
	s.mutex.Unlock()

	log.Printf("Removed note %s due to external file deletion", noteID)
}

// syncFromDisk performs a full sync from disk, useful for resolving conflicts
func (s *NoteStore) syncFromDisk() error {
	if s.key == nil {
		return fmt.Errorf("not authenticated")
	}

	files, err := filepath.Glob(filepath.Join(s.dataDir, "*.json"))
	if err != nil {
		return fmt.Errorf("error reading data directory: %v", err)
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Track which notes exist on disk
	diskNotes := make(map[string]bool)

	for _, file := range files {
		fileInfo, err := os.Stat(file)
		if err != nil {
			log.Printf("Error getting file info for %s: %v", file, err)
			continue
		}

		data, err := os.ReadFile(file)
		if err != nil {
			log.Printf("Error reading file %s: %v", file, err)
			continue
		}

		var encryptedNote EncryptedNote
		if err := json.Unmarshal(data, &encryptedNote); err != nil {
			log.Printf("Error unmarshalling encrypted note from %s: %v", file, err)
			continue
		}

		// Decrypt the note content
		decryptedContent, err := decrypt(encryptedNote.EncryptedData, s.key)
		if err != nil {
			log.Printf("Error decrypting note from %s: %v", file, err)
			continue
		}

		note := &Note{
			ID:        encryptedNote.ID,
			Content:   decryptedContent,
			CreatedAt: encryptedNote.CreatedAt,
			UpdatedAt: encryptedNote.UpdatedAt,
		}

		diskNotes[note.ID] = true
		s.fileModTimes[file] = fileInfo.ModTime()

		// Update note if it's newer or doesn't exist in memory
		existingNote, exists := s.notes[note.ID]
		if !exists || note.UpdatedAt.After(existingNote.UpdatedAt) {
			s.notes[note.ID] = note
		}
	}

	// Remove notes that no longer exist on disk
	for noteID := range s.notes {
		if !diskNotes[noteID] {
			delete(s.notes, noteID)
		}
	}

	s.lastSync = time.Now()
	return nil
}

func (s *NoteStore) loadNotes(key []byte) error {
	s.mutex.Lock()
	s.key = key
	s.mutex.Unlock()

	// Start file watching
	s.startWatching()

	// Load notes from disk
	if err := s.syncFromDisk(); err != nil {
		return err
	}

	return nil
}

func (s *NoteStore) saveNote(note *Note, key []byte) error {
	// Encrypt the note content
	encryptedContent, err := encrypt(note.Content, key)
	if err != nil {
		return err
	}

	encryptedNote := EncryptedNote{
		ID:            note.ID,
		EncryptedData: encryptedContent,
		CreatedAt:     note.CreatedAt,
		UpdatedAt:     note.UpdatedAt,
	}

	filename := filepath.Join(s.dataDir, fmt.Sprintf("%s.json", note.ID))
	data, err := json.MarshalIndent(encryptedNote, "", "  ")
	if err != nil {
		return err
	}

	// Write the file
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return err
	}

	// Update our modification time tracking to prevent processing our own write
	if fileInfo, err := os.Stat(filename); err == nil {
		s.mutex.Lock()
		s.fileModTimes[filename] = fileInfo.ModTime()
		s.mutex.Unlock()
	}

	return nil
}

func (s *NoteStore) deleteNote(id string) error {
	filename := filepath.Join(s.dataDir, fmt.Sprintf("%s.json", id))

	s.mutex.Lock()
	delete(s.notes, id)
	delete(s.fileModTimes, filename)
	s.mutex.Unlock()

	return os.Remove(filename)
}

func (s *NoteStore) CreateNote(content string, key []byte) (*Note, error) {
	note := &Note{
		ID:        generateShortUUID(),
		Content:   content,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	s.mutex.Lock()
	s.notes[note.ID] = note
	s.mutex.Unlock()

	if err := s.saveNote(note, key); err != nil {
		s.mutex.Lock()
		delete(s.notes, note.ID)
		s.mutex.Unlock()
		return nil, err
	}

	return note, nil
}

func (s *NoteStore) UpdateNote(id string, content string, key []byte) (*Note, error) {
	s.mutex.Lock()
	note, exists := s.notes[id]
	if !exists {
		s.mutex.Unlock()
		return nil, fmt.Errorf("note not found")
	}

	note.Content = content
	note.UpdatedAt = time.Now()
	s.mutex.Unlock()

	if err := s.saveNote(note, key); err != nil {
		return nil, err
	}

	return note, nil
}

func (s *NoteStore) GetNote(id string) (*Note, error) {
	s.mutex.RLock()
	note, exists := s.notes[id]
	s.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("note not found")
	}
	return note, nil
}

func (s *NoteStore) GetAllNotes() []*Note {
	s.mutex.RLock()
	notes := make([]*Note, 0, len(s.notes))
	for _, note := range s.notes {
		notes = append(notes, note)
	}
	s.mutex.RUnlock()

	// Sort by updated time, newest first
	sort.Slice(notes, func(i, j int) bool {
		return notes[i].UpdatedAt.After(notes[j].UpdatedAt)
	})

	return notes
}

func (s *NoteStore) SearchNotes(query string) []*Note {
	var results []*Note
	query = strings.ToLower(query)

	s.mutex.RLock()
	for _, note := range s.notes {
		if strings.Contains(strings.ToLower(note.Content), query) {
			results = append(results, note)
		}
	}
	s.mutex.RUnlock()

	// Sort by updated time, newest first
	sort.Slice(results, func(i, j int) bool {
		return results[i].UpdatedAt.After(results[j].UpdatedAt)
	})

	return results
}

func (s *NoteStore) DeleteNote(id string) error {
	s.mutex.Lock()
	_, exists := s.notes[id]
	s.mutex.Unlock()

	if !exists {
		return fmt.Errorf("note not found")
	}
	return s.deleteNote(id)
}

// Close cleans up the file watcher
func (s *NoteStore) Close() error {
	if s.watcher != nil {
		return s.watcher.Close()
	}
	return nil
}

// RefreshFromDisk forces a full refresh from disk, useful for conflict resolution
func (s *NoteStore) RefreshFromDisk() error {
	return s.syncFromDisk()
}

var store *NoteStore

// Authentication handlers
func loginHandler(w http.ResponseWriter, r *http.Request) {
	isFirstTime := isFirstTimeSetup()

	tmpl, err := template.ParseFiles("./static/login.html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		log.Printf("Template parsing error: %v", err)
		return
	}

	data := struct {
		Error       string
		IsFirstTime bool
	}{
		Error:       r.URL.Query().Get("error"),
		IsFirstTime: isFirstTime,
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Template execution error", http.StatusInternalServerError)
		log.Printf("Template execution error: %v", err)
	}
}

func authHandler(w http.ResponseWriter, r *http.Request) {
	password := r.FormValue("password")
	if password == "" {
		http.Redirect(w, r, "/login?error=Password required", http.StatusSeeOther)
		return
	}

	// Handle first-time setup
	if isFirstTimeSetup() {
		confirmPassword := r.FormValue("confirm_password")
		if confirmPassword == "" {
			http.Redirect(w, r, "/login?error=Please confirm your password", http.StatusSeeOther)
			return
		}

		if password != confirmPassword {
			http.Redirect(w, r, "/login?error=Passwords do not match", http.StatusSeeOther)
			return
		}

		if len(password) < 6 {
			http.Redirect(w, r, "/login?error=Password must be at least 6 characters", http.StatusSeeOther)
			return
		}

		// Store the password hash
		if err := storePasswordHash(password); err != nil {
			http.Redirect(w, r, "/login?error=Failed to create password", http.StatusSeeOther)
			return
		}
	} else {
		// Verify existing password
		if !verifyPassword(password) {
			http.Redirect(w, r, "/login?error=Invalid password", http.StatusSeeOther)
			return
		}
	}

	key := deriveKey(password)

	// Try to load notes with this password
	if err := store.loadNotes(key); err != nil {
		// For existing setup, this should not fail if password is correct
		if !isFirstTimeSetup() {
			http.Redirect(w, r, "/login?error=Failed to decrypt notes", http.StatusSeeOther)
			return
		}
		// For first-time setup, it's expected that there are no notes to load
	}

	// Create session
	sessionID := generateSessionID()
	sessions[sessionID] = &Session{
		key:       key,
		expiresAt: time.Now().Add(sessionTimeout),
	}

	// Set session cookie
	cookie := &http.Cookie{
		Name:     "session",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err == nil {
		delete(sessions, cookie.Value)
	}

	// Clear session cookie
	cookie = &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	}
	http.SetCookie(w, cookie)

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session := isAuthenticated(r)
		if session == nil {
			if r.Header.Get("Content-Type") == "application/json" ||
				strings.HasPrefix(r.URL.Path, "/api/") {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

func main() {
	currentConfig = loadConfig()

	// Log configuration paths for user information
	log.Printf("Configuration loaded:")
	log.Printf("  Notes directory: %s", currentConfig.NotesPath)
	log.Printf("  Password hash file: %s", currentConfig.PasswordHashPath)
	log.Printf("  Config file: %s", getConfigFilePath())

	store = NewNoteStore(currentConfig.NotesPath)

	r := chi.NewRouter()

	// Add middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Serve static files
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))

	// Authentication routes (no auth required)
	r.Get("/login", loginHandler)
	r.Post("/auth", authHandler)
	r.Post("/logout", logoutHandler)

	// Protected routes
	r.Get("/", requireAuth(indexHandler))

	// Protected API routes
	r.Route("/api", func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				session := isAuthenticated(r)
				if session == nil {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
				next.ServeHTTP(w, r)
			})
		})
		r.Get("/notes", apiGetNotesHandler)
		r.Post("/notes", apiCreateNoteHandler)
		r.Get("/notes/{id}", apiGetNoteHandler)
		r.Put("/notes/{id}", apiUpdateNoteHandler)
		r.Delete("/notes/{id}", apiDeleteNoteHandler)
		r.Get("/search", apiSearchHandler)
		r.Get("/settings", apiGetSettingsHandler)
		r.Post("/settings", apiSettingsHandler)
		r.Post("/sync", apiSyncHandler)
	})

	fmt.Println("Server starting on :8080")

	// Handle graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("Shutting down gracefully...")
		if store != nil {
			store.Close()
		}
		os.Exit(0)
	}()

	log.Fatal(http.ListenAndServe(":8080", r))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	var notes []*Note

	if query != "" {
		notes = store.SearchNotes(query)
	} else {
		notes = store.GetAllNotes()
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
		Notes []*Note
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

// API Handlers
func apiGetNotesHandler(w http.ResponseWriter, r *http.Request) {
	notes := store.GetAllNotes()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notes)
}

func apiCreateNoteHandler(w http.ResponseWriter, r *http.Request) {
	session := isAuthenticated(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Content string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	note, err := store.CreateNote(req.Content, session.key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(note)
}

func apiGetNoteHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Invalid note ID", http.StatusBadRequest)
		return
	}

	note, err := store.GetNote(id)
	if err != nil {
		http.Error(w, "Note not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(note)
}

func apiUpdateNoteHandler(w http.ResponseWriter, r *http.Request) {
	session := isAuthenticated(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Invalid note ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Content string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	note, err := store.UpdateNote(id, req.Content, session.key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(note)
}

func apiDeleteNoteHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Invalid note ID", http.StatusBadRequest)
		return
	}

	if err := store.DeleteNote(id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func apiSearchHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Missing query parameter", http.StatusBadRequest)
		return
	}

	notes := store.SearchNotes(query)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notes)
}

func apiGetSettingsHandler(w http.ResponseWriter, r *http.Request) {
	session := isAuthenticated(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(currentConfig)
}

func apiSettingsHandler(w http.ResponseWriter, r *http.Request) {
	session := isAuthenticated(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req Config
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate and set default paths if empty
	if req.NotesPath == "" {
		req.NotesPath = getDefaultDataPath()
	}
	if req.PasswordHashPath == "" {
		req.PasswordHashPath = getDefaultPasswordHashPath()
	}

	// Ensure directories exist before saving config
	if err := os.MkdirAll(req.NotesPath, 0755); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create notes directory: %v", err), http.StatusBadRequest)
		return
	}

	passwordDir := filepath.Dir(req.PasswordHashPath)
	if err := os.MkdirAll(passwordDir, 0755); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create password hash directory: %v", err), http.StatusBadRequest)
		return
	}

	// Update global config
	currentConfig.NotesPath = req.NotesPath
	currentConfig.PasswordHashPath = req.PasswordHashPath

	// Save config to file
	if err := saveConfig(currentConfig); err != nil {
		http.Error(w, "Failed to save configuration", http.StatusInternalServerError)
		return
	}

	// Update note store with new path
	if currentConfig.NotesPath != req.NotesPath {
		// Close existing watcher
		if store != nil {
			store.Close()
		}

		// Create new note store with new path
		store = NewNoteStore(currentConfig.NotesPath)

		// Reload notes if user is authenticated
		if session := isAuthenticated(r); session != nil {
			if err := store.loadNotes(session.key); err != nil {
				log.Printf("Warning: Could not reload notes after path change: %v", err)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Settings saved successfully",
	})
}

func apiSyncHandler(w http.ResponseWriter, r *http.Request) {
	session := isAuthenticated(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := store.RefreshFromDisk(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to sync from disk: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Successfully synced from disk",
	})
}
