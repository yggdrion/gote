package errors

import (
	"fmt"
	"log"
	"strings"
)

// ErrorType represents different categories of errors
type ErrorType string

const (
	// Authentication errors
	ErrTypeAuth ErrorType = "authentication"
	// File system errors
	ErrTypeFileSystem ErrorType = "filesystem"
	// Configuration errors
	ErrTypeConfig ErrorType = "configuration"
	// Validation errors
	ErrTypeValidation ErrorType = "validation"
	// Encryption/decryption errors
	ErrTypeCrypto ErrorType = "crypto"
	// Network/IO errors
	ErrTypeIO ErrorType = "io"
	// Generic application errors
	ErrTypeApp ErrorType = "application"
)

// AppError represents a structured application error
type AppError struct {
	Type        ErrorType              `json:"type"`
	Code        string                 `json:"code"`
	Message     string                 `json:"message"`
	UserMessage string                 `json:"userMessage"`
	InternalErr error                  `json:"-"`
	Retryable   bool                   `json:"retryable"`
	Context     map[string]interface{} `json:"context,omitempty"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.InternalErr != nil {
		return fmt.Sprintf("[%s:%s] %s: %v", e.Type, e.Code, e.Message, e.InternalErr)
	}
	return fmt.Sprintf("[%s:%s] %s", e.Type, e.Code, e.Message)
}

// GetUserMessage returns a user-friendly error message
func (e *AppError) GetUserMessage() string {
	if e.UserMessage != "" {
		return e.UserMessage
	}
	return e.Message
}

// WithContext adds context information to the error
func (e *AppError) WithContext(key string, value interface{}) *AppError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// Log logs the error with appropriate level
func (e *AppError) Log() {
	contextStr := ""
	if len(e.Context) > 0 {
		var parts []string
		for k, v := range e.Context {
			parts = append(parts, fmt.Sprintf("%s=%v", k, v))
		}
		contextStr = fmt.Sprintf(" [%s]", strings.Join(parts, ", "))
	}

	log.Printf("ERROR [%s:%s] %s%s", e.Type, e.Code, e.Error(), contextStr)
}

// New creates a new AppError
func New(errType ErrorType, code, message string) *AppError {
	return &AppError{
		Type:    errType,
		Code:    code,
		Message: message,
	}
}

// Wrap wraps an existing error with additional context
func Wrap(err error, errType ErrorType, code, message string) *AppError {
	return &AppError{
		Type:        errType,
		Code:        code,
		Message:     message,
		InternalErr: err,
	}
}

// Predefined errors for common scenarios
var (
	// Authentication errors
	ErrNotAuthenticated = New(ErrTypeAuth, "NOT_AUTHENTICATED", "user not authenticated").
				WithUserMessage("Please log in to continue")

	ErrInvalidPassword = New(ErrTypeAuth, "INVALID_PASSWORD", "invalid password").
				WithUserMessage("Invalid password. Please try again")

	ErrPasswordTooShort = New(ErrTypeValidation, "PASSWORD_TOO_SHORT", "password too short").
				WithUserMessage("Password must be at least 6 characters long")

	ErrPasswordMismatch = New(ErrTypeValidation, "PASSWORD_MISMATCH", "passwords do not match").
				WithUserMessage("Passwords do not match. Please try again")

	// File system errors
	ErrNoteNotFound = New(ErrTypeFileSystem, "NOTE_NOT_FOUND", "note not found").
			WithUserMessage("The requested note could not be found")

	ErrDirectoryCreationFailed = New(ErrTypeFileSystem, "DIR_CREATE_FAILED", "failed to create directory").
					WithUserMessage("Unable to create required directory. Check permissions")

	ErrFileReadFailed = New(ErrTypeFileSystem, "FILE_READ_FAILED", "failed to read file").
				WithUserMessage("Unable to read file. It may be corrupted or inaccessible")

	ErrFileWriteFailed = New(ErrTypeFileSystem, "FILE_WRITE_FAILED", "failed to write file").
				WithUserMessage("Unable to save file. Check disk space and permissions")

	// Configuration errors
	ErrConfigLoadFailed = New(ErrTypeConfig, "CONFIG_LOAD_FAILED", "failed to load configuration").
				WithUserMessage("Configuration file could not be loaded. Using defaults")

	ErrConfigSaveFailed = New(ErrTypeConfig, "CONFIG_SAVE_FAILED", "failed to save configuration").
				WithUserMessage("Unable to save settings. Check permissions")

	// Crypto errors
	ErrDecryptionFailed = New(ErrTypeCrypto, "DECRYPT_FAILED", "decryption failed").
				WithUserMessage("Unable to decrypt data. The password may be incorrect")

	ErrEncryptionFailed = New(ErrTypeCrypto, "ENCRYPT_FAILED", "encryption failed").
				WithUserMessage("Unable to encrypt data. Please try again")
)

// WithUserMessage sets a user-friendly message
func (e *AppError) WithUserMessage(msg string) *AppError {
	e.UserMessage = msg
	return e
}

// WithRetryable marks the error as retryable
func (e *AppError) WithRetryable(retryable bool) *AppError {
	e.Retryable = retryable
	return e
}

// IsRetryable checks if the error can be retried
func (e *AppError) IsRetryable() bool {
	return e.Retryable
}

// RetryHandler provides retry functionality for operations
type RetryHandler struct {
	MaxAttempts int
	OnRetry     func(attempt int, err error)
}

// NewRetryHandler creates a new retry handler
func NewRetryHandler(maxAttempts int) *RetryHandler {
	return &RetryHandler{
		MaxAttempts: maxAttempts,
		OnRetry: func(attempt int, err error) {
			log.Printf("Retry attempt %d/%d failed: %v", attempt, maxAttempts, err)
		},
	}
}

// Execute runs a function with retry logic
func (r *RetryHandler) Execute(fn func() error) error {
	var lastErr error

	for attempt := 1; attempt <= r.MaxAttempts; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if appErr, ok := err.(*AppError); ok && !appErr.IsRetryable() {
			return err // Don't retry non-retryable errors
		}

		if attempt < r.MaxAttempts {
			if r.OnRetry != nil {
				r.OnRetry(attempt, err)
			}
		}
	}

	return Wrap(lastErr, ErrTypeApp, "MAX_RETRIES_EXCEEDED",
		fmt.Sprintf("operation failed after %d attempts", r.MaxAttempts)).
		WithUserMessage("Operation failed after multiple attempts. Please try again later")
}
