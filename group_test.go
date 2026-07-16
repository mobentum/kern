package kern

import (
	"net/http"
	"testing"
)

type wrapperHandler struct {
	next http.Handler
}

func (h wrapperHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.next.ServeHTTP(w, r)
}

func TestGroup_Basic(t *testing.T) {
	app := New()
	group := app.Group("/api")

	group.GET("/users", func(c *Context) {
		_ = c.Text(200, "users")
	})

	req := newRequest(http.MethodGet, "/api/users")
	res := serve(app, req)

	if res.Body.String() != "users" {
		t.Errorf("got %s, want users", res.Body.String())
	}
}

func TestGroup_Nested(t *testing.T) {
	app := New()
	api := app.Group("/api")
	users := api.Group("/users")

	users.GET("/{id}/posts", func(c *Context) {
		_ = c.Text(200, "posts for %s", c.Param("id"))
	})

	req := newRequest(http.MethodGet, "/api/users/123/posts")
	res := serve(app, req)

	if res.Body.String() != "posts for 123" {
		t.Errorf("got %s, want posts for 123", res.Body.String())
	}
}

func TestGroup_Middleware(t *testing.T) {
	app := New()
	order := []int{}

	group := app.Group("/api", func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, 1)
			next.ServeHTTP(w, r)
		})
	})

	group.GET("/test", func(c *Context) {
		order = append(order, 2)
	})

	req := newRequest(http.MethodGet, "/api/test")
	res := serve(app, req)

	if len(order) != 2 || order[0] != 1 || order[1] != 2 {
		t.Errorf("wrong order: %v", order)
	}
	if res.Code != http.StatusOK {
		t.Errorf("got %d, want 200", res.Code)
	}
}

func TestGroup_MiddlewareHandlerCompatibility(t *testing.T) {
	app := New()
	group := app.Group("/api", func(next http.Handler) http.Handler {
		return wrapperHandler{next: next}
	})

	group.GET("/test", func(c *Context) {
		_ = c.Text(http.StatusOK, "ok")
	})

	req := newRequest(http.MethodGet, "/api/test")
	res := serve(app, req)

	if res.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", res.Code)
	}
}

func TestGroup_NamedRoute_Introspection(t *testing.T) {
	app := New()
	group := app.Group("/api")

	group.GETNamed("api_users", "/users", func(c *Context) {
		_ = c.Text(http.StatusOK, "ok")
	})

	info, ok := app.RouteByName("api_users")
	if !ok {
		t.Fatal("expected route by name")
	}

	if info.Method != http.MethodGet || info.Path != "/api/users" || info.Name != "api_users" {
		t.Fatalf("unexpected route info: %+v", info)
	}
}

func TestGroup_RouteWithConstraints(t *testing.T) {
	app := New()
	group := app.Group("/api")
	group.AddConstraints(http.MethodGet, "/users/{id}", Constraints{
		Path: PathConstraints{"id": UintPathConstraint},
	}, func(c *Context) {
		_ = c.Text(http.StatusOK, "%s", c.Param("id"))
	})

	res := serve(app, newRequest(http.MethodGet, "/api/users/9"))
	if res.Code != http.StatusOK || res.Body.String() != "9" {
		t.Fatalf("unexpected valid response: code=%d body=%q", res.Code, res.Body.String())
	}

	res = serve(app, newRequest(http.MethodGet, "/api/users/not-number"))
	if res.Code != http.StatusNotFound {
		t.Fatalf("got %d, want 404", res.Code)
	}
}

func TestGroup_RouteWithMiddleware(t *testing.T) {
	app := New()
	group := app.Group("/api")
	group.AddConstraints(http.MethodGet, "/guarded", Constraints{
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

	res := serve(app, newRequest(http.MethodGet, "/api/guarded"))
	if res.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", res.Code)
	}

	req := newRequest(http.MethodGet, "/api/guarded")
	req.Header.Set("X-Guard", "1")
	res = serve(app, req)
	if res.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", res.Code)
	}
}

func TestGroup_Use(t *testing.T) {
	app := New()
	group := app.Group("/api")

	var visited bool
	group.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			visited = true
			next.ServeHTTP(w, r)
		})
	})

	group.GET("/test", func(c *Context) {
		_ = c.Text(http.StatusOK, "ok")
	})

	res := serve(app, newRequest(http.MethodGet, "/api/test"))
	if res.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", res.Code)
	}
	if !visited {
		t.Fatal("expected middleware to run")
	}
}

func TestGroup_VerbMethods(t *testing.T) {
	t.Run("POST", func(t *testing.T) {
		app := New()
		grp := app.Group("/api")
		grp.POST("/data", func(c *Context) {
			_ = c.Text(http.StatusOK, "post")
		})
		res := serve(app, newRequest(http.MethodPost, "/api/data"))
		if res.Body.String() != "post" {
			t.Fatalf("got %q, want %q", res.Body.String(), "post")
		}
	})

	t.Run("PUT", func(t *testing.T) {
		app := New()
		grp := app.Group("/api")
		grp.PUT("/data", func(c *Context) {
			_ = c.Text(http.StatusOK, "put")
		})
		res := serve(app, newRequest(http.MethodPut, "/api/data"))
		if res.Body.String() != "put" {
			t.Fatalf("got %q, want %q", res.Body.String(), "put")
		}
	})

	t.Run("PATCH", func(t *testing.T) {
		app := New()
		grp := app.Group("/api")
		grp.PATCH("/data", func(c *Context) {
			_ = c.Text(http.StatusOK, "patch")
		})
		res := serve(app, newRequest(http.MethodPatch, "/api/data"))
		if res.Body.String() != "patch" {
			t.Fatalf("got %q, want %q", res.Body.String(), "patch")
		}
	})

	t.Run("DELETE", func(t *testing.T) {
		app := New()
		grp := app.Group("/api")
		grp.DELETE("/data", func(c *Context) {
			_ = c.Text(http.StatusOK, "delete")
		})
		res := serve(app, newRequest(http.MethodDelete, "/api/data"))
		if res.Body.String() != "delete" {
			t.Fatalf("got %q, want %q", res.Body.String(), "delete")
		}
	})

	t.Run("HEAD", func(t *testing.T) {
		app := New()
		grp := app.Group("/api")
		grp.HEAD("/data", func(c *Context) {
			_ = c.Text(http.StatusOK, "head")
		})
		res := serve(app, newRequest(http.MethodHead, "/api/data"))
		if res.Code != http.StatusOK {
			t.Fatalf("got %d, want 200", res.Code)
		}
	})

	t.Run("OPTIONS", func(t *testing.T) {
		app := New()
		grp := app.Group("/api")
		grp.OPTIONS("/data", func(c *Context) {
			_ = c.Text(http.StatusOK, "options")
		})
		res := serve(app, newRequest(http.MethodOptions, "/api/data"))
		if res.Body.String() != "options" {
			t.Fatalf("got %q, want %q", res.Body.String(), "options")
		}
	})

	t.Run("POSTNamed", func(t *testing.T) {
		app := New()
		grp := app.Group("/api")
		grp.POSTNamed("create", "/data", func(c *Context) {
			_ = c.Text(http.StatusOK, "post-named")
		})
		_, ok := app.RouteByName("create")
		if !ok {
			t.Fatal("expected POSTNamed route")
		}
	})

	t.Run("PUTNamed", func(t *testing.T) {
		app := New()
		grp := app.Group("/api")
		grp.PUTNamed("update", "/data", func(c *Context) {
			_ = c.Text(http.StatusOK, "put-named")
		})
		_, ok := app.RouteByName("update")
		if !ok {
			t.Fatal("expected PUTNamed route")
		}
	})

	t.Run("PATCHNamed", func(t *testing.T) {
		app := New()
		grp := app.Group("/api")
		grp.PATCHNamed("patch", "/data", func(c *Context) {
			_ = c.Text(http.StatusOK, "patch-named")
		})
		_, ok := app.RouteByName("patch")
		if !ok {
			t.Fatal("expected PATCHNamed route")
		}
	})

	t.Run("DELETENamed", func(t *testing.T) {
		app := New()
		grp := app.Group("/api")
		grp.DELETENamed("delete", "/data", func(c *Context) {
			_ = c.Text(http.StatusOK, "delete-named")
		})
		_, ok := app.RouteByName("delete")
		if !ok {
			t.Fatal("expected DELETENamed route")
		}
	})

	t.Run("HEADNamed", func(t *testing.T) {
		app := New()
		grp := app.Group("/api")
		grp.HEADNamed("head_named", "/data", func(c *Context) {
			_ = c.Text(http.StatusOK, "head-named")
		})
		_, ok := app.RouteByName("head_named")
		if !ok {
			t.Fatal("expected HEADNamed route")
		}
	})

	t.Run("OPTIONSNamed", func(t *testing.T) {
		app := New()
		grp := app.Group("/api")
		grp.OPTIONSNamed("options_named", "/data", func(c *Context) {
			_ = c.Text(http.StatusOK, "options-named")
		})
		_, ok := app.RouteByName("options_named")
		if !ok {
			t.Fatal("expected OPTIONSNamed route")
		}
	})

	t.Run("RouteNamedWithConstraints", func(t *testing.T) {
		app := New()
		grp := app.Group("/api")
		grp.AddNamedConstraints("constrained", http.MethodGet, "/items/{id}", Constraints{
			Path: PathConstraints{"id": IntPathConstraint},
		}, func(c *Context) {
			_ = c.Text(http.StatusOK, "%s", c.Param("id"))
		})

		_, ok := app.RouteByName("constrained")
		if !ok {
			t.Fatal("expected RouteNamedWithConstraints route")
		}

		res := serve(app, newRequest(http.MethodGet, "/api/items/42"))
		if res.Body.String() != "42" {
			t.Fatalf("got %q, want %q", res.Body.String(), "42")
		}

		res = serve(app, newRequest(http.MethodGet, "/api/items/abc"))
		if res.Code != http.StatusNotFound {
			t.Fatalf("got %d, want 404", res.Code)
		}
	})

	t.Run("RouteNamedWithMiddleware", func(t *testing.T) {
		app := New()
		grp := app.Group("/api")
		grp.AddNamedConstraints("guarded", http.MethodGet, "/secure", Constraints{
			Validate: func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Header.Get("X-Auth") == "" {
						http.Error(w, "unauthorized", http.StatusUnauthorized)
						return
					}
					next.ServeHTTP(w, r)
				})
			},
		}, func(c *Context) {
			_ = c.Text(http.StatusOK, "secure")
		})

		_, ok := app.RouteByName("guarded")
		if !ok {
			t.Fatal("expected RouteNamedWithMiddleware route")
		}

		res := serve(app, newRequest(http.MethodGet, "/api/secure"))
		if res.Code != http.StatusUnauthorized {
			t.Fatalf("got %d, want 401", res.Code)
		}

		req := newRequest(http.MethodGet, "/api/secure")
		req.Header.Set("X-Auth", "1")
		res = serve(app, req)
		if res.Code != http.StatusOK || res.Body.String() != "secure" {
			t.Fatalf("got %d %q, want 200 secure", res.Code, res.Body.String())
		}
	})
}
