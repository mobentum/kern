package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mobentum/kern"
)

func TestResponseLimit_AllowsResponseWithinLimit(t *testing.T) {
	app := kern.New()
	app.AddConstraints(http.MethodGet, "/ok", kern.Constraints{
		Validate: ResponseLimit(ResponseLimitConfig{MaxBytes: 16}),
	}, func(c *kern.Context) {
		_, _ = c.Response.Write([]byte("ok"))
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", w.Code)
	}
	if body := w.Body.String(); body != "ok" {
		t.Fatalf("got %q, want %q", body, "ok")
	}
}

func TestResponseLimit_BlocksFirstWriteWhenOverLimit(t *testing.T) {
	app := kern.New()
	app.AddConstraints(http.MethodGet, "/big", kern.Constraints{
		Validate: ResponseLimit(ResponseLimitConfig{
			MaxBytes:   4,
			StatusCode: http.StatusRequestEntityTooLarge,
			Message:    "response too large",
		}),
	}, func(c *kern.Context) {
		_, _ = c.Response.Write([]byte("this-is-too-large"))
	})

	req := httptest.NewRequest(http.MethodGet, "/big", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("got %d, want 413", w.Code)
	}
	if !strings.Contains(w.Body.String(), "response too large") {
		t.Fatalf("got body %q, want contains %q", w.Body.String(), "response too large")
	}
}

func TestResponseLimit_ReturnsErrorOnSubsequentWriteOverflow(t *testing.T) {
	app := kern.New()
	app.AddConstraints(http.MethodGet, "/stream", kern.Constraints{
		Validate: ResponseLimit(ResponseLimitConfig{MaxBytes: 4}),
	}, func(c *kern.Context) {
		_, _ = c.Response.Write([]byte("1234"))
		_, err := c.Response.Write([]byte("56"))
		if err != nil && errors.Is(err, ErrResponseTooLarge) {
			return
		}
		_ = c.Text(http.StatusInternalServerError, "expected overflow")
	})

	req := httptest.NewRequest(http.MethodGet, "/stream", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", w.Code)
	}
	if body := w.Body.String(); body != "1234" {
		t.Fatalf("got %q, want %q", body, "1234")
	}
}

func TestResponseLimit_RejectsLargeContentLengthBeforeBody(t *testing.T) {
	app := kern.New()
	app.AddConstraints(http.MethodGet, "/cl", kern.Constraints{
		Validate: ResponseLimit(ResponseLimitConfig{
			MaxBytes:   8,
			StatusCode: http.StatusRequestEntityTooLarge,
			Message:    "response too large",
		}),
	}, func(c *kern.Context) {
		c.Response.Header().Set("Content-Length", "32")
		c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/cl", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("got %d, want 413", w.Code)
	}
}

func TestResponseLimit_Skip(t *testing.T) {
	app := kern.New()
	app.AddConstraints(http.MethodGet, "/health", kern.Constraints{
		Validate: ResponseLimit(ResponseLimitConfig{
			MaxBytes: 1,
			Skip: func(r *http.Request) bool {
				return r.URL.Path == "/health"
			},
		}),
	}, func(c *kern.Context) {
		_, _ = c.Response.Write([]byte("alive"))
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", w.Code)
	}
	if body := w.Body.String(); body != "alive" {
		t.Fatalf("got %q, want %q", body, "alive")
	}
}
