package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mobentum/kern"
)

func TestCSRF_SetsCookieAndContextToken(t *testing.T) {
	app := kern.New()
	app.Use(CSRF())
	app.GET("/csrf", func(c *kern.Context) {
		token, ok := CSRFToken(c.Context())
		if !ok || token == "" {
			_ = c.Text(http.StatusInternalServerError, "missing-token")
			return
		}
		_ = c.Text(http.StatusOK, "%s", token)
	})

	req := httptest.NewRequest(http.MethodGet, "/csrf", nil)
	res := httptest.NewRecorder()
	app.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", res.Code)
	}

	cookies := res.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected csrf cookie")
	}
	if cookies[0].Value == "" {
		t.Fatal("expected csrf cookie value")
	}
	if got := strings.TrimSpace(res.Body.String()); got == "" {
		t.Fatal("expected csrf token in response")
	}
}

func TestCSRF_RejectsUnsafeWithoutToken(t *testing.T) {
	app := kern.New()
	app.Use(CSRF())
	app.POST("/submit", func(c *kern.Context) {
		c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/submit", nil)
	res := httptest.NewRecorder()
	app.ServeHTTP(res, req)

	if res.Code != http.StatusForbidden {
		t.Fatalf("got %d, want 403", res.Code)
	}
}

func TestCSRF_AllowsUnsafeWithMatchingToken(t *testing.T) {
	app := kern.New()
	app.Use(CSRF())
	app.POST("/submit", func(c *kern.Context) {
		c.NoContent(http.StatusOK)
	})

	seedReq := httptest.NewRequest(http.MethodGet, "/submit", nil)
	seedRes := httptest.NewRecorder()
	app.ServeHTTP(seedRes, seedReq)

	cookies := seedRes.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected csrf cookie")
	}
	csrfCookie := cookies[0]

	req := httptest.NewRequest(http.MethodPost, "/submit", nil)
	req.AddCookie(csrfCookie)
	req.Header.Set("X-CSRF-Token", csrfCookie.Value)
	res := httptest.NewRecorder()
	app.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", res.Code)
	}
}

func TestTimeout_ExceedsDeadline(t *testing.T) {
	app := kern.New()
	app.Use(Timeout(TimeoutConfig{Duration: 10 * time.Millisecond, Message: "timed out"}))
	app.GET("/slow", func(c *kern.Context) {
		time.Sleep(60 * time.Millisecond)
		_ = c.Text(http.StatusOK, "late")
	})

	req := httptest.NewRequest(http.MethodGet, "/slow", nil)
	res := httptest.NewRecorder()
	app.ServeHTTP(res, req)

	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("got %d, want 503", res.Code)
	}
	if !strings.Contains(res.Body.String(), "timed out") {
		t.Fatalf("unexpected timeout body: %q", res.Body.String())
	}
}

func TestTimeout_CompletesWithinDeadline(t *testing.T) {
	app := kern.New()
	app.Use(Timeout(TimeoutConfig{Duration: 100 * time.Millisecond, Message: "timed out"}))
	app.GET("/fast", func(c *kern.Context) {
		_ = c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/fast", nil)
	res := httptest.NewRecorder()
	app.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", res.Code)
	}
	if strings.TrimSpace(res.Body.String()) != "ok" {
		t.Fatalf("unexpected response body: %q", res.Body.String())
	}
}

func TestTimeout_CancelsRequestContextOnDeadline(t *testing.T) {
	app := kern.New()
	app.Use(Timeout(TimeoutConfig{Duration: 10 * time.Millisecond, Message: "timed out"}))

	errCh := make(chan error, 1)
	app.GET("/slow", func(c *kern.Context) {
		<-c.Context().Done()
		errCh <- c.Context().Err()
	})

	req := httptest.NewRequest(http.MethodGet, "/slow", nil)
	res := httptest.NewRecorder()
	app.ServeHTTP(res, req)

	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("got %d, want 503", res.Code)
	}

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected context cancellation error")
		}
		if err != context.DeadlineExceeded && err != context.Canceled {
			t.Fatalf("unexpected context error: %v", err)
		}
	case <-time.After(300 * time.Millisecond):
		t.Fatal("handler did not observe context cancellation")
	}
}
