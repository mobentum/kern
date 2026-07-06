package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mobentum/kern"
)

type discardResponseWriter struct {
	header http.Header
	code   int
	size   int
}

func newDiscardResponseWriter() *discardResponseWriter {
	return &discardResponseWriter{
		header: make(http.Header),
		code:   http.StatusOK,
	}
}

func (w *discardResponseWriter) Header() http.Header {
	return w.header
}

func (w *discardResponseWriter) Write(data []byte) (int, error) {
	w.size += len(data)
	return len(data), nil
}

func (w *discardResponseWriter) WriteHeader(status int) {
	w.code = status
}

func (w *discardResponseWriter) Reset() {
	for key := range w.header {
		delete(w.header, key)
	}
	w.code = http.StatusOK
	w.size = 0
}

func BenchmarkRateLimiterAllow(b *testing.B) {
	app := kern.New()
	app.Use(RateLimiter(RateLimiterConfig{Requests: b.N + 1024, Window: time.Minute}))
	app.GET("/limited", func(c *kern.Context) {
		_ = c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/limited", nil)
	req.RemoteAddr = "127.0.0.1:9000"
	res := newDiscardResponseWriter()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res.Reset()
		app.ServeHTTP(res, req)
	}
}

func BenchmarkJWTValidToken(b *testing.B) {
	secret := []byte("secret")
	token := benchmarkHS256Token(secret, map[string]interface{}{
		"sub": "user-1",
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	app := kern.New()
	app.Use(JWT(JWTConfig{SigningKey: secret}))
	app.GET("/secure", func(c *kern.Context) {
		c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res := newDiscardResponseWriter()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res.Reset()
		app.ServeHTTP(res, req)
	}
}

func BenchmarkJWTExpiredToken(b *testing.B) {
	secret := []byte("secret")
	token := benchmarkHS256Token(secret, map[string]interface{}{
		"sub": "user-1",
		"exp": time.Now().Add(-time.Hour).Unix(),
	})

	app := kern.New()
	app.Use(JWT(JWTConfig{SigningKey: secret}))
	app.GET("/secure", func(c *kern.Context) {
		c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res := newDiscardResponseWriter()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res.Reset()
		app.ServeHTTP(res, req)
	}
}

func BenchmarkJWTInvalidSignature(b *testing.B) {
	secret := []byte("secret")
	token := benchmarkHS256Token([]byte("different"), map[string]interface{}{
		"sub": "user-1",
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	app := kern.New()
	app.Use(JWT(JWTConfig{SigningKey: secret}))
	app.GET("/secure", func(c *kern.Context) {
		c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res := newDiscardResponseWriter()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res.Reset()
		app.ServeHTTP(res, req)
	}
}

func BenchmarkRateLimiterParallel(b *testing.B) {
	app := kern.New()
	app.Use(RateLimiter(RateLimiterConfig{Requests: 1000000, Window: time.Minute}))
	app.GET("/limited", func(c *kern.Context) {
		c.NoContent(http.StatusOK)
	})

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		req := httptest.NewRequest(http.MethodGet, "/limited", nil)
		req.RemoteAddr = "127.0.0.1:9000"
		res := newDiscardResponseWriter()

		for pb.Next() {
			res.Reset()
			app.ServeHTTP(res, req)
		}
	})
}

func benchmarkHS256Token(secret []byte, claims map[string]interface{}) string {
	header := map[string]interface{}{"alg": "HS256", "typ": "JWT"}
	headerJSON, _ := json.Marshal(header)
	payloadJSON, _ := json.Marshal(claims)

	headerPart := base64.RawURLEncoding.EncodeToString(headerJSON)
	payloadPart := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signingInput := headerPart + "." + payloadPart

	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(signingInput))
	signaturePart := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return signingInput + "." + signaturePart
}
