package services

import (
	"gote/pkg/auth"
	"gote/pkg/config"
	"gote/pkg/crypto"
)

// AuthService handles authentication business logic
type AuthService struct {
	authManager *auth.Manager
	config      *config.Config
}

// NewAuthService creates a new authentication service
func NewAuthService(authManager *auth.Manager, config *config.Config) *AuthService {
	return &AuthService{
		authManager: authManager,
		config:      config,
	}
}

// IsPasswordSet checks if a password is configured
func (s *AuthService) IsPasswordSet() bool {
	return s.authManager != nil && !s.authManager.IsFirstTimeSetup()
}

// SetPassword stores a new password hash
func (s *AuthService) SetPassword(password string) ([]byte, error) {
	if err := s.authManager.StorePasswordHash(password); err != nil {
		return nil, err
	}
	return crypto.DeriveKey(password), nil
}

// VerifyPassword checks if the provided password is correct
func (s *AuthService) VerifyPassword(password string) ([]byte, bool) {
	if s.authManager.VerifyPassword(password) {
		return crypto.DeriveKey(password), true
	}
	return nil, false
}

// ResetApplication removes the password hash file
func (s *AuthService) ResetApplication() error {
	return s.authManager.RemovePasswordHash()
}