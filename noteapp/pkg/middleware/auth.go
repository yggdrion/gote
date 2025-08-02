package middleware

import (
	"net/http"
	"strings"

	"gote/pkg/models"
)

// AuthManager interface for authentication operations
type AuthManager interface {
	IsAuthenticated(r *http.Request) *models.Session
}

// RequireAuth creates a middleware that requires authentication
func RequireAuth(authManager AuthManager) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			session := authManager.IsAuthenticated(r)
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
}

// RequireAuthAPI creates middleware for API routes that require authentication
func RequireAuthAPI(authManager AuthManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session := authManager.IsAuthenticated(r)
			if session == nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
