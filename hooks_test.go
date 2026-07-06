package kern

import (
	"errors"
	"net/http"
	"testing"
)

func TestHooks_OnRoute(t *testing.T) {
	app := New()

	var seen RouteInfo
	app.OnRoute(func(info RouteInfo) {
		seen = info
	})

	app.GET("/health", func(c *Context) {
		_ = c.Text(http.StatusOK, "ok")
	})

	if seen.Method != http.MethodGet || seen.Path != "/health" {
		t.Fatalf("unexpected route hook payload: %+v", seen)
	}
}

func TestHooks_OnListen(t *testing.T) {
	app := New()

	var seen ListenInfo
	app.OnListen(func(info ListenInfo) {
		seen = info
	})

	app.emitListen(ListenInfo{Addr: ":8081", TLS: true})

	if seen.Addr != ":8081" || !seen.TLS {
		t.Fatalf("unexpected listen hook payload: %+v", seen)
	}
}

func TestHooks_OnShutdown(t *testing.T) {
	app := New()

	want := errors.New("shutdown test")
	var got error
	app.OnShutdown(func(err error) {
		got = err
	})

	app.emitShutdown(want)
	if !errors.Is(got, want) {
		t.Fatalf("expected shutdown error %v, got %v", want, got)
	}
}

func TestHooks_OnError(t *testing.T) {
	app := New()

	want := errors.New("server error")
	var got error
	app.OnError(func(err error) {
		got = err
	})

	app.emitError(want)
	if !errors.Is(got, want) {
		t.Fatalf("expected error %v, got %v", want, got)
	}
}

func TestHooks_OnRoute_GroupWithoutMiddlewareEmitsOnce(t *testing.T) {
	app := New()

	count := 0
	app.OnRoute(func(info RouteInfo) {
		if info.Method == http.MethodGet && info.Path == "/api/test" {
			count++
		}
	})

	group := app.Group("/api")
	group.GET("/test", func(c *Context) {
		_ = c.Text(http.StatusOK, "ok")
	})

	if count != 1 {
		t.Fatalf("expected 1 route hook call, got %d", count)
	}
}

func TestHooks_OnRoute_GroupWithMiddlewareEmitsOnce(t *testing.T) {
	app := New()

	count := 0
	app.OnRoute(func(info RouteInfo) {
		if info.Method == http.MethodGet && info.Path == "/api/test" {
			count++
		}
	})

	group := app.Group("/api", func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	})
	group.GET("/test", func(c *Context) {
		_ = c.Text(http.StatusOK, "ok")
	})

	if count != 1 {
		t.Fatalf("expected 1 route hook call, got %d", count)
	}
}
