package fourway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	kern "github.com/mobentum/kern"
	"github.com/mobentum/kern/middleware"
)

func BenchmarkRequestGuard(b *testing.B) {
	b.Run("unguarded", benchmarkKernRequestGuardUnguarded)
	b.Run("guarded", benchmarkKernRequestGuardGuarded)
}

func benchmarkKernRequestGuardUnguarded(b *testing.B) {
	app := kern.New()
	app.POST("/ingest", func(c *kern.Context) {
		c.NoContent(http.StatusAccepted)
	})

	body := []byte(`{"name":"kern","email":"dev@mobentum.dev"}`)
	bodyReader := newRewindReadCloser(body)
	req := httptest.NewRequest(http.MethodPost, "/ingest", bodyReader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant", "acme")
	res := newDiscardResponseWriter()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res.Reset()
		bodyReader.Reset(body)
		app.ServeHTTP(res, req)
	}
}

func benchmarkKernRequestGuardGuarded(b *testing.B) {
	app := kern.New()
	app.RouteWithMiddleware(http.MethodPost, "/ingest", func(c *kern.Context) {
		c.NoContent(http.StatusAccepted)
	}, middleware.RequestGuard(middleware.RequestGuardConfig{
		RequireBody:       true,
		RequireHeaders:    []string{"X-Tenant"},
		AllowContentTypes: []string{"application/json"},
		MaxBodyBytes:      1 << 20,
	}))

	body := []byte(`{"name":"kern","email":"dev@mobentum.dev"}`)
	bodyReader := newRewindReadCloser(body)
	req := httptest.NewRequest(http.MethodPost, "/ingest", bodyReader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant", "acme")
	res := newDiscardResponseWriter()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res.Reset()
		bodyReader.Reset(body)
		app.ServeHTTP(res, req)
	}
}
