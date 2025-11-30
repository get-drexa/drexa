package httperr

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
)

// HTTPError represents an HTTP error with a status code and underlying error.
type HTTPError struct {
	Code    int
	Message string
	Err     error
}

// Error implements the error interface.
func (e *HTTPError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("HTTP %d: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("HTTP %d: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error.
func (e *HTTPError) Unwrap() error {
	return e.Err
}

// NewHTTPError creates a new HTTPError with the given status code, message, and underlying error.
func NewHTTPError(code int, message string, err error) *HTTPError {
	return &HTTPError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Internal creates a new HTTPError with status 500.
func Internal(err error) *HTTPError {
	return NewHTTPError(fiber.StatusInternalServerError, "Internal", err)
}

