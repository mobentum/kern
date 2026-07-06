package fourway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	kern "github.com/mobentum/kern"
	"github.com/mobentum/kern/middleware"
)

func BenchmarkResponseLimit(b *testing.B) {
	b.Run("unguarded", benchmarkKernResponseLimitUnguarded)
	b.Run("limited", benchmarkKernResponseLimitLimited)
}

func benchmarkKernResponseLimitUnguarded(b *testing.B) {
	app := kern.New()
	app.GET("/stream", func(c *kern.Context) {
		_, _ = c.Response.Write([]byte("ok"))
	})

	req := httptest.NewRequest(http.MethodGet, "/stream", nil)
	res := newDiscardResponseWriter()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res.Reset()
		app.ServeHTTP(res, req)
	}
}

func benchmarkKernResponseLimitLimited(b *testing.B) {
	app := kern.New()
	app.RouteWithMiddleware(http.MethodGet, "/stream", func(c *kern.Context) {
		_, _ = c.Response.Write([]byte("ok"))
	}, middleware.ResponseLimit(middleware.ResponseLimitConfig{
		MaxBytes: 64,
	}))

	req := httptest.NewRequest(http.MethodGet, "/stream", nil)
	res := newDiscardResponseWriter()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res.Reset()
		app.ServeHTTP(res, req)
	}
}
