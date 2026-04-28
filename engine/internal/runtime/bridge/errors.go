package bridge

import "fmt"

// Error codes — machine-readable, stable across all runtimes.
const (
	ErrRecordNotFound    = "RECORD_NOT_FOUND"
	ErrModelNotFound     = "MODEL_NOT_FOUND"
	ErrValidation        = "VALIDATION_ERROR"
	ErrPermissionDenied  = "PERMISSION_DENIED"
	ErrSudoNotAllowed    = "SUDO_NOT_ALLOWED"
	ErrTenantRequired    = "TENANT_REQUIRED"
	ErrTenantNotFound    = "TENANT_NOT_FOUND"
	ErrEnvAccessDenied   = "ENV_ACCESS_DENIED"
	ErrExecDenied        = "EXEC_DENIED"
	ErrExecNotFound      = "EXEC_NOT_FOUND"
	ErrExecTimeout       = "EXEC_TIMEOUT"
	ErrFSAccessDenied    = "FS_ACCESS_DENIED"
	ErrFSNotFound        = "FS_NOT_FOUND"
	ErrTxTimeout         = "TX_TIMEOUT"
	ErrTxConflict        = "TX_CONFLICT"
	ErrHTTPTimeout       = "HTTP_TIMEOUT"
	ErrHTTPError         = "HTTP_ERROR"
	ErrEmailNotConfigured = "EMAIL_NOT_CONFIGURED"
	ErrStorageError      = "STORAGE_ERROR"
	ErrCryptoError       = "CRYPTO_ERROR"
	ErrInternalError     = "INTERNAL_ERROR"
)

// BridgeError is the universal error type returned by all bridge methods.
// All 4 runtimes (Node.js, Python, goja, yaegi) translate this into their
// native error type (Promise rejection, exception, error object, error return).
type BridgeError struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	Details   map[string]any `json:"details,omitempty"`
	Retryable bool           `json:"retryable"`
}

func (e *BridgeError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// NewError creates a non-retryable BridgeError.
func NewError(code, message string) *BridgeError {
	return &BridgeError{Code: code, Message: message}
}

// NewErrorf creates a non-retryable BridgeError with formatted message.
func NewErrorf(code, format string, args ...any) *BridgeError {
	return &BridgeError{Code: code, Message: fmt.Sprintf(format, args...)}
}

// NewErrorWithDetails creates a non-retryable BridgeError with details.
func NewErrorWithDetails(code, message string, details map[string]any) *BridgeError {
	return &BridgeError{Code: code, Message: message, Details: details}
}

// NewRetryableError creates a retryable BridgeError.
func NewRetryableError(code, message string) *BridgeError {
	return &BridgeError{Code: code, Message: message, Retryable: true}
}

// Convenience constructors for common errors.

func ErrRecordNotFoundFor(model, id string) *BridgeError {
	return NewErrorWithDetails(ErrRecordNotFound, fmt.Sprintf("record not found in %s", model), map[string]any{
		"model": model,
		"id":    id,
	})
}

func ErrModelNotFoundFor(name string) *BridgeError {
	return NewErrorWithDetails(ErrModelNotFound, fmt.Sprintf("model '%s' not found", name), map[string]any{
		"model": name,
	})
}

func ErrValidationFor(model string, fieldErrors map[string]string) *BridgeError {
	details := map[string]any{"model": model}
	for k, v := range fieldErrors {
		details[k] = v
	}
	return NewErrorWithDetails(ErrValidation, fmt.Sprintf("validation failed for %s", model), details)
}

func ErrPermissionDeniedFor(model, operation string) *BridgeError {
	return NewErrorWithDetails(ErrPermissionDenied, fmt.Sprintf("permission denied: %s on %s", operation, model), map[string]any{
		"model":     model,
		"operation": operation,
	})
}

func ErrSudoNotAllowedFor(module string) *BridgeError {
	return NewErrorWithDetails(ErrSudoNotAllowed, fmt.Sprintf("sudo not allowed for module '%s'", module), map[string]any{
		"module": module,
	})
}
