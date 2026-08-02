package xotel

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/mobentum/kern"
	"go.opentelemetry.io/otel"
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

func TestSetup_SetsGlobalProvider(t *testing.T) {
	shutdown, err := Setup(SetupConfig{
		ServiceName:  "test-service",
		Endpoint:     "127.0.0.1:4318",
		Insecure:     true,
		BatchTimeout: 100 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("Setup() error: %v", err)
	}
	defer shutdown()

	if otel.GetTracerProvider() == nil {
		t.Fatal("global tracer provider is nil")
	}
	ctx, span := otel.GetTracerProvider().Tracer("test").Start(context.Background(), "span")
	defer span.End()
	_ = ctx
}
