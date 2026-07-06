package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mobentum/kern"
)

func TestRequestID(t *testing.T) {
	app := kern.New()
	app.Use(RequestID())
	app.GET("/test", func(c *kern.Context) {
		_ = c.Text(200, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Header().Get("X-Request-ID") == "" {
		t.Error("X-Request-ID header should be set")
	}
}

func TestRequestID_UseExisting(t *testing.T) {
	app := kern.New()
	app.Use(RequestID())
	app.GET("/test", func(c *kern.Context) {
		_ = c.Text(200, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-ID", "existing-id")
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Header().Get("X-Request-ID") != "existing-id" {
		t.Error("should use existing request ID")
	}
}

func TestGetRequestID(t *testing.T) {
	app := kern.New()
	app.Use(RequestID())
	app.GET("/test", func(c *kern.Context) {
		id := GetRequestID(c.Context())
		if id == "" {
			_ = c.Text(http.StatusOK, "empty")
			return
		}
		_ = c.Text(http.StatusOK, "%s", id)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	res := httptest.NewRecorder()
	app.ServeHTTP(res, req)
	if res.Body.String() == "empty" {
		t.Fatal("expected non-empty request ID")
	}
	if got := res.Body.String(); got != res.Header().Get("X-Request-ID") {
		t.Fatalf("body %q != header %q", got, res.Header().Get("X-Request-ID"))
	}

	t.Run("returns empty for context without value", func(t *testing.T) {
		id := GetRequestID(httptest.NewRequest(http.MethodGet, "/", nil).Context())
		if id != "" {
			t.Fatalf("got %q, want empty", id)
		}
	})
}

func TestGzip(t *testing.T) {
	app := kern.New()
	app.Use(Gzip())
	app.GET("/test", func(c *kern.Context) {
		_ = c.Text(200, "hello world")
	})

	t.Run("with gzip accept", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)

		if w.Header().Get("Content-Encoding") != "gzip" {
			t.Error("gzip encoding should be set")
		}
	})

	t.Run("without gzip accept", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)

		if w.Header().Get("Content-Encoding") == "gzip" {
			t.Error("gzip should not be set")
		}
	})
}
