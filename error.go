package kern

import (
	"errors"
	"fmt"
	"net/http"
)

// Error represents a framework-level HTTP error.
type Error struct {
	Code    int
	Message string
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%d: %s", e.Code, e.Message)
}

// NewError creates a new framework error with status code and message.
func NewError(code int, message string) *Error {
	if message == "" {
		message = http.StatusText(code)
	}
	if message == "" {
		message = "error"
	}
	return &Error{Code: code, Message: message}
}

// IsBodyTooLarge reports whether err is caused by MaxBytesReader limits.
func IsBodyTooLarge(err error) bool {
	var maxErr *http.MaxBytesError
	return errors.As(err, &maxErr)
}
