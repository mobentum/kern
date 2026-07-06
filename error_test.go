package kern

import (
	"errors"
	"net/http"
	"testing"
)

func TestNewError_DefaultMessage(t *testing.T) {
	err := NewError(http.StatusNotFound, "")
	if err.Message != http.StatusText(http.StatusNotFound) {
		t.Fatalf("got %q, want %q", err.Message, http.StatusText(http.StatusNotFound))
	}
}

func TestNewError_FallbackMessage(t *testing.T) {
	err := NewError(499, "")
	if err.Message != "error" {
		t.Fatalf("got %q, want %q", err.Message, "error")
	}
}

func TestError_Error(t *testing.T) {
	err := NewError(http.StatusNotFound, "not found")
	if got := err.Error(); got != "404: not found" {
		t.Fatalf("got %q, want %q", got, "404: not found")
	}

	var nilErr *Error
	if got := nilErr.Error(); got != "<nil>" {
		t.Fatalf("got %q, want %q", got, "<nil>")
	}
}

func TestIsBodyTooLarge(t *testing.T) {
	wrapped := errors.New("any error")
	if IsBodyTooLarge(wrapped) {
		t.Fatalf("expected false for non MaxBytes error")
	}

	err := &http.MaxBytesError{Limit: 10}
	if !IsBodyTooLarge(err) {
		t.Fatalf("expected true for MaxBytesError")
	}
}
