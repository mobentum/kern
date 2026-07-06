package middleware

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mobentum/kern"
)

func TestSecurityHeaders_Defaults(t *testing.T) {
	app := kern.New()
	app.Use(SecurityHeaders())
	app.GET("/", func(c *kern.Context) {
		_ = c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	res := httptest.NewRecorder()
	app.ServeHTTP(res, req)

	if got := res.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("got %q, want %q", got, "nosniff")
	}
	if got := res.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("got %q, want %q", got, "DENY")
	}
	if got := res.Header().Get("Referrer-Policy"); got != "no-referrer" {
		t.Fatalf("got %q, want %q", got, "no-referrer")
	}
	if got := res.Header().Get("Cross-Origin-Opener-Policy"); got != "same-origin" {
		t.Fatalf("got %q, want %q", got, "same-origin")
	}
}

func TestSecurityHeaders_CustomValues(t *testing.T) {
	app := kern.New()
	app.Use(SecurityHeaders(SecurityHeadersConfig{
		ContentSecurityPolicy: "default-src 'self'",
		PermissionsPolicy:     "geolocation=()",
	}))
	app.GET("/", func(c *kern.Context) {
		_ = c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	res := httptest.NewRecorder()
	app.ServeHTTP(res, req)

	if got := res.Header().Get("Content-Security-Policy"); got != "default-src 'self'" {
		t.Fatalf("got %q, want CSP", got)
	}
	if got := res.Header().Get("Permissions-Policy"); got != "geolocation=()" {
		t.Fatalf("got %q, want Permissions-Policy", got)
	}
}

func TestSecurityHeaders_DoesNotOverwriteExisting(t *testing.T) {
	app := kern.New()
	app.Use(SecurityHeaders())
	app.GET("/", func(c *kern.Context) {
		c.SetHeader("X-Frame-Options", "SAMEORIGIN")
		_ = c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	res := httptest.NewRecorder()
	app.ServeHTTP(res, req)

	if got := res.Header().Get("X-Frame-Options"); got != "SAMEORIGIN" {
		t.Fatalf("got %q, want %q", got, "SAMEORIGIN")
	}
}

func TestSecurityHeaders_HSTSOnlyOnTLS(t *testing.T) {
	app := kern.New()
	app.Use(SecurityHeaders(SecurityHeadersConfig{
		StrictTransportSecurity: "max-age=31536000",
	}))
	app.GET("/", func(c *kern.Context) {
		_ = c.Text(http.StatusOK, "ok")
	})

	nonTLSReq := httptest.NewRequest(http.MethodGet, "/", nil)
	nonTLSRes := httptest.NewRecorder()
	app.ServeHTTP(nonTLSRes, nonTLSReq)
	if got := nonTLSRes.Header().Get("Strict-Transport-Security"); got != "" {
		t.Fatalf("got %q, want empty on non-TLS request", got)
	}

	tlsReq := httptest.NewRequest(http.MethodGet, "/", nil)
	tlsReq.TLS = &tls.ConnectionState{}
	tlsRes := httptest.NewRecorder()
	app.ServeHTTP(tlsRes, tlsReq)
	if got := tlsRes.Header().Get("Strict-Transport-Security"); got != "max-age=31536000" {
		t.Fatalf("got %q, want %q", got, "max-age=31536000")
	}
}
