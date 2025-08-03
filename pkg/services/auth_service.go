package services

import (
	"gote/pkg/auth"
	"gote/pkg/config"
	"gote/pkg/crypto"
	"gote/pkg/errors"
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

// SetPassword stores a new password hash with validation
func (s *AuthService) SetPassword(password string) ([]byte, error) {
	// Validate password
	validator := errors.NewValidator()
	if result := validator.ValidatePassword(password); !result.IsValid {
		err := result.GetFirstError()
		err.Log()
		return nil, err
	}

	// Attempt to store with retry logic for transient failures
	retryHandler := errors.NewRetryHandler(3)
	var key []byte

	err := retryHandler.Execute(func() error {
		if err := s.authManager.StorePasswordHash(password); err != nil {
			return errors.Wrap(err, errors.ErrTypeFileSystem, "PASSWORD_STORE_FAILED",
				"failed to store password hash").
				WithUserMessage("Unable to save password. Please try again").
				WithRetryable(true)
		}
		key = crypto.DeriveKey(password)
		return nil
	})

	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			appErr.Log()
			return nil, appErr
		}
		return nil, err
	}

	return key, nil
}

// VerifyPassword checks if the provided password is correct
func (s *AuthService) VerifyPassword(password string) ([]byte, bool) {
	// Basic validation
	if password == "" {
		err := errors.New(errors.ErrTypeValidation, "PASSWORD_EMPTY", "password cannot be empty").
			WithUserMessage("Please enter your password")
		err.Log()
		return nil, false
	}

	if s.authManager.VerifyPassword(password) {
		return crypto.DeriveKey(password), true
	}

	// Log authentication failure for security monitoring
	err := errors.ErrInvalidPassword.WithContext("timestamp", "auth_attempt")
	err.Log()
	return nil, false
}

// ResetApplication removes the password hash file with proper error handling
func (s *AuthService) ResetApplication() error {
	retryHandler := errors.NewRetryHandler(2)

	err := retryHandler.Execute(func() error {
		if err := s.authManager.RemovePasswordHash(); err != nil {
			return errors.Wrap(err, errors.ErrTypeFileSystem, "RESET_FAILED",
				"failed to remove password hash").
				WithUserMessage("Unable to reset application. Please try again").
				WithRetryable(true)
		}
		return nil
	})

	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			appErr.Log()
			return appErr
		}
		return err
	}

	return nil
}
