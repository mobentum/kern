package kern

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"testing"
)

func TestLogger(t *testing.T) {
	var out bytes.Buffer
	app := New()
	app.Use(Logger(LoggerConfig{Output: &out}))
	app.GET("/test", func(c *Context) {
		_ = c.Text(200, "ok")
	})

	req := newRequest(http.MethodGet, "/test")
	res := serve(app, req)

	if res.Code != http.StatusOK {
		t.Errorf("got %d, want 200", res.Code)
	}
	if out.Len() == 0 {
		t.Fatalf("expected logger output")
	}
}

func TestLogger_JSON_WithFields(t *testing.T) {
	var out bytes.Buffer
	app := New()
	app.Use(Logger(LoggerConfig{
		Format: "json",
		Output: &out,
		Fields: map[string]interface{}{
			"service": "kern",
			"env":     "test",
		},
	}))
	app.GET("/test", func(c *Context) {
		_ = c.Text(200, "ok")
	})

	req := newRequest(http.MethodGet, "/test")
	req.Header.Set("X-Request-ID", "req-123")
	res := serve(app, req)

	if res.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", res.Code)
	}

	line := strings.TrimSpace(out.String())
	if line == "" {
		t.Fatalf("expected json log line")
	}

	entry := map[string]interface{}{}
	if err := json.Unmarshal([]byte(line), &entry); err != nil {
		t.Fatalf("expected valid json log, got %q: %v", line, err)
	}

	if entry["service"] != "kern" || entry["env"] != "test" {
		t.Fatalf("missing static fields: %+v", entry)
	}
	if entry["request_id"] != "req-123" {
		t.Fatalf("missing request_id: %+v", entry)
	}
	if entry["timestamp"] == nil {
		t.Fatalf("missing timestamp: %+v", entry)
	}
}

func TestLogger_Text_WithFields(t *testing.T) {
	var out bytes.Buffer
	app := New()
	app.Use(Logger(LoggerConfig{
		Output: &out,
		Fields: map[string]interface{}{
			"service": "kern",
			"env":     "test",
		},
	}))
	app.GET("/test", func(c *Context) {
		_ = c.Text(200, "ok")
	})

	req := newRequest(http.MethodGet, "/test")
	res := serve(app, req)

	if res.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", res.Code)
	}

	line := out.String()
	if !strings.Contains(line, "service=kern") || !strings.Contains(line, "env=test") {
		t.Fatalf("expected static fields in text log, got %q", line)
	}
}

func TestLogger_WithSlogLogger(t *testing.T) {
	var out bytes.Buffer
	slogger := slog.New(slog.NewJSONHandler(&out, &slog.HandlerOptions{Level: slog.LevelInfo}))

	app := New()
	app.Use(Logger(LoggerConfig{
		SLogger: slogger,
		Fields: map[string]interface{}{
			"service": "kern",
		},
	}))
	app.GET("/test", func(c *Context) {
		_ = c.Text(200, "ok")
	})

	req := newRequest(http.MethodGet, "/test")
	req.Header.Set("X-Request-ID", "req-1")
	res := serve(app, req)

	if res.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", res.Code)
	}

	line := strings.TrimSpace(out.String())
	if line == "" {
		t.Fatalf("expected logger output")
	}

	entry := map[string]interface{}{}
	if err := json.Unmarshal([]byte(line), &entry); err != nil {
		t.Fatalf("expected valid json output, got %q: %v", line, err)
	}

	if entry["msg"] != "http_request" {
		t.Fatalf("expected http_request message, got %+v", entry)
	}
	if entry["method"] != "GET" || entry["path"] != "/test" {
		t.Fatalf("missing request attrs: %+v", entry)
	}
	if entry["request_id"] != "req-1" {
		t.Fatalf("missing request_id: %+v", entry)
	}
	if entry["service"] != "kern" {
		t.Fatalf("missing static field: %+v", entry)
	}
}

func TestRecovery(t *testing.T) {
	app := New()
	app.Use(Recovery())
	app.GET("/panic", func(c *Context) {
		panic("test panic")
	})

	req := newRequest(http.MethodGet, "/panic")
	res := serve(app, req)

	if res.Code != http.StatusInternalServerError {
		t.Errorf("got %d, want 500", res.Code)
	}
}

func TestCORS(t *testing.T) {
	tests := []struct {
		name        string
		config      CORSConfig
		origin      string
		checkHeader string
		checkValue  string
	}{
		{
			name: "allow all",
			config: CORSConfig{
				AllowOrigins: []string{"*"},
			},
			origin:      "http://example.com",
			checkHeader: "Access-Control-Allow-Origin",
			checkValue:  "*",
		},
		{
			name: "specific origin allowed",
			config: CORSConfig{
				AllowOrigins: []string{"http://example.com"},
			},
			origin:      "http://example.com",
			checkHeader: "Access-Control-Allow-Origin",
			checkValue:  "http://example.com",
		},
		{
			name: "specific origin not allowed",
			config: CORSConfig{
				AllowOrigins: []string{"http://example.com"},
			},
			origin:      "http://evil.com",
			checkHeader: "Access-Control-Allow-Origin",
			checkValue:  "",
		},
		{
			name: "with credentials",
			config: CORSConfig{
				AllowOrigins:     []string{"http://example.com"},
				AllowCredentials: true,
			},
			origin:      "http://example.com",
			checkHeader: "Access-Control-Allow-Credentials",
			checkValue:  "true",
		},
		{
			name: "with max age",
			config: CORSConfig{
				AllowOrigins: []string{"http://example.com"},
				MaxAge:       3600,
			},
			origin:      "http://example.com",
			checkHeader: "Access-Control-Max-Age",
			checkValue:  "3600",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := New()
			app.Use(CORSWithConfig(tt.config))
			app.GET("/test", func(c *Context) {
				_ = c.Text(200, "ok")
			})

			req := newRequest(http.MethodGet, "/test")
			req.Header.Set("Origin", tt.origin)
			res := serve(app, req)

			if got := res.Header().Get(tt.checkHeader); got != tt.checkValue {
				t.Errorf("got %s, want %s", got, tt.checkValue)
			}
		})
	}
}

func TestCORS_Preflight(t *testing.T) {
	app := New()
	app.Use(CORS([]string{"http://example.com"}))

	req := newRequest(http.MethodOptions, "/test")
	req.Header.Set("Origin", "http://example.com")
	res := serve(app, req)

	if res.Code != http.StatusNoContent {
		t.Errorf("got %d, want 204", res.Code)
	}
}

func TestBearerAuth_StaticToken(t *testing.T) {
	app := New()
	app.Use(BearerAuth("secret-token"))
	app.GET("/secure", func(c *Context) {
		_ = c.Text(http.StatusOK, "ok")
	})

	t.Run("valid token", func(t *testing.T) {
		req := newRequest(http.MethodGet, "/secure")
		req.Header.Set("Authorization", "Bearer secret-token")
		res := serve(app, req)
		if res.Code != http.StatusOK {
			t.Fatalf("got %d, want 200", res.Code)
		}
	})

	t.Run("missing token", func(t *testing.T) {
		req := newRequest(http.MethodGet, "/secure")
		res := serve(app, req)
		if res.Code != http.StatusUnauthorized {
			t.Fatalf("got %d, want 401", res.Code)
		}
		if got := res.Header().Get("WWW-Authenticate"); got == "" {
			t.Fatalf("expected WWW-Authenticate header to be set")
		}
	})

	t.Run("wrong scheme", func(t *testing.T) {
		req := newRequest(http.MethodGet, "/secure")
		req.Header.Set("Authorization", "Basic abc")
		res := serve(app, req)
		if res.Code != http.StatusUnauthorized {
			t.Fatalf("got %d, want 401", res.Code)
		}
	})
}

func TestBearerAuth_CustomValidator(t *testing.T) {
	app := New()
	app.Use(BearerAuthWithConfig(BearerAuthConfig{
		Realm: "api",
		ValidateToken: func(token string, r *http.Request) bool {
			return token == "tenant-token" && r.Header.Get("X-Tenant") == "acme"
		},
	}))
	app.GET("/secure", func(c *Context) {
		_ = c.Text(http.StatusOK, "ok")
	})

	req := newRequest(http.MethodGet, "/secure")
	req.Header.Set("Authorization", "Bearer tenant-token")
	req.Header.Set("X-Tenant", "acme")
	res := serve(app, req)

	if res.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", res.Code)
	}

	reqFail := newRequest(http.MethodGet, "/secure")
	reqFail.Header.Set("Authorization", "Bearer tenant-token")
	resFail := serve(app, reqFail)
	if resFail.Code != http.StatusUnauthorized {
		t.Fatalf("got %d, want 401", resFail.Code)
	}
	if got := resFail.Header().Get("WWW-Authenticate"); got != `Bearer realm="api"` {
		t.Fatalf("got %q, want %q", got, `Bearer realm="api"`)
	}
}

func TestBasicAuth_StaticCredentials(t *testing.T) {
	app := New()
	app.Use(BasicAuth("alice", "s3cr3t"))
	app.GET("/secure", func(c *Context) {
		_ = c.Text(http.StatusOK, "ok")
	})

	t.Run("valid credentials", func(t *testing.T) {
		req := newRequest(http.MethodGet, "/secure")
		req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("alice:s3cr3t")))
		res := serve(app, req)
		if res.Code != http.StatusOK {
			t.Fatalf("got %d, want 200", res.Code)
		}
	})

	t.Run("missing header", func(t *testing.T) {
		req := newRequest(http.MethodGet, "/secure")
		res := serve(app, req)
		if res.Code != http.StatusUnauthorized {
			t.Fatalf("got %d, want 401", res.Code)
		}
		if got := res.Header().Get("WWW-Authenticate"); got == "" {
			t.Fatalf("expected WWW-Authenticate header")
		}
	})

	t.Run("invalid credentials", func(t *testing.T) {
		req := newRequest(http.MethodGet, "/secure")
		req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("alice:wrong")))
		res := serve(app, req)
		if res.Code != http.StatusUnauthorized {
			t.Fatalf("got %d, want 401", res.Code)
		}
	})
}

func TestBasicAuth_CustomValidator(t *testing.T) {
	app := New()
	app.Use(BasicAuthWithConfig(BasicAuthConfig{
		Realm: "api",
		ValidateCredentials: func(username, password string, r *http.Request) bool {
			return username == "tenant" && password == "token" && r.Header.Get("X-Tenant") == "acme"
		},
	}))
	app.GET("/secure", func(c *Context) {
		_ = c.Text(http.StatusOK, "ok")
	})

	req := newRequest(http.MethodGet, "/secure")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("tenant:token")))
	req.Header.Set("X-Tenant", "acme")
	res := serve(app, req)
	if res.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", res.Code)
	}

	reqFail := newRequest(http.MethodGet, "/secure")
	reqFail.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("tenant:token")))
	resFail := serve(app, reqFail)
	if resFail.Code != http.StatusUnauthorized {
		t.Fatalf("got %d, want 401", resFail.Code)
	}
	if got := resFail.Header().Get("WWW-Authenticate"); got != `Basic realm="api"` {
		t.Fatalf("got %q, want %q", got, `Basic realm="api"`)
	}
}
