package xotel

import (
	"net/http"
	"testing"

	"github.com/mobentum/kern"
)

func TestMiddleware_AddsSpanAttributes(t *testing.T) {
	app := kern.New()
	app.Use(Middleware())
	app.GET("/test", func(c *kern.Context) {
		_ = c.Text(http.StatusOK, "ok")
	})

	client := kern.NewTestClient(app)
	res := client.Get("/test")

	if res.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", res.Code)
	}
}

func TestMiddleware_SkipsConfiguredPaths(t *testing.T) {
	app := kern.New()
	app.Use(Middleware(Config{
		Skip: func(r *http.Request) bool {
			return r.URL.Path == "/health"
		},
	}))
	app.GET("/health", func(c *kern.Context) {
		_ = c.Text(http.StatusOK, "healthy")
	})

	client := kern.NewTestClient(app)
	res := client.Get("/health")

	if res.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", res.Code)
	}
}
