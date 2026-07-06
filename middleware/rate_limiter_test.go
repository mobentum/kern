package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mobentum/kern"
)

func TestRateLimiter_BlocksAfterLimit(t *testing.T) {
	app := kern.New()
	app.Use(RateLimiter(RateLimiterConfig{Requests: 2, Window: time.Minute}))
	app.GET("/limited", func(c *kern.Context) {
		_ = c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/limited", nil)
	req.RemoteAddr = "127.0.0.1:9000"

	for i := 0; i < 2; i++ {
		res := httptest.NewRecorder()
		app.ServeHTTP(res, req)
		if res.Code != http.StatusOK {
			t.Fatalf("attempt %d got %d, want 200", i+1, res.Code)
		}
	}

	blocked := httptest.NewRecorder()
	app.ServeHTTP(blocked, req)
	if blocked.Code != http.StatusTooManyRequests {
		t.Fatalf("got %d, want %d", blocked.Code, http.StatusTooManyRequests)
	}
	if got := blocked.Header().Get("X-RateLimit-Remaining"); got != "0" {
		t.Fatalf("got remaining=%q, want 0", got)
	}
}

func TestRateLimiter_WindowReset(t *testing.T) {
	app := kern.New()
	app.Use(RateLimiter(RateLimiterConfig{Requests: 1, Window: 20 * time.Millisecond}))
	app.GET("/limited", func(c *kern.Context) {
		_ = c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/limited", nil)
	req.RemoteAddr = "127.0.0.1:9001"

	first := httptest.NewRecorder()
	app.ServeHTTP(first, req)
	if first.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", first.Code)
	}

	second := httptest.NewRecorder()
	app.ServeHTTP(second, req)
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("got %d, want 429", second.Code)
	}

	time.Sleep(25 * time.Millisecond)

	third := httptest.NewRecorder()
	app.ServeHTTP(third, req)
	if third.Code != http.StatusOK {
		t.Fatalf("got %d, want 200 after reset", third.Code)
	}
}

func TestRateLimiter_KeyFromRequest(t *testing.T) {
	t.Run("uses X-Forwarded-For first IP", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2")
		req.RemoteAddr = "192.168.1.1:1234"
		key := rateLimitKeyFromRequest(req)
		if key != "10.0.0.1" {
			t.Fatalf("got %q, want %q", key, "10.0.0.1")
		}
	})

	t.Run("uses X-Forwarded-For single IP", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Forwarded-For", "10.0.0.1")
		key := rateLimitKeyFromRequest(req)
		if key != "10.0.0.1" {
			t.Fatalf("got %q, want %q", key, "10.0.0.1")
		}
	})

	t.Run("uses X-Real-IP when no X-Forwarded-For", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Real-IP", "10.0.0.5")
		req.RemoteAddr = "192.168.1.1:1234"
		key := rateLimitKeyFromRequest(req)
		if key != "10.0.0.5" {
			t.Fatalf("got %q, want %q", key, "10.0.0.5")
		}
	})

	t.Run("uses RemoteAddr host", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.9:5678"
		key := rateLimitKeyFromRequest(req)
		if key != "10.0.0.9" {
			t.Fatalf("got %q, want %q", key, "10.0.0.9")
		}
	})

	t.Run("uses RemoteAddr directly when no port", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.9"
		key := rateLimitKeyFromRequest(req)
		if key != "10.0.0.9" {
			t.Fatalf("got %q, want %q", key, "10.0.0.9")
		}
	})

	t.Run("falls back to global", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = ""
		key := rateLimitKeyFromRequest(req)
		if key != "global" {
			t.Fatalf("got %q, want %q", key, "global")
		}
	})
}

func TestRateLimiter_CustomKey(t *testing.T) {
	app := kern.New()
	app.Use(RateLimiter(RateLimiterConfig{
		Requests: 1,
		Window:   time.Minute,
		KeyFunc: func(r *http.Request) string {
			return r.Header.Get("X-API-Key")
		},
	}))
	app.GET("/limited", func(c *kern.Context) {
		_ = c.Text(http.StatusOK, "ok")
	})

	reqA1 := httptest.NewRequest(http.MethodGet, "/limited", nil)
	reqA1.Header.Set("X-API-Key", "key-a")
	resA1 := httptest.NewRecorder()
	app.ServeHTTP(resA1, reqA1)
	if resA1.Code != http.StatusOK {
		t.Fatalf("key-a first got %d, want 200", resA1.Code)
	}

	reqB1 := httptest.NewRequest(http.MethodGet, "/limited", nil)
	reqB1.Header.Set("X-API-Key", "key-b")
	resB1 := httptest.NewRecorder()
	app.ServeHTTP(resB1, reqB1)
	if resB1.Code != http.StatusOK {
		t.Fatalf("key-b first got %d, want 200", resB1.Code)
	}

	reqA2 := httptest.NewRequest(http.MethodGet, "/limited", nil)
	reqA2.Header.Set("X-API-Key", "key-a")
	resA2 := httptest.NewRecorder()
	app.ServeHTTP(resA2, reqA2)
	if resA2.Code != http.StatusTooManyRequests {
		t.Fatalf("key-a second got %d, want 429", resA2.Code)
	}
}
