package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"gote/pkg/auth"
	"gote/pkg/config"
	"gote/pkg/handlers"
	"gote/pkg/middleware"
	"gote/pkg/storage"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Log configuration paths for user information
	log.Printf("Configuration loaded:")
	log.Printf("  Notes directory: %s", cfg.NotesPath)
	log.Printf("  Password hash file: %s", cfg.PasswordHashPath)
	log.Printf("  Config file: %s", config.GetConfigFilePath())

	// Initialize components
	authManager := auth.NewManager(cfg.PasswordHashPath)
	store := storage.NewNoteStore(cfg.NotesPath)

	// Initialize handlers
	authHandlers := handlers.NewAuthHandlers(authManager, store)
	webHandlers := handlers.NewWebHandlers(store, authManager)
	apiHandlers := handlers.NewAPIHandlers(store, authManager, cfg)

	// Create router
	r := chi.NewRouter()

	// Add middleware
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)

	// Serve static files
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))

	// Authentication routes (no auth required)
	r.Get("/login", authHandlers.LoginHandler)
	r.Post("/auth", authHandlers.AuthHandler)
	r.Post("/logout", authHandlers.LogoutHandler)

	// Password reset route (no auth required)
	r.Post("/reset-password", func(w http.ResponseWriter, r *http.Request) {
		// Delete the password hash file
		if err := os.Remove(cfg.PasswordHashPath); err != nil {
			if os.IsNotExist(err) {
				w.WriteHeader(http.StatusOK)
				if _, err := w.Write([]byte("Password already reset.")); err != nil {
					log.Printf("Failed to write response: %v", err)
				}
				return
			}
			log.Printf("Failed to delete password hash file: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			if _, err := w.Write([]byte("Failed to reset password.")); err != nil {
				log.Printf("Failed to write response: %v", err)
			}
			return
		}
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("Password reset. Please set a new password.")); err != nil {
			log.Printf("Failed to write response: %v", err)
		}
	})

	// Protected routes
	requireAuth := middleware.RequireAuth(authManager)
	r.Get("/", requireAuth(webHandlers.IndexHandler))

	// Protected API routes
	r.Route("/api", func(r chi.Router) {
		r.Use(middleware.RequireAuthAPI(authManager))
		r.Get("/notes", apiHandlers.GetNotesHandler)
		r.Post("/notes", apiHandlers.CreateNoteHandler)
		r.Get("/notes/{id}", apiHandlers.GetNoteHandler)
		r.Put("/notes/{id}", apiHandlers.UpdateNoteHandler)
		r.Delete("/notes/{id}", apiHandlers.DeleteNoteHandler)
		r.Get("/search", apiHandlers.SearchHandler)
		r.Get("/settings", apiHandlers.GetSettingsHandler)
		r.Post("/settings", apiHandlers.SettingsHandler)
		r.Post("/sync", apiHandlers.SyncHandler)
		r.Post("/change-password", apiHandlers.ChangePasswordHandler)
		// Add manual backup endpoint
		r.Post("/backup", apiHandlers.BackupHandler)
	})

	log.Println("Server starting on :8080")

	// Handle graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("Shutting down gracefully...")
		if store != nil {
			if err := store.Close(); err != nil {
				log.Printf("Error closing store: %v", err)
			}
		}
		os.Exit(0)
	}()

	log.Fatal(http.ListenAndServe(":8080", r))
}
