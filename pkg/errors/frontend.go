package errors

// FrontendError represents an error formatted for frontend consumption
type FrontendError struct {
	Type      string                 `json:"type"`
	Code      string                 `json:"code"`
	Message   string                 `json:"message"`
	Retryable bool                   `json:"retryable"`
	Context   map[string]interface{} `json:"context,omitempty"`
}

// ToFrontendError converts an AppError to a frontend-friendly format
func ToFrontendError(err error) *FrontendError {
	if appErr, ok := err.(*AppError); ok {
		return &FrontendError{
			Type:      string(appErr.Type),
			Code:      appErr.Code,
			Message:   appErr.GetUserMessage(),
			Retryable: appErr.Retryable,
			Context:   appErr.Context,
		}
	}

	// Handle generic errors
	return &FrontendError{
		Type:      string(ErrTypeApp),
		Code:      "GENERIC_ERROR",
		Message:   "An unexpected error occurred. Please try again",
		Retryable: true,
		Context:   map[string]interface{}{"originalError": err.Error()},
	}
}

// ErrorHandler provides utilities for handling errors in the application layer
type ErrorHandler struct {
	validator *Validator
}

// NewErrorHandler creates a new error handler
func NewErrorHandler() *ErrorHandler {
	return &ErrorHandler{
		validator: NewValidator(),
	}
}

// HandleError processes an error and returns appropriate response
func (h *ErrorHandler) HandleError(err error, context map[string]interface{}) *FrontendError {
	frontendErr := ToFrontendError(err)

	// Add additional context if provided
	if context != nil {
		if frontendErr.Context == nil {
			frontendErr.Context = make(map[string]interface{})
		}
		for k, v := range context {
			frontendErr.Context[k] = v
		}
	}

	return frontendErr
}

// ValidateAndExecute validates inputs and executes a function with error handling
func (h *ErrorHandler) ValidateAndExecute(validators []func() *ValidationResult, fn func() error) error {
	// Run validations
	for _, validate := range validators {
		if result := validate(); !result.IsValid {
			return result.GetFirstError()
		}
	}

	// Execute function with retry logic for retryable errors
	retryHandler := NewRetryHandler(3)
	return retryHandler.Execute(fn)
}
