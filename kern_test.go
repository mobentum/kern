package kern

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestApp_New(t *testing.T) {
	app := New()
	if app == nil {
		t.Fatal("New() returned nil")
	}
}

func TestApp_Default(t *testing.T) {
	app := Default()
	if len(app.middlewares) != 2 {
		t.Errorf("expected 2 middlewares, got %d", len(app.middlewares))
	}
}

func TestApp_RouteMethods(t *testing.T) {
	methods := []string{
		http.MethodGet, http.MethodPost, http.MethodPut,
		http.MethodPatch, http.MethodDelete, http.MethodHead,
		http.MethodOptions,
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			app := New()
			app.Route(method, "/test", func(c *Context) {
				_ = c.Text(200, "ok")
			})

			req := newRequest(method, "/test")
			res := serve(app, req)

			if res.Code != http.StatusOK {
				t.Errorf("%s: expected 200, got %d", method, res.Code)
			}
		})
	}
}

func TestApp_PathParameters(t *testing.T) {
	app := New()
	app.GET("/users/{id}/posts/{postId}", func(c *Context) {
		_ = c.Text(200, "%s-%s", c.Param("id"), c.Param("postId"))
	})

	req := newRequest(http.MethodGet, "/users/123/posts/456")
	res := serve(app, req)

	if res.Body.String() != "123-456" {
		t.Errorf("expected '123-456', got '%s'", res.Body.String())
	}
}

func TestApp_GinStylePathParameters(t *testing.T) {
	app := New()
	app.GET("/users/:id/posts/:postId", func(c *Context) {
		_ = c.Text(200, "%s-%s", c.Param("id"), c.Param("postId"))
	})

	req := newRequest(http.MethodGet, "/users/123/posts/456")
	res := serve(app, req)

	if res.Body.String() != "123-456" {
		t.Errorf("expected '123-456', got '%s'", res.Body.String())
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct{ in, want string }{
		{"/users/:id", "/users/{id}"},
		{"/users/:id/posts/:postId", "/users/{id}/posts/{postId}"},
		{"/users/{id}", "/users/{id}"},
		{"/static/", "/static/"},
		{"/api/:version/users/:id", "/api/{version}/users/{id}"},
		{"/users/:id/:action", "/users/{id}/{action}"},
	}
	for _, tt := range tests {
		if got := normalizePath(tt.in); got != tt.want {
			t.Errorf("normalizePath(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestApp_MiddlewareChain(t *testing.T) {
	app := New()
	var order []int

	app.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, 1)
			next.ServeHTTP(w, r)
		})
	})

	app.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, 2)
			next.ServeHTTP(w, r)
		})
	})

	app.GET("/test", func(c *Context) {
		order = append(order, 3)
	})

	req := newRequest(http.MethodGet, "/test")
	res := serve(app, req)

	if res.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", res.Code)
	}

	if len(order) != 3 || order[0] != 1 || order[1] != 2 || order[2] != 3 {
		t.Errorf("wrong order: %v", order)
	}
}

func TestApp_Static(t *testing.T) {
	app := New()
	app.Static("/static", ".")

	req := newRequest(http.MethodGet, "/static/kern.go")
	res := serve(app, req)

	if res.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", res.Code)
	}
}

func TestApp_404(t *testing.T) {
	app := New()
	app.GET("/exists", func(c *Context) {
		_ = c.Text(200, "found")
	})

	req := newRequest(http.MethodGet, "/notfound")
	res := serve(app, req)

	if res.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", res.Code)
	}
}

func TestApp_Run_ListenAndErrorHooks(t *testing.T) {
	app := New()

	order := make([]string, 0, 2)
	app.OnListen(func(info ListenInfo) {
		order = append(order, "listen")
	})
	app.OnError(func(err error) {
		order = append(order, "error")
	})

	err := app.Run("invalid-addr")
	if err == nil {
		t.Fatal("expected Run to return error for invalid addr")
	}

	if len(order) != 2 || order[0] != "listen" || order[1] != "error" {
		t.Fatalf("unexpected hook order: %v", order)
	}
}

func TestApp_RunTLS_ListenAndErrorHooks(t *testing.T) {
	app := New()

	order := make([]string, 0, 2)
	app.OnListen(func(info ListenInfo) {
		if !info.TLS {
			t.Fatalf("expected TLS listen info, got %+v", info)
		}
		order = append(order, "listen")
	})
	app.OnError(func(err error) {
		order = append(order, "error")
	})

	err := app.RunTLS(":0", "missing-cert.pem", "missing-key.pem")
	if err == nil {
		t.Fatal("expected RunTLS to return cert/key error")
	}

	if !strings.Contains(err.Error(), "missing-cert.pem") {
		t.Fatalf("unexpected RunTLS error: %v", err)
	}

	if len(order) != 2 || order[0] != "listen" || order[1] != "error" {
		t.Fatalf("unexpected hook order: %v", order)
	}
}

func TestApp_GracefulShutdown_ErrorThenShutdownHooks(t *testing.T) {
	app := New()

	order := make([]string, 0, 2)
	app.OnError(func(err error) {
		order = append(order, "error")
	})
	app.OnShutdown(func(err error) {
		order = append(order, "shutdown")
	})

	server := &http.Server{Addr: "invalid-addr", Handler: app}
	err := app.runWithGracefulShutdown(server, 50*time.Millisecond)
	if err == nil {
		t.Fatal("expected graceful startup failure")
	}

	if len(order) != 2 || order[0] != "error" || order[1] != "shutdown" {
		t.Fatalf("unexpected hook order: %v", order)
	}
}

func TestApp_RouteNamingAndIntrospection(t *testing.T) {
	app := New()
	app.GETNamed("users_show", "/users/{id}", func(c *Context) {
		_ = c.Text(http.StatusOK, "%s", c.Param("id"))
	})

	routes := app.Routes()
	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}
	if routes[0].Name != "users_show" || routes[0].Method != http.MethodGet || routes[0].Path != "/users/{id}" {
		t.Fatalf("unexpected route info: %+v", routes[0])
	}

	info, ok := app.RouteByName("users_show")
	if !ok {
		t.Fatal("expected route by name")
	}
	if info.Path != "/users/{id}" {
		t.Fatalf("unexpected route path: %s", info.Path)
	}
}

func TestApp_RouteNamed_DuplicatePanics(t *testing.T) {
	app := New()
	app.GETNamed("dup", "/a", func(c *Context) {
		c.NoContent(http.StatusOK)
	})

	defer func() {
		if recover() == nil {
			t.Fatal("expected duplicate route name panic")
		}
	}()

	app.GETNamed("dup", "/b", func(c *Context) {
		c.NoContent(http.StatusOK)
	})
}

func TestApp_RouteWithConstraints(t *testing.T) {
	app := New()
	app.AddConstraints(http.MethodGet, "/users/{id}", Constraints{
		Path: PathConstraints{"id": UintPathConstraint},
	}, func(c *Context) {
		_ = c.Text(http.StatusOK, "%s", c.Param("id"))
	})

	res := serve(app, newRequest(http.MethodGet, "/users/42"))
	if res.Code != http.StatusOK || res.Body.String() != "42" {
		t.Fatalf("unexpected valid response: code=%d body=%q", res.Code, res.Body.String())
	}

	res = serve(app, newRequest(http.MethodGet, "/users/-1"))
	if res.Code != http.StatusNotFound {
		t.Fatalf("got %d, want 404", res.Code)
	}
}

func TestApp_RouteNamedWithConstraints_Introspection(t *testing.T) {
	app := New()
	app.AddNamedConstraints("users_numeric", http.MethodGet, "/users/{id}", Constraints{
		Path: PathConstraints{"id": UintPathConstraint},
	}, func(c *Context) {
		_ = c.Text(http.StatusOK, "ok")
	})

	info, ok := app.RouteByName("users_numeric")
	if !ok {
		t.Fatal("expected route by name")
	}
	if info.Path != "/users/{id}" || info.Method != http.MethodGet {
		t.Fatalf("unexpected route info: %+v", info)
	}
}

func TestApp_RouteWithMiddleware(t *testing.T) {
	app := New()
	app.AddConstraints(http.MethodGet, "/guarded", Constraints{
		Validate: func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("X-Guard") == "" {
					http.Error(w, "missing guard", http.StatusBadRequest)
					return
				}
				next.ServeHTTP(w, r)
			})
		},
	}, func(c *Context) {
		_ = c.Text(http.StatusOK, "%s", "ok")
	})

	res := serve(app, newRequest(http.MethodGet, "/guarded"))
	if res.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", res.Code)
	}

	req := newRequest(http.MethodGet, "/guarded")
	req.Header.Set("X-Guard", "1")
	res = serve(app, req)
	if res.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", res.Code)
	}
}

func TestApp_RouteNamedWithMiddleware_Introspection(t *testing.T) {
	app := New()
	app.AddNamedConstraints("guarded_show", http.MethodGet, "/guarded", Constraints{
		Validate: func(next http.Handler) http.Handler { return next },
	}, func(c *Context) {
		_ = c.Text(http.StatusOK, "%s", "ok")
	})

	info, ok := app.RouteByName("guarded_show")
	if !ok {
		t.Fatal("expected route by name")
	}
	if info.Path != "/guarded" || info.Method != http.MethodGet {
		t.Fatalf("unexpected route info: %+v", info)
	}
}

func TestRunOptions_ServerKnobs(t *testing.T) {
	cfg := &serverConfig{
		keepAlivesEnabled: true,
		maxHeaderBytes:    http.DefaultMaxHeaderBytes,
	}

	WithReadTimeout(1 * time.Second)(cfg)
	WithReadHeaderTimeout(2 * time.Second)(cfg)
	WithWriteTimeout(3 * time.Second)(cfg)
	WithIdleTimeout(4 * time.Second)(cfg)
	WithMaxHeaderBytes(8192)(cfg)
	WithKeepAlivesEnabled(false)(cfg)

	if cfg.readTimeout != 1*time.Second {
		t.Fatalf("unexpected readTimeout: %s", cfg.readTimeout)
	}
	if cfg.readHeaderTimeout != 2*time.Second {
		t.Fatalf("unexpected readHeaderTimeout: %s", cfg.readHeaderTimeout)
	}
	if cfg.writeTimeout != 3*time.Second {
		t.Fatalf("unexpected writeTimeout: %s", cfg.writeTimeout)
	}
	if cfg.idleTimeout != 4*time.Second {
		t.Fatalf("unexpected idleTimeout: %s", cfg.idleTimeout)
	}
	if cfg.maxHeaderBytes != 8192 {
		t.Fatalf("unexpected maxHeaderBytes: %d", cfg.maxHeaderBytes)
	}
	if cfg.keepAlivesEnabled {
		t.Fatal("expected keepAlivesEnabled=false")
	}
}

func TestWithMaxHeaderBytes_IgnoresNonPositiveValues(t *testing.T) {
	cfg := &serverConfig{maxHeaderBytes: 1024}
	WithMaxHeaderBytes(0)(cfg)
	if cfg.maxHeaderBytes != 1024 {
		t.Fatalf("unexpected maxHeaderBytes: %d", cfg.maxHeaderBytes)
	}

	WithMaxHeaderBytes(-1)(cfg)
	if cfg.maxHeaderBytes != 1024 {
		t.Fatalf("unexpected maxHeaderBytes: %d", cfg.maxHeaderBytes)
	}
}

func TestWithTrustedProxies_InvalidEntryPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for invalid trusted proxy entry")
		}
	}()

	_ = New(WithTrustedProxies("not-an-ip"))
}

func TestWithStrictProxyHeaders_Option(t *testing.T) {
	app := New(WithStrictProxyHeaders(true))
	if !app.strictProxyHeaders {
		t.Fatal("expected strictProxyHeaders=true")
	}
}

func TestServer_MaxHeaderBytes_RejectsLargeHeaders(t *testing.T) {
	app := New()
	app.GET("/", func(c *Context) {
		_ = c.Text(http.StatusOK, "ok")
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	defer func() { _ = ln.Close() }()

	server := &http.Server{
		Handler:        app,
		MaxHeaderBytes: 128,
	}

	go func() {
		_ = server.Serve(ln)
	}()
	defer func() { _ = server.Close() }()

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer func() { _ = conn.Close() }()

	largeHeader := strings.Repeat("a", 12*1024)
	_, err = fmt.Fprintf(conn, "GET / HTTP/1.1\r\nHost: test\r\nX-Large: %s\r\n\r\n", largeHeader)
	if err != nil {
		t.Fatalf("write request failed: %v", err)
	}

	statusLine, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		t.Fatalf("read status failed: %v", err)
	}

	if !strings.Contains(statusLine, "431") {
		t.Fatalf("expected 431 for oversized headers, got status line %q", statusLine)
	}
}

func TestApp_WithLogger(t *testing.T) {
	app := New(WithLogger())
	app.GET("/test", func(c *Context) {
		_ = c.Text(http.StatusOK, "ok")
	})

	res := serve(app, newRequest(http.MethodGet, "/test"))
	if res.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", res.Code)
	}
}

func TestApp_WithRecovery(t *testing.T) {
	app := New(WithRecovery())
	app.GET("/panic", func(c *Context) {
		panic("test panic")
	})

	res := serve(app, newRequest(http.MethodGet, "/panic"))
	if res.Code != http.StatusInternalServerError {
		t.Fatalf("got %d, want 500", res.Code)
	}
}

func TestApp_WithDebug(t *testing.T) {
	app := New(WithDebug())
	app.GET("/test", func(c *Context) {
		_ = c.Text(http.StatusOK, "ok")
	})

	res := serve(app, newRequest(http.MethodGet, "/test"))
	if res.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", res.Code)
	}
}

func TestApp_RouteNamed(t *testing.T) {
	app := New()
	app.RouteNamed("my_route", http.MethodGet, "/named", func(c *Context) {
		_ = c.Text(http.StatusOK, "named")
	})

	info, ok := app.RouteByName("my_route")
	if !ok {
		t.Fatal("expected route by name")
	}
	if info.Name != "my_route" || info.Method != http.MethodGet || info.Path != "/named" {
		t.Fatalf("unexpected route info: %+v", info)
	}

	res := serve(app, newRequest(http.MethodGet, "/named"))
	if res.Code != http.StatusOK || res.Body.String() != "named" {
		t.Fatalf("got %d %q, want 200 named", res.Code, res.Body.String())
	}
}

func TestApp_NamedRouteMethods(t *testing.T) {
	t.Run("POSTNamed", func(t *testing.T) {
		app := New()
		app.POSTNamed("create", "/create", func(c *Context) {
			_ = c.Text(http.StatusOK, "created")
		})

		info, ok := app.RouteByName("create")
		if !ok || info.Method != http.MethodPost {
			t.Fatalf("unexpected route: %+v", info)
		}
		res := serve(app, newRequest(http.MethodPost, "/create"))
		if res.Body.String() != "created" {
			t.Fatalf("got %q, want %q", res.Body.String(), "created")
		}
	})

	t.Run("PUTNamed", func(t *testing.T) {
		app := New()
		app.PUTNamed("update", "/update", func(c *Context) {
			_ = c.Text(http.StatusOK, "updated")
		})

		info, ok := app.RouteByName("update")
		if !ok || info.Method != http.MethodPut {
			t.Fatalf("unexpected route: %+v", info)
		}
		res := serve(app, newRequest(http.MethodPut, "/update"))
		if res.Body.String() != "updated" {
			t.Fatalf("got %q, want %q", res.Body.String(), "updated")
		}
	})

	t.Run("PATCHNamed", func(t *testing.T) {
		app := New()
		app.PATCHNamed("patch", "/patch", func(c *Context) {
			_ = c.Text(http.StatusOK, "patched")
		})

		info, ok := app.RouteByName("patch")
		if !ok || info.Method != http.MethodPatch {
			t.Fatalf("unexpected route: %+v", info)
		}
		res := serve(app, newRequest(http.MethodPatch, "/patch"))
		if res.Body.String() != "patched" {
			t.Fatalf("got %q, want %q", res.Body.String(), "patched")
		}
	})

	t.Run("DELETENamed", func(t *testing.T) {
		app := New()
		app.DELETENamed("delete", "/delete", func(c *Context) {
			_ = c.Text(http.StatusOK, "deleted")
		})

		info, ok := app.RouteByName("delete")
		if !ok || info.Method != http.MethodDelete {
			t.Fatalf("unexpected route: %+v", info)
		}
		res := serve(app, newRequest(http.MethodDelete, "/delete"))
		if res.Body.String() != "deleted" {
			t.Fatalf("got %q, want %q", res.Body.String(), "deleted")
		}
	})

	t.Run("HEAD", func(t *testing.T) {
		app := New()
		app.HEAD("/head", func(c *Context) {
			_ = c.Text(http.StatusOK, "head")
		})

		res := serve(app, newRequest(http.MethodHead, "/head"))
		if res.Code != http.StatusOK {
			t.Fatalf("got %d, want 200", res.Code)
		}
	})

	t.Run("OPTIONS", func(t *testing.T) {
		app := New()
		app.OPTIONS("/options", func(c *Context) {
			_ = c.Text(http.StatusOK, "options")
		})

		res := serve(app, newRequest(http.MethodOptions, "/options"))
		if res.Code != http.StatusOK || res.Body.String() != "options" {
			t.Fatalf("got %d %q, want 200 options", res.Code, res.Body.String())
		}
	})

	t.Run("HEADNamed", func(t *testing.T) {
		app := New()
		app.HEADNamed("head_named", "/head", func(c *Context) {
			_ = c.Text(http.StatusOK, "head")
		})
		_, ok := app.RouteByName("head_named")
		if !ok {
			t.Fatal("expected HEADNamed route")
		}
	})

	t.Run("OPTIONSNamed", func(t *testing.T) {
		app := New()
		app.OPTIONSNamed("options_named", "/options", func(c *Context) {
			_ = c.Text(http.StatusOK, "options")
		})
		_, ok := app.RouteByName("options_named")
		if !ok {
			t.Fatal("expected OPTIONSNamed route")
		}
	})
}

func TestApp_WithGracefulShutdown(t *testing.T) {
	// Verify the option constructor works.
	opt := WithGracefulShutdown(50 * time.Millisecond)
	if opt == nil {
		t.Fatal("WithGracefulShutdown returned nil")
	}

	// Verify it applies to a config correctly.
	cfg := &serverConfig{}
	opt(cfg)
	if cfg.gracefulTimeout != 50*time.Millisecond {
		t.Fatalf("got %v, want %v", cfg.gracefulTimeout, 50*time.Millisecond)
	}
}
