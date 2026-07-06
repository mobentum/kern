package kern

import (
	"net/http"
	"testing"
	"time"
)

// Cache validator and precondition behavior.
func TestContext_IsNotModified_ETagMatch(t *testing.T) {
	app := New()
	app.GET("/cache", func(c *Context) {
		if c.IsNotModified("v1", time.Time{}) {
			return
		}
		_ = c.Text(http.StatusOK, "fresh")
	})

	req := newRequest(http.MethodGet, "/cache")
	req.Header.Set("If-None-Match", "\"v1\"")
	res := serve(app, req)

	if res.Code != http.StatusNotModified {
		t.Fatalf("got %d, want 304", res.Code)
	}
	if got := res.Header().Get("ETag"); got != "\"v1\"" {
		t.Fatalf("got %q, want %q", got, "\"v1\"")
	}
}

func TestContext_IsNotModified_LastModifiedMatch(t *testing.T) {
	modTime := time.Date(2026, 6, 24, 10, 0, 0, 0, time.UTC)
	app := New()
	app.GET("/cache", func(c *Context) {
		if c.IsNotModified("", modTime) {
			return
		}
		_ = c.Text(http.StatusOK, "fresh")
	})

	req := newRequest(http.MethodGet, "/cache")
	req.Header.Set("If-Modified-Since", modTime.Format(http.TimeFormat))
	res := serve(app, req)

	if res.Code != http.StatusNotModified {
		t.Fatalf("got %d, want 304", res.Code)
	}
	if got := res.Header().Get("Last-Modified"); got == "" {
		t.Fatal("expected Last-Modified header")
	}
}

func TestContext_IsNotModified_StaleRequestReturnsFresh(t *testing.T) {
	modTime := time.Date(2026, 6, 24, 10, 5, 0, 0, time.UTC)
	app := New()
	app.GET("/cache", func(c *Context) {
		if c.IsNotModified("v2", modTime) {
			return
		}
		_ = c.Text(http.StatusOK, "fresh")
	})

	req := newRequest(http.MethodGet, "/cache")
	req.Header.Set("If-None-Match", "\"v1\"")
	req.Header.Set("If-Modified-Since", modTime.Add(-1*time.Hour).Format(http.TimeFormat))
	res := serve(app, req)

	if res.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", res.Code)
	}
	if got := res.Body.String(); got != "fresh" {
		t.Fatalf("got %q, want %q", got, "fresh")
	}
}

func TestContext_CheckPreconditions_IfMatchFails412(t *testing.T) {
	app := New()
	app.PUT("/resource", func(c *Context) {
		if c.CheckPreconditions("v1", time.Time{}) {
			return
		}
		_ = c.Text(http.StatusOK, "updated")
	})

	req := newRequest(http.MethodPut, "/resource")
	req.Header.Set("If-Match", "\"v2\"")
	res := serve(app, req)

	if res.Code != http.StatusPreconditionFailed {
		t.Fatalf("got %d, want 412", res.Code)
	}
}

func TestContext_CheckPreconditions_IfMatchWildcardWithoutETagFails412(t *testing.T) {
	app := New()
	app.DELETE("/resource", func(c *Context) {
		if c.CheckPreconditions("", time.Time{}) {
			return
		}
		_ = c.Text(http.StatusOK, "deleted")
	})

	req := newRequest(http.MethodDelete, "/resource")
	req.Header.Set("If-Match", "*")
	res := serve(app, req)

	if res.Code != http.StatusPreconditionFailed {
		t.Fatalf("got %d, want 412", res.Code)
	}
}

func TestContext_CheckPreconditions_IfUnmodifiedSinceFails412(t *testing.T) {
	modTime := time.Date(2026, 6, 24, 11, 0, 0, 0, time.UTC)
	app := New()
	app.PATCH("/resource", func(c *Context) {
		if c.CheckPreconditions("v1", modTime) {
			return
		}
		_ = c.Text(http.StatusOK, "patched")
	})

	req := newRequest(http.MethodPatch, "/resource")
	req.Header.Set("If-Unmodified-Since", modTime.Add(-1*time.Hour).Format(http.TimeFormat))
	res := serve(app, req)

	if res.Code != http.StatusPreconditionFailed {
		t.Fatalf("got %d, want 412", res.Code)
	}
}

func TestContext_CheckPreconditions_IfNoneMatchOnMutationFails412(t *testing.T) {
	app := New()
	app.POST("/resource", func(c *Context) {
		if c.CheckPreconditions("v1", time.Time{}) {
			return
		}
		_ = c.Text(http.StatusCreated, "created")
	})

	req := newRequest(http.MethodPost, "/resource")
	req.Header.Set("If-None-Match", "\"v1\"")
	res := serve(app, req)

	if res.Code != http.StatusPreconditionFailed {
		t.Fatalf("got %d, want 412", res.Code)
	}
}

func TestContext_CheckPreconditions_SatisfiesAndContinues(t *testing.T) {
	modTime := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	app := New()
	app.PUT("/resource", func(c *Context) {
		if c.CheckPreconditions("v1", modTime) {
			return
		}
		_ = c.Text(http.StatusOK, "updated")
	})

	req := newRequest(http.MethodPut, "/resource")
	req.Header.Set("If-Match", "\"v1\"")
	req.Header.Set("If-Unmodified-Since", modTime.Format(http.TimeFormat))
	res := serve(app, req)

	if res.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", res.Code)
	}
	if got := res.Header().Get("ETag"); got != "\"v1\"" {
		t.Fatalf("got %q, want %q", got, "\"v1\"")
	}
}
