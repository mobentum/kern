package kern

import (
	"net/http"
	"testing"
)

// Proxy and client-IP behavior.
func TestContext_ClientIP_IPv6RemoteAddr(t *testing.T) {
	app := New()
	app.GET("/ip", func(c *Context) {
		_ = c.Text(http.StatusOK, "%s", c.ClientIP())
	})

	req := newRequest(http.MethodGet, "/ip")
	req.RemoteAddr = "[2001:db8::1]:8080"
	res := serve(app, req)

	if got := res.Body.String(); got != "2001:db8::1" {
		t.Fatalf("got %q, want %q", got, "2001:db8::1")
	}
}

func TestContext_ClientIP_TrustedProxyUsesForwardedFor(t *testing.T) {
	app := New(WithTrustedProxies("203.0.113.0/24"))
	app.GET("/ip", func(c *Context) {
		_ = c.Text(http.StatusOK, "%s", c.ClientIP())
	})

	req := newRequest(http.MethodGet, "/ip")
	req.RemoteAddr = "203.0.113.10:8080"
	req.Header.Set("X-Forwarded-For", "198.51.100.7, 203.0.113.10")
	res := serve(app, req)

	if got := res.Body.String(); got != "198.51.100.7" {
		t.Fatalf("got %q, want %q", got, "198.51.100.7")
	}
}

func TestContext_ClientIP_UntrustedProxyIgnoresForwardedFor(t *testing.T) {
	app := New(WithTrustedProxies("203.0.113.0/24"))
	app.GET("/ip", func(c *Context) {
		_ = c.Text(http.StatusOK, "%s", c.ClientIP())
	})

	req := newRequest(http.MethodGet, "/ip")
	req.RemoteAddr = "198.51.100.9:8080"
	req.Header.Set("X-Forwarded-For", "192.0.2.99")
	req.Header.Set("X-Real-IP", "192.0.2.100")
	res := serve(app, req)

	if got := res.Body.String(); got != "198.51.100.9" {
		t.Fatalf("got %q, want %q", got, "198.51.100.9")
	}
}

func TestContext_ClientIP_StrictProxyHeadersRejectMalformedForwardedFor(t *testing.T) {
	app := New(WithTrustedProxies("203.0.113.0/24"), WithStrictProxyHeaders(true))
	app.GET("/ip", func(c *Context) {
		_ = c.Text(http.StatusOK, "%s", c.ClientIP())
	})

	req := newRequest(http.MethodGet, "/ip")
	req.RemoteAddr = "203.0.113.10:8080"
	req.Header.Set("X-Forwarded-For", "198.51.100.7, bad-ip")
	res := serve(app, req)

	if got := res.Body.String(); got != "203.0.113.10" {
		t.Fatalf("got %q, want %q", got, "203.0.113.10")
	}
}

func TestContext_ClientIP_StrictProxyHeadersAcceptsValidForwardedFor(t *testing.T) {
	app := New(WithTrustedProxies("203.0.113.0/24"), WithStrictProxyHeaders(true))
	app.GET("/ip", func(c *Context) {
		_ = c.Text(http.StatusOK, "%s", c.ClientIP())
	})

	req := newRequest(http.MethodGet, "/ip")
	req.RemoteAddr = "203.0.113.10:8080"
	req.Header.Set("X-Forwarded-For", "198.51.100.7, 203.0.113.10")
	res := serve(app, req)

	if got := res.Body.String(); got != "198.51.100.7" {
		t.Fatalf("got %q, want %q", got, "198.51.100.7")
	}
}

func TestContext_ClientIP_StrictProxyHeadersRejectsMalformedRealIP(t *testing.T) {
	app := New(WithTrustedProxies("203.0.113.0/24"), WithStrictProxyHeaders(true))
	app.GET("/ip", func(c *Context) {
		_ = c.Text(http.StatusOK, "%s", c.ClientIP())
	})

	req := newRequest(http.MethodGet, "/ip")
	req.RemoteAddr = "203.0.113.10:8080"
	req.Header.Set("X-Real-IP", "not-an-ip")
	res := serve(app, req)

	if got := res.Body.String(); got != "203.0.113.10" {
		t.Fatalf("got %q, want %q", got, "203.0.113.10")
	}
}
