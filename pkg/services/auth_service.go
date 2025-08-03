package services

import (
	"gote/pkg/auth"
	"gote/pkg/config"
	"gote/pkg/errors"
)

// AuthService handles authentication business logic with enhanced security
type AuthService struct {
	authManager   *auth.Manager
	secureManager *auth.SecureManager
	config        *config.Config
}

// NewAuthService creates a new authentication service
func NewAuthService(authManager *auth.Manager, config *config.Config) *AuthService {
	return &AuthService{
		authManager:   authManager,
		secureManager: auth.NewSecureManager(config.PasswordHashPath),
		config:        config,
	}
}

// IsPasswordSet checks if a password is configured (works with both legacy and secure methods)
func (s *AuthService) IsPasswordSet() bool {
	if s.secureManager != nil {
		method, err := s.secureManager.DetectPasswordMethod()
		return err == nil && method != "none"
	}
	return s.authManager != nil && !s.authManager.IsFirstTimeSetup()
}

// SetPassword stores a new password hash with enhanced security
func (s *AuthService) SetPassword(password string) ([]byte, error) {
	// Validate password
	validator := errors.NewValidator()
	if result := validator.ValidatePassword(password); !result.IsValid {
		err := result.GetFirstError()
		err.Log()
		return nil, err
	}

	// Use secure PBKDF2 method for new passwords
	retryHandler := errors.NewRetryHandler(3)
	var key []byte

	err := retryHandler.Execute(func() error {
		if err := s.secureManager.StorePasswordHashSecure(password); err != nil {
			return errors.Wrap(err, errors.ErrTypeFileSystem, "PASSWORD_STORE_FAILED",
				"failed to store password hash").
				WithUserMessage("Unable to save password. Please try again").
				WithRetryable(true)
		}

		// Get the secure encryption key
		var success bool
		key, success = s.secureManager.GetEncryptionKey(password)
		if !success {
			return errors.New(errors.ErrTypeAuth, "KEY_DERIVATION_FAILED",
				"failed to derive encryption key").
				WithUserMessage("Unable to generate encryption key")
		}

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

// VerifyPassword checks if the provided password is correct with enhanced security
func (s *AuthService) VerifyPassword(password string) ([]byte, bool) {
	// Basic validation
	if password == "" {
		err := errors.New(errors.ErrTypeValidation, "PASSWORD_EMPTY", "password cannot be empty").
			WithUserMessage("Please enter your password")
		err.Log()
		return nil, false
	}

	// Try secure verification first, then fallback to legacy
	if key, success := s.secureManager.VerifyPasswordSecure(password); success {
		// Check if we should migrate from legacy to secure
		if !s.secureManager.IsSecureMethod() {
			// Password verified with legacy method - migrate to secure
			if err := s.secureManager.MigrateToSecure(password); err != nil {
				// Log migration error but don't fail authentication
				migrationErr := errors.Wrap(err, errors.ErrTypeAuth, "MIGRATION_FAILED",
					"failed to migrate to secure password storage").
					WithUserMessage("Password migration failed")
				migrationErr.Log()
			}
		}
		return key, true
	}

	// Log authentication failure for security monitoring
	err := errors.ErrInvalidPassword.WithContext("timestamp", "auth_attempt")
	err.Log()
	return nil, false
}

// ResetApplication removes password storage with proper error handling
func (s *AuthService) ResetApplication() error {
	retryHandler := errors.NewRetryHandler(2)

	err := retryHandler.Execute(func() error {
		if err := s.secureManager.RemovePasswordHashSecure(); err != nil {
			return errors.Wrap(err, errors.ErrTypeFileSystem, "RESET_FAILED",
				"failed to remove password configuration").
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

// Security enhancement methods

// IsUsingSecureMethod checks if the current password storage uses secure PBKDF2
func (s *AuthService) IsUsingSecureMethod() bool {
	return s.secureManager.IsSecureMethod()
}

// GetSecurityInfo returns information about the current security configuration
func (s *AuthService) GetSecurityInfo() map[string]interface{} {
	method, err := s.secureManager.DetectPasswordMethod()
	if err != nil {
		return map[string]interface{}{
			"method": "unknown",
			"secure": false,
			"error":  err.Error(),
		}
	}

	return map[string]interface{}{
		"method":          method,
		"secure":          method == "pbkdf2",
		"recommendations": s.getSecurityRecommendations(method),
	}
}

// getSecurityRecommendations provides security recommendations based on current method
func (s *AuthService) getSecurityRecommendations(method string) []string {
	recommendations := []string{}

	switch method {
	case "legacy":
		recommendations = append(recommendations,
			"Consider changing your password to upgrade to enhanced security (PBKDF2)")
	case "none":
		recommendations = append(recommendations,
			"Set up a strong password to secure your notes")
	case "pbkdf2":
		recommendations = append(recommendations,
			"Your password is protected with enhanced security (PBKDF2)")
	}

	return recommendations
}
