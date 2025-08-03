package errors

import (
	"os"
	"path/filepath"
	"strings"
)

// ValidationResult holds validation results
type ValidationResult struct {
	IsValid bool
	Errors  []*AppError
}

// AddError adds an error to the validation result
func (vr *ValidationResult) AddError(err *AppError) {
	vr.IsValid = false
	vr.Errors = append(vr.Errors, err)
}

// GetFirstError returns the first error or nil
func (vr *ValidationResult) GetFirstError() *AppError {
	if len(vr.Errors) > 0 {
		return vr.Errors[0]
	}
	return nil
}

// Validator provides validation utilities
type Validator struct{}

// NewValidator creates a new validator
func NewValidator() *Validator {
	return &Validator{}
}

// ValidatePassword validates password requirements
func (v *Validator) ValidatePassword(password string) *ValidationResult {
	result := &ValidationResult{IsValid: true}

	if len(password) < 6 {
		result.AddError(ErrPasswordTooShort)
	}

	if strings.TrimSpace(password) == "" {
		result.AddError(New(ErrTypeValidation, "PASSWORD_EMPTY", "password cannot be empty").
			WithUserMessage("Password cannot be empty"))
	}

	return result
}

// ValidatePasswordMatch validates that passwords match
func (v *Validator) ValidatePasswordMatch(password, confirmPassword string) *ValidationResult {
	result := &ValidationResult{IsValid: true}

	if password != confirmPassword {
		result.AddError(ErrPasswordMismatch)
	}

	return result
}

// ValidateFilePath validates file path requirements
func (v *Validator) ValidateFilePath(path string, shouldExist bool) *ValidationResult {
	result := &ValidationResult{IsValid: true}

	if strings.TrimSpace(path) == "" {
		result.AddError(New(ErrTypeValidation, "PATH_EMPTY", "path cannot be empty").
			WithUserMessage("File path cannot be empty"))
		return result
	}

	// Check if path is absolute
	if !filepath.IsAbs(path) {
		result.AddError(New(ErrTypeValidation, "PATH_NOT_ABSOLUTE", "path must be absolute").
			WithUserMessage("Please provide a complete file path"))
		return result
	}

	// Check existence if required
	if shouldExist {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			result.AddError(New(ErrTypeFileSystem, "PATH_NOT_EXISTS", "path does not exist").
				WithUserMessage("The specified path does not exist").
				WithContext("path", path))
		}
	}

	return result
}

// ValidateDirectoryPath validates directory path and permissions
func (v *Validator) ValidateDirectoryPath(path string) *ValidationResult {
	result := &ValidationResult{IsValid: true}

	if strings.TrimSpace(path) == "" {
		result.AddError(New(ErrTypeValidation, "PATH_EMPTY", "directory path cannot be empty").
			WithUserMessage("Directory path cannot be empty"))
		return result
	}

	// Check if we can create the directory
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		result.AddError(Wrap(err, ErrTypeFileSystem, "DIR_CREATE_FAILED", "cannot create directory").
			WithUserMessage("Cannot create directory. Check permissions").
			WithContext("path", path))
	}

	return result
}

// ValidateNoteContent validates note content
func (v *Validator) ValidateNoteContent(content string) *ValidationResult {
	result := &ValidationResult{IsValid: true}

	// Allow empty content for now, but trim whitespace
	content = strings.TrimSpace(content)

	// Check for extremely large content (> 1MB)
	if len(content) > 1024*1024 {
		result.AddError(New(ErrTypeValidation, "CONTENT_TOO_LARGE", "note content too large").
			WithUserMessage("Note content is too large. Maximum size is 1MB").
			WithContext("size", len(content)))
	}

	return result
}

// ValidateNoteID validates note ID format
func (v *Validator) ValidateNoteID(id string) *ValidationResult {
	result := &ValidationResult{IsValid: true}

	if strings.TrimSpace(id) == "" {
		result.AddError(New(ErrTypeValidation, "ID_EMPTY", "note ID cannot be empty").
			WithUserMessage("Note ID is required"))
		return result
	}

	// Basic UUID format validation (simple check)
	if len(id) < 8 {
		result.AddError(New(ErrTypeValidation, "ID_INVALID", "invalid note ID format").
			WithUserMessage("Invalid note ID format"))
	}

	return result
}
