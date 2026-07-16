package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mobentum/kern"
)

func TestRequestGuard_RequiredHeader(t *testing.T) {
	app := kern.New()
	app.AddConstraints(http.MethodGet, "/secure", kern.Constraints{
		Validate: RequestGuard(RequestGuardConfig{RequireHeaders: []string{"X-Tenant"}, Message: "missing tenant"}),
	}, func(c *kern.Context) {
		_ = c.Text(http.StatusOK, "%s", "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", w.Code)
	}
}

func TestRequestGuard_RequireBody(t *testing.T) {
	app := kern.New()
	app.AddConstraints(http.MethodPost, "/submit", kern.Constraints{
		Validate: RequestGuard(RequestGuardConfig{RequireBody: true}),
	}, func(c *kern.Context) {
		_ = c.Text(http.StatusOK, "%s", "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/submit", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", w.Code)
	}
}

func TestRequestGuard_AllowContentTypes(t *testing.T) {
	app := kern.New()
	app.AddConstraints(http.MethodPost, "/submit", kern.Constraints{
		Validate: RequestGuard(RequestGuardConfig{AllowContentTypes: []string{"application/json"}}),
	}, func(c *kern.Context) {
		_ = c.Text(http.StatusOK, "%s", "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/submit", bytes.NewBufferString(`{"ok":true}`))
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", w.Code)
	}
}

func TestRequestGuard_MaxBodyBytes(t *testing.T) {
	app := kern.New()
	app.AddConstraints(http.MethodPost, "/limited", kern.Constraints{
		Validate: RequestGuard(RequestGuardConfig{MaxBodyBytes: 4}),
	}, func(c *kern.Context) {
		_, err := c.Body()
		if err != nil {
			if kern.IsBodyTooLarge(err) {
				_ = c.Error(http.StatusRequestEntityTooLarge, "request body too large")
				return
			}
			_ = c.Error(http.StatusBadRequest, err.Error())
			return
		}
		_ = c.Text(http.StatusOK, "%s", "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/limited", bytes.NewBufferString("123456"))
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("got %d, want 413", w.Code)
	}
}

func TestRequestGuard_Skip(t *testing.T) {
	app := kern.New()
	app.AddConstraints(http.MethodGet, "/health", kern.Constraints{
		Validate: RequestGuard(RequestGuardConfig{
			RequireHeaders: []string{"X-Required"},
			Skip: func(r *http.Request) bool {
				return r.URL.Path == "/health"
			},
		}),
	}, func(c *kern.Context) {
		_ = c.Text(http.StatusOK, "%s", "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", w.Code)
	}
}
