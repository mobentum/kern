package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mobentum/kern"
)

func TestHeaderLimits_RejectByCount(t *testing.T) {
	app := kern.New()
	app.Use(HeaderLimits(HeaderLimitsConfig{MaxHeaderCount: 1, MaxHeaderBytes: 1024}))
	app.GET("/", func(c *kern.Context) { _ = c.Text(http.StatusOK, "ok") })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-One", "a")
	req.Header.Set("X-Two", "b")
	w := httptest.NewRecorder()

	app.ServeHTTP(w, req)

	if w.Code != http.StatusRequestHeaderFieldsTooLarge {
		t.Fatalf("expected %d, got %d", http.StatusRequestHeaderFieldsTooLarge, w.Code)
	}
}

func TestHeaderLimits_RejectByBytes(t *testing.T) {
	app := kern.New()
	app.Use(HeaderLimits(HeaderLimitsConfig{
		MaxHeaderCount: 10,
		MaxHeaderBytes: 8,
		Message:        "header budget exceeded",
	}))
	app.GET("/", func(c *kern.Context) { _ = c.Text(http.StatusOK, "ok") })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Test", "123456789")
	w := httptest.NewRecorder()

	app.ServeHTTP(w, req)

	if w.Code != http.StatusRequestHeaderFieldsTooLarge {
		t.Fatalf("expected %d, got %d", http.StatusRequestHeaderFieldsTooLarge, w.Code)
	}
	if got := w.Body.String(); got == "" {
		t.Fatal("expected non-empty error response body")
	}
}

func TestHeaderLimits_Skip(t *testing.T) {
	app := kern.New()
	app.Use(HeaderLimits(HeaderLimitsConfig{
		MaxHeaderCount: 1,
		MaxHeaderBytes: 8,
		Skip: func(r *http.Request) bool {
			return r.URL.Path == "/health"
		},
	}))
	app.GET("/health", func(c *kern.Context) { _ = c.Text(http.StatusOK, "ok") })

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("X-One", "a")
	req.Header.Set("X-Two", "b")
	w := httptest.NewRecorder()

	app.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, w.Code)
	}
}

func TestHeaderLimits_AllowWithinLimits(t *testing.T) {
	app := kern.New()
	app.Use(HeaderLimits(HeaderLimitsConfig{
		MaxHeaderCount: 3,
		MaxHeaderBytes: 32,
	}))
	app.GET("/", func(c *kern.Context) { _ = c.Text(http.StatusOK, "ok") })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-One", "a")
	req.Header.Set("X-Two", "bb")
	w := httptest.NewRecorder()

	app.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, w.Code)
	}
}
