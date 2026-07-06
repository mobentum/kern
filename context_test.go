package kern

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestContext_Logger_DefaultNil(t *testing.T) {
	app := New()

	var sawNil bool
	app.GET("/test", func(c *Context) {
		sawNil = c.Logger() == nil
		_ = c.Text(http.StatusOK, "ok")
	})

	res := serve(app, newRequest(http.MethodGet, "/test"))
	if res.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200", res.Code)
	}
	if !sawNil {
		t.Fatal("expected context logger to be nil by default")
	}
}

func TestContext_Logger_FromAppSLogger(t *testing.T) {
	var out bytes.Buffer
	base := slog.New(slog.NewJSONHandler(&out, &slog.HandlerOptions{Level: slog.LevelInfo}))

	app := New(WithSlogLogger(base))
	app.GET("/test", func(c *Context) {
		if c.Logger() == nil {
			_ = c.Error(http.StatusInternalServerError, "logger not configured")
			return
		}
		c.Logger().Info("handler_event", slog.String("source", "context"))
		_ = c.Text(http.StatusOK, "ok")
	})

	res := serve(app, newRequest(http.MethodGet, "/test"))
	if res.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200", res.Code)
	}

	line := strings.TrimSpace(out.String())
	if !strings.Contains(line, "handler_event") {
		t.Fatalf("expected handler log entry, got %q", line)
	}
}

func TestContext_SetLogger_OverridesPerRequest(t *testing.T) {
	var baseOut bytes.Buffer
	base := slog.New(slog.NewJSONHandler(&baseOut, &slog.HandlerOptions{Level: slog.LevelInfo}))

	var reqOut bytes.Buffer
	reqLogger := slog.New(slog.NewJSONHandler(&reqOut, &slog.HandlerOptions{Level: slog.LevelInfo}))

	app := New(WithSlogLogger(base))
	app.GET("/test", func(c *Context) {
		c.SetLogger(reqLogger)
		c.Logger().Info("request_logger_only")
		_ = c.Text(http.StatusOK, "ok")
	})

	res := serve(app, newRequest(http.MethodGet, "/test"))
	if res.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200", res.Code)
	}

	if strings.Contains(baseOut.String(), "request_logger_only") {
		t.Fatalf("did not expect overridden log in base logger output: %q", baseOut.String())
	}
	if !strings.Contains(reqOut.String(), "request_logger_only") {
		t.Fatalf("expected overridden log in request logger output: %q", reqOut.String())
	}
}

// General Context helpers and binding behavior.
func TestContext_Query(t *testing.T) {
	app := New()
	app.GET("/test", func(c *Context) {
		_ = c.Text(200, "%s", c.Query("key"))
	})

	req := newRequest(http.MethodGet, "/test?key=value")
	res := serve(app, req)

	if res.Body.String() != "value" {
		t.Errorf("got %s, want value", res.Body.String())
	}
}

func TestContext_QueryInt(t *testing.T) {
	app := New()
	app.GET("/test", func(c *Context) {
		v, err := c.QueryInt("page", 1)
		if err != nil {
			_ = c.Error(http.StatusBadRequest, err.Error())
			return
		}
		_ = c.Text(http.StatusOK, "%d", v)
	})

	res := serve(app, newRequest(http.MethodGet, "/test?page=5"))
	if got := res.Body.String(); got != "5" {
		t.Fatalf("got %q, want %q", got, "5")
	}

	res = serve(app, newRequest(http.MethodGet, "/test"))
	if got := res.Body.String(); got != "1" {
		t.Fatalf("got %q, want %q", got, "1")
	}

	res = serve(app, newRequest(http.MethodGet, "/test?page=oops"))
	if res.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", res.Code)
	}
}

func TestContext_QueryBool(t *testing.T) {
	app := New()
	app.GET("/test", func(c *Context) {
		v, err := c.QueryBool("draft", false)
		if err != nil {
			_ = c.Error(http.StatusBadRequest, err.Error())
			return
		}
		if v {
			_ = c.Text(http.StatusOK, "true")
			return
		}
		_ = c.Text(http.StatusOK, "false")
	})

	res := serve(app, newRequest(http.MethodGet, "/test?draft=true"))
	if got := res.Body.String(); got != "true" {
		t.Fatalf("got %q, want %q", got, "true")
	}

	res = serve(app, newRequest(http.MethodGet, "/test"))
	if got := res.Body.String(); got != "false" {
		t.Fatalf("got %q, want %q", got, "false")
	}

	res = serve(app, newRequest(http.MethodGet, "/test?draft=oops"))
	if res.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", res.Code)
	}
}

func TestContext_Query_EncodedKeyAndValue(t *testing.T) {
	app := New()
	app.GET("/test", func(c *Context) {
		_ = c.Text(http.StatusOK, "%s", c.Query("user name"))
	})

	res := serve(app, newRequest(http.MethodGet, "/test?user+name=alice%2Badmin"))
	if got := res.Body.String(); got != "alice+admin" {
		t.Fatalf("got %q, want %q", got, "alice+admin")
	}
}

func TestContext_Query_RepeatedParamReturnsFirstValue(t *testing.T) {
	app := New()
	app.GET("/test", func(c *Context) {
		_ = c.Text(http.StatusOK, "%s", c.Query("tag"))
	})

	res := serve(app, newRequest(http.MethodGet, "/test?tag=first&tag=second"))
	if got := res.Body.String(); got != "first" {
		t.Fatalf("got %q, want %q", got, "first")
	}
}

func TestContext_Query_MalformedEscapeSkipsInvalidValue(t *testing.T) {
	app := New()
	app.GET("/test", func(c *Context) {
		_ = c.Text(http.StatusOK, "%s", c.Query("q"))
	})

	res := serve(app, newRequest(http.MethodGet, "/test?q=%zz&q=ok"))
	if got := res.Body.String(); got != "ok" {
		t.Fatalf("got %q, want %q", got, "ok")
	}
}

func TestContext_QueryPair(t *testing.T) {
	app := New()
	app.GET("/test", func(c *Context) {
		q, page := c.QueryPair("q", "page")
		_ = c.TextPair(http.StatusOK, q, "-", page)
	})

	res := serve(app, newRequest(http.MethodGet, "/test?q=kern&page=2"))
	if got := res.Body.String(); got != "kern-2" {
		t.Fatalf("got %q, want %q", got, "kern-2")
	}
}

func TestContext_QueryPairDefault(t *testing.T) {
	app := New()
	app.GET("/test", func(c *Context) {
		q, page := c.QueryPairDefault("q", "none", "page", "1")
		_ = c.TextPair(http.StatusOK, q, "-", page)
	})

	res := serve(app, newRequest(http.MethodGet, "/test?q=kern"))
	if got := res.Body.String(); got != "kern-1" {
		t.Fatalf("got %q, want %q", got, "kern-1")
	}
}

func TestContext_QueryPair_EncodedAndMalformed(t *testing.T) {
	app := New()
	app.GET("/test", func(c *Context) {
		user, page := c.QueryPair("user name", "page")
		_ = c.TextPair(http.StatusOK, user, "-", page)
	})

	res := serve(app, newRequest(http.MethodGet, "/test?user+name=%zz&user+name=alice%2Badmin&page=2"))
	if got := res.Body.String(); got != "alice+admin-2" {
		t.Fatalf("got %q, want %q", got, "alice+admin-2")
	}
}

func TestContext_QueryPairRaw(t *testing.T) {
	app := New()
	app.GET("/test", func(c *Context) {
		q, page := c.QueryPairRaw("q", "page")
		_ = c.TextPair(http.StatusOK, q, "-", page)
	})

	res := serve(app, newRequest(http.MethodGet, "/test?q=kern&page=2"))
	if got := res.Body.String(); got != "kern-2" {
		t.Fatalf("got %q, want %q", got, "kern-2")
	}
}

func TestContext_QueryPairDefaultRaw(t *testing.T) {
	app := New()
	app.GET("/test", func(c *Context) {
		q, page := c.QueryPairDefaultRaw("q", "none", "page", "1")
		_ = c.TextPair(http.StatusOK, q, "-", page)
	})

	res := serve(app, newRequest(http.MethodGet, "/test?q=kern"))
	if got := res.Body.String(); got != "kern-1" {
		t.Fatalf("got %q, want %q", got, "kern-1")
	}
}

func TestContext_QueryPairRaw_DoesNotDecode(t *testing.T) {
	app := New()
	app.GET("/test", func(c *Context) {
		user, _ := c.QueryPairRaw("user+name", "unused")
		_ = c.Text(http.StatusOK, "%s", user)
	})

	res := serve(app, newRequest(http.MethodGet, "/test?user+name=alice%2Badmin"))
	if got := res.Body.String(); got != "alice%2Badmin" {
		t.Fatalf("got %q, want %q", got, "alice%2Badmin")
	}
}

func TestContext_Param(t *testing.T) {
	app := New()
	app.GET("/users/{id}/posts/{postId}", func(c *Context) {
		_ = c.Text(200, "%s-%s", c.Param("id"), c.Param("postId"))
	})

	req := newRequest(http.MethodGet, "/users/123/posts/456")
	res := serve(app, req)

	if res.Body.String() != "123-456" {
		t.Errorf("got %s, want 123-456", res.Body.String())
	}
}

func TestContext_GetHeader(t *testing.T) {
	app := New()
	app.GET("/test", func(c *Context) {
		_ = c.Text(200, "%s", c.GetHeader("X-Custom"))
	})

	req := newRequest(http.MethodGet, "/test")
	req.Header.Set("X-Custom", "value")
	res := serve(app, req)

	if res.Body.String() != "value" {
		t.Errorf("got %s, want value", res.Body.String())
	}
}

func TestContext_HeaderInt(t *testing.T) {
	app := New()
	app.GET("/test", func(c *Context) {
		v, err := c.HeaderInt("X-Limit", 10)
		if err != nil {
			_ = c.Error(http.StatusBadRequest, err.Error())
			return
		}
		_ = c.Text(http.StatusOK, "%d", v)
	})

	req := newRequest(http.MethodGet, "/test")
	req.Header.Set("X-Limit", "25")
	res := serve(app, req)
	if got := res.Body.String(); got != "25" {
		t.Fatalf("got %q, want %q", got, "25")
	}

	res = serve(app, newRequest(http.MethodGet, "/test"))
	if got := res.Body.String(); got != "10" {
		t.Fatalf("got %q, want %q", got, "10")
	}

	req = newRequest(http.MethodGet, "/test")
	req.Header.Set("X-Limit", "oops")
	res = serve(app, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", res.Code)
	}
}

func TestContext_HeaderBool(t *testing.T) {
	app := New()
	app.GET("/test", func(c *Context) {
		v, err := c.HeaderBool("X-Draft", false)
		if err != nil {
			_ = c.Error(http.StatusBadRequest, err.Error())
			return
		}
		if v {
			_ = c.Text(http.StatusOK, "true")
			return
		}
		_ = c.Text(http.StatusOK, "false")
	})

	req := newRequest(http.MethodGet, "/test")
	req.Header.Set("X-Draft", "true")
	res := serve(app, req)
	if got := res.Body.String(); got != "true" {
		t.Fatalf("got %q, want %q", got, "true")
	}

	res = serve(app, newRequest(http.MethodGet, "/test"))
	if got := res.Body.String(); got != "false" {
		t.Fatalf("got %q, want %q", got, "false")
	}

	req = newRequest(http.MethodGet, "/test")
	req.Header.Set("X-Draft", "oops")
	res = serve(app, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", res.Code)
	}
}

func TestContext_JSON(t *testing.T) {
	app := New()
	app.GET("/test", func(c *Context) {
		_ = c.JSON(200, map[string]string{"message": "hello"})
	})

	req := newRequest(http.MethodGet, "/test")
	res := serve(app, req)

	var got map[string]string
	_ = json.Unmarshal(res.Body.Bytes(), &got)
	if got["message"] != "hello" {
		t.Errorf("got %v, want hello", got)
	}
}

func TestContext_OK(t *testing.T) {
	app := New()
	app.GET("/test", func(c *Context) {
		_ = c.OK(map[string]string{"message": "ok"})
	})

	res := serve(app, newRequest(http.MethodGet, "/test"))
	if res.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", res.Code)
	}

	var got map[string]string
	_ = json.Unmarshal(res.Body.Bytes(), &got)
	if got["message"] != "ok" {
		t.Fatalf("got %v, want message=ok", got)
	}
}

func TestContext_Created(t *testing.T) {
	app := New()
	app.POST("/test", func(c *Context) {
		_ = c.Created(map[string]bool{"created": true})
	})

	res := serve(app, newRequest(http.MethodPost, "/test"))
	if res.Code != http.StatusCreated {
		t.Fatalf("got %d, want 201", res.Code)
	}
}

func TestContext_Accepted(t *testing.T) {
	app := New()
	app.POST("/test", func(c *Context) {
		_ = c.Accepted(map[string]bool{"queued": true})
	})

	res := serve(app, newRequest(http.MethodPost, "/test"))
	if res.Code != http.StatusAccepted {
		t.Fatalf("got %d, want 202", res.Code)
	}
}

func TestContext_Text(t *testing.T) {
	app := New()
	app.GET("/test", func(c *Context) {
		_ = c.Text(200, "Hello %s", "World")
	})

	req := newRequest(http.MethodGet, "/test")
	res := serve(app, req)

	if res.Body.String() != "Hello World" {
		t.Errorf("got %s, want Hello World", res.Body.String())
	}
}

func TestContext_TextPair(t *testing.T) {
	app := New()
	app.GET("/test", func(c *Context) {
		_ = c.TextPair(http.StatusOK, "kern", "-", "2")
	})

	req := newRequest(http.MethodGet, "/test")
	res := serve(app, req)

	if res.Body.String() != "kern-2" {
		t.Errorf("got %s, want kern-2", res.Body.String())
	}
}

func TestContext_NoContent(t *testing.T) {
	app := New()
	app.DELETE("/test", func(c *Context) {
		c.NoContent(http.StatusNoContent)
	})

	req := newRequest(http.MethodDelete, "/test")
	res := serve(app, req)

	if res.Code != http.StatusNoContent {
		t.Errorf("got %d, want 204", res.Code)
	}
}

func TestContext_Status(t *testing.T) {
	app := New()
	app.GET("/test", func(c *Context) {
		c.Status(http.StatusNotModified)
	})

	res := serve(app, newRequest(http.MethodGet, "/test"))
	if res.Code != http.StatusNotModified {
		t.Fatalf("got %d, want 304", res.Code)
	}
}

func TestContext_SetCookie(t *testing.T) {
	app := New()
	app.GET("/test", func(c *Context) {
		c.SetCookie(&http.Cookie{Name: "session", Value: "abc"})
		_ = c.Text(200, "ok")
	})

	req := newRequest(http.MethodGet, "/test")
	res := serve(app, req)

	cookies := res.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != "session" {
		t.Errorf("cookie not set")
	}
}

func TestContext_Redirect(t *testing.T) {
	app := New()
	app.GET("/old", func(c *Context) {
		c.Redirect(http.StatusMovedPermanently, "/new")
	})

	req := newRequest(http.MethodGet, "/old")
	res := serve(app, req)

	if res.Code != http.StatusMovedPermanently || res.Header().Get("Location") != "/new" {
		t.Errorf("redirect not working")
	}
}

func TestContext_Error(t *testing.T) {
	app := New()
	app.GET("/test", func(c *Context) {
		_ = c.Error(http.StatusBadRequest, "invalid payload")
	})

	req := newRequest(http.MethodGet, "/test")
	res := serve(app, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", res.Code)
	}

	var got map[string]string
	_ = json.Unmarshal(res.Body.Bytes(), &got)
	if got["error"] != "invalid payload" {
		t.Fatalf("unexpected error payload: %v", got)
	}
}

func TestContext_JSONError_WithDetails(t *testing.T) {
	app := New()
	app.GET("/test", func(c *Context) {
		_ = c.JSONError(http.StatusUnprocessableEntity, "validation failed", map[string]string{"field": "email"})
	})

	res := serve(app, newRequest(http.MethodGet, "/test"))
	if res.Code != http.StatusUnprocessableEntity {
		t.Fatalf("got %d, want 422", res.Code)
	}

	var got map[string]interface{}
	_ = json.Unmarshal(res.Body.Bytes(), &got)
	if got["error"] != "validation failed" {
		t.Fatalf("unexpected error value: %v", got)
	}
	details, ok := got["details"].(map[string]interface{})
	if !ok || details["field"] != "email" {
		t.Fatalf("unexpected details value: %v", got)
	}
}

func TestContext_BindQuery(t *testing.T) {
	type input struct {
		Name string `query:"name"`
		Age  int    `query:"age"`
	}

	app := New()
	app.GET("/users", func(c *Context) {
		var in input
		if err := c.BindQuery(&in); err != nil {
			_ = c.Error(http.StatusBadRequest, err.Error())
			return
		}
		_ = c.JSON(http.StatusOK, in)
	})

	req := newRequest(http.MethodGet, "/users?name=mobentum&age=28")
	res := serve(app, req)

	if res.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", res.Code)
	}

	var got input
	_ = json.Unmarshal(res.Body.Bytes(), &got)
	if got.Name != "mobentum" || got.Age != 28 {
		t.Fatalf("unexpected bind result: %+v", got)
	}
}

func TestContext_BindForm(t *testing.T) {
	type input struct {
		Title string `form:"title"`
		Draft bool   `form:"draft"`
	}

	app := New()
	app.POST("/posts", func(c *Context) {
		var in input
		if err := c.BindForm(&in); err != nil {
			_ = c.Error(http.StatusBadRequest, err.Error())
			return
		}
		_ = c.JSON(http.StatusOK, in)
	})

	body := bytes.NewBufferString("title=hello&draft=true")
	req := newRequestWithBody(http.MethodPost, "/posts", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := serve(app, req)

	if res.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", res.Code)
	}

	var got input
	_ = json.Unmarshal(res.Body.Bytes(), &got)
	if got.Title != "hello" || !got.Draft {
		t.Fatalf("unexpected bind result: %+v", got)
	}
}

func TestContext_BindHeader(t *testing.T) {
	type input struct {
		RequestID string `header:"X-Request-ID"`
		Tenant    string `header:"X-Tenant"`
	}

	app := New()
	app.GET("/header", func(c *Context) {
		var in input
		if err := c.BindHeader(&in); err != nil {
			_ = c.Error(http.StatusBadRequest, err.Error())
			return
		}
		_ = c.JSON(http.StatusOK, in)
	})

	req := newRequest(http.MethodGet, "/header")
	req.Header.Set("X-Request-ID", "req-1")
	req.Header.Set("X-Tenant", "acme")
	res := serve(app, req)

	if res.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", res.Code)
	}

	var got input
	_ = json.Unmarshal(res.Body.Bytes(), &got)
	if got.RequestID != "req-1" || got.Tenant != "acme" {
		t.Fatalf("unexpected bind result: %+v", got)
	}
}

func TestContext_DecodeJSON_BodyLimit(t *testing.T) {
	app := New(WithBodyLimit(8))
	app.POST("/json", func(c *Context) {
		var payload map[string]string
		err := c.DecodeJSON(&payload)
		if err != nil {
			if IsBodyTooLarge(err) {
				_ = c.Error(http.StatusRequestEntityTooLarge, "request body too large")
				return
			}
			_ = c.Error(http.StatusBadRequest, err.Error())
			return
		}

		_ = c.JSON(http.StatusOK, payload)
	})

	req := newRequestWithBody(http.MethodPost, "/json", bytes.NewBufferString(`{"name":"very-large"}`))
	req.Header.Set("Content-Type", "application/json")
	res := serve(app, req)

	if res.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("got %d, want %d", res.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestContext_BindForm_BodyLimit(t *testing.T) {
	app := New(WithBodyLimit(6))
	app.POST("/form", func(c *Context) {
		var payload struct {
			Title string `form:"title"`
		}

		err := c.BindForm(&payload)
		if err != nil {
			if IsBodyTooLarge(err) {
				_ = c.Error(http.StatusRequestEntityTooLarge, "request body too large")
				return
			}
			_ = c.Error(http.StatusBadRequest, err.Error())
			return
		}

		_ = c.JSON(http.StatusOK, payload)
	})

	req := newRequestWithBody(http.MethodPost, "/form", bytes.NewBufferString("title=toolong"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := serve(app, req)

	if res.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("got %d, want %d", res.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestContext_DecodeJSON_MalformedPayload(t *testing.T) {
	app := New()
	app.POST("/json", func(c *Context) {
		var payload map[string]any
		if err := c.DecodeJSON(&payload); err != nil {
			_ = c.Error(http.StatusBadRequest, "invalid json")
			return
		}
		_ = c.JSON(http.StatusOK, payload)
	})

	req := newRequestWithBody(http.MethodPost, "/json", bytes.NewBufferString(`{"name":`))
	req.Header.Set("Content-Type", "application/json")
	res := serve(app, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", res.Code)
	}
}

func TestContext_DecodeXML_MalformedPayload(t *testing.T) {
	type payload struct {
		Name string `xml:"name"`
	}

	app := New()
	app.POST("/xml", func(c *Context) {
		var in payload
		if err := c.DecodeXML(&in); err != nil {
			_ = c.Error(http.StatusBadRequest, "invalid xml")
			return
		}
		_ = c.JSON(http.StatusOK, in)
	})

	req := newRequestWithBody(http.MethodPost, "/xml", bytes.NewBufferString(`<payload><name>bad</payload>`))
	req.Header.Set("Content-Type", "application/xml")
	res := serve(app, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", res.Code)
	}
}

func TestContext_BindQuery_InvalidTarget(t *testing.T) {
	app := New()
	app.GET("/bind", func(c *Context) {
		var payload struct {
			Age int `query:"age"`
		}

		err := c.BindQuery(payload)
		if !errors.Is(err, ErrInvalidBindTarget) {
			_ = c.Error(http.StatusInternalServerError, "unexpected bind error")
			return
		}

		_ = c.Text(http.StatusOK, "ok")
	})

	req := newRequest(http.MethodGet, "/bind?age=42")
	res := serve(app, req)

	if res.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", res.Code)
	}
}

func TestContext_BindQuery_InvalidType(t *testing.T) {
	app := New()
	app.GET("/bind", func(c *Context) {
		var payload struct {
			Age int `query:"age"`
		}

		err := c.BindQuery(&payload)
		if err == nil {
			_ = c.Error(http.StatusInternalServerError, "expected bind error")
			return
		}

		_ = c.Error(http.StatusBadRequest, err.Error())
	})

	req := newRequest(http.MethodGet, "/bind?age=not-a-number")
	res := serve(app, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", res.Code)
	}
}

func TestContext_BindQuery_StrictRequestParsing(t *testing.T) {
	type payload struct {
		Q string `query:"q"`
	}

	app := New(WithStrictRequestParsing(true))
	app.GET("/bind", func(c *Context) {
		var in payload
		if err := c.BindQuery(&in); err != nil {
			_ = c.Error(http.StatusBadRequest, "invalid query")
			return
		}

		_ = c.Text(http.StatusOK, "%s", in.Q)
	})

	res := serve(app, newRequest(http.MethodGet, "/bind?q=%zz&q=ok"))
	if res.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", res.Code)
	}
}

func TestContext_BindQuery_DefaultNonStrictParsing(t *testing.T) {
	type payload struct {
		Q string `query:"q"`
	}

	app := New()
	app.GET("/bind", func(c *Context) {
		var in payload
		if err := c.BindQuery(&in); err != nil {
			_ = c.Error(http.StatusBadRequest, "unexpected bind error")
			return
		}

		_ = c.Text(http.StatusOK, "%s", in.Q)
	})

	res := serve(app, newRequest(http.MethodGet, "/bind?q=%zz&q=ok"))
	if res.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", res.Code)
	}
	if got := res.Body.String(); got != "ok" {
		t.Fatalf("got %q, want %q", got, "ok")
	}
}

func TestContext_Bind_StrictRequestParsingGet(t *testing.T) {
	type payload struct {
		Q string `query:"q"`
	}

	app := New(WithStrictRequestParsing(true))
	app.GET("/bind", func(c *Context) {
		var in payload
		if err := c.Bind(&in); err != nil {
			_ = c.Error(http.StatusBadRequest, "invalid query")
			return
		}

		_ = c.Text(http.StatusOK, "%s", in.Q)
	})

	res := serve(app, newRequest(http.MethodGet, "/bind?q=%zz&q=ok"))
	if res.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", res.Code)
	}
}

func TestContext_BindQuery_PointerAndSlice(t *testing.T) {
	type payload struct {
		Limit  *int     `query:"limit"`
		Tags   []string `query:"tag"`
		Active bool     `query:"active"`
	}

	app := New()
	app.GET("/bind", func(c *Context) {
		var in payload
		if err := c.BindQuery(&in); err != nil {
			_ = c.Error(http.StatusBadRequest, err.Error())
			return
		}
		_ = c.JSON(http.StatusOK, in)
	})

	req := newRequest(http.MethodGet, "/bind?limit=5&tag=api&tag=core&active=true")
	res := serve(app, req)

	if res.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", res.Code)
	}

	var got payload
	_ = json.Unmarshal(res.Body.Bytes(), &got)
	if got.Limit == nil || *got.Limit != 5 {
		t.Fatalf("unexpected limit: %+v", got.Limit)
	}
	if len(got.Tags) != 2 || got.Tags[0] != "api" || got.Tags[1] != "core" {
		t.Fatalf("unexpected tags: %+v", got.Tags)
	}
	if !got.Active {
		t.Fatalf("expected active=true, got false")
	}
}

func TestContext_BindForm_InvalidBoolType(t *testing.T) {
	type payload struct {
		Draft bool `form:"draft"`
	}

	app := New()
	app.POST("/bind", func(c *Context) {
		var in payload
		if err := c.BindForm(&in); err != nil {
			_ = c.Error(http.StatusBadRequest, err.Error())
			return
		}
		_ = c.JSON(http.StatusOK, in)
	})

	req := newRequestWithBody(http.MethodPost, "/bind", bytes.NewBufferString("draft=not-bool"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := serve(app, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", res.Code)
	}
}

func TestContext_Bind_AutoJSONAndValidateSuccess(t *testing.T) {
	type payload struct {
		Name string `json:"name" validate:"required,min=3"`
		Role string `json:"role" validate:"oneof=admin user"`
	}

	app := New()
	app.POST("/bind", func(c *Context) {
		var in payload
		if err := c.Bind(&in); err != nil {
			_ = c.Error(http.StatusBadRequest, err.Error())
			return
		}

		_ = c.JSON(http.StatusOK, in)
	})

	req := newRequestWithBody(http.MethodPost, "/bind", bytes.NewBufferString(`{"name":"kern","role":"admin"}`))
	req.Header.Set("Content-Type", "application/json")
	res := serve(app, req)

	if res.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", res.Code)
	}
}

func TestContext_Bind_AutoJSONAndValidateFailure(t *testing.T) {
	type payload struct {
		Name string `json:"name" validate:"required,min=3"`
	}

	app := New()
	app.POST("/bind", func(c *Context) {
		var in payload
		err := c.Bind(&in)
		if err != nil {
			if _, ok := err.(ValidationErrors); ok {
				_ = c.Error(http.StatusBadRequest, "validation failed")
				return
			}
			_ = c.Error(http.StatusBadRequest, err.Error())
			return
		}

		_ = c.JSON(http.StatusOK, in)
	})

	req := newRequestWithBody(http.MethodPost, "/bind", bytes.NewBufferString(`{"name":"ab"}`))
	req.Header.Set("Content-Type", "application/json")
	res := serve(app, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", res.Code)
	}
}

func TestContext_Bind_AutoQuery(t *testing.T) {
	type payload struct {
		Q    string `query:"q" validate:"required"`
		Page int    `query:"page" validate:"min=1"`
	}

	app := New()
	app.GET("/search", func(c *Context) {
		var in payload
		if err := c.Bind(&in); err != nil {
			_ = c.Error(http.StatusBadRequest, err.Error())
			return
		}
		_ = c.JSON(http.StatusOK, in)
	})

	req := newRequest(http.MethodGet, "/search?q=kern&page=2")
	res := serve(app, req)

	if res.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", res.Code)
	}

	var got payload
	_ = json.Unmarshal(res.Body.Bytes(), &got)
	if got.Q != "kern" || got.Page != 2 {
		t.Fatalf("unexpected bind result: %+v", got)
	}
}

func TestContext_DefaultQuery(t *testing.T) {
	app := New()
	app.GET("/test", func(c *Context) {
		val := c.DefaultQuery("key", "fallback")
		_ = c.Text(http.StatusOK, "%s", val)
	})

	res := serve(app, newRequest(http.MethodGet, "/test?key=value"))
	if got := res.Body.String(); got != "value" {
		t.Fatalf("got %q, want %q", got, "value")
	}

	res = serve(app, newRequest(http.MethodGet, "/test"))
	if got := res.Body.String(); got != "fallback" {
		t.Fatalf("got %q, want %q", got, "fallback")
	}
}

func TestContext_Form(t *testing.T) {
	app := New()
	app.POST("/test", func(c *Context) {
		_ = c.Text(http.StatusOK, "%s", c.Form("name"))
	})

	body := bytes.NewBufferString("name=kern")
	req := newRequestWithBody(http.MethodPost, "/test", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := serve(app, req)

	if got := res.Body.String(); got != "kern" {
		t.Fatalf("got %q, want %q", got, "kern")
	}
}

func TestContext_Cookie(t *testing.T) {
	app := New()
	app.GET("/test", func(c *Context) {
		cookie, err := c.Cookie("session")
		if err != nil {
			_ = c.Text(http.StatusOK, "missing")
			return
		}
		_ = c.Text(http.StatusOK, "%s", cookie.Value)
	})

	req := newRequest(http.MethodGet, "/test")
	req.AddCookie(&http.Cookie{Name: "session", Value: "abc"})
	res := serve(app, req)
	if got := res.Body.String(); got != "abc" {
		t.Fatalf("got %q, want %q", got, "abc")
	}

	res = serve(app, newRequest(http.MethodGet, "/test"))
	if got := res.Body.String(); got != "missing" {
		t.Fatalf("got %q, want %q", got, "missing")
	}
}

func TestContext_MethodPath(t *testing.T) {
	app := New()
	app.GET("/test", func(c *Context) {
		_ = c.Text(http.StatusOK, "%s:%s", c.Method(), c.Path())
	})

	res := serve(app, newRequest(http.MethodGet, "/test"))
	if got := res.Body.String(); got != "GET:/test" {
		t.Fatalf("got %q, want %q", got, "GET:/test")
	}
}

func TestContext_Body(t *testing.T) {
	app := New()
	app.POST("/test", func(c *Context) {
		body, err := c.Body()
		if err != nil {
			_ = c.Error(http.StatusInternalServerError, err.Error())
			return
		}
		_ = c.Text(http.StatusOK, "%s", string(body))
	})

	req := newRequestWithBody(http.MethodPost, "/test", bytes.NewBufferString("raw-body"))
	res := serve(app, req)
	if got := res.Body.String(); got != "raw-body" {
		t.Fatalf("got %q, want %q", got, "raw-body")
	}
}

func TestContext_Context(t *testing.T) {
	app := New()
	app.GET("/test", func(c *Context) {
		ctx := c.Context()
		if ctx == nil {
			_ = c.Error(http.StatusInternalServerError, "nil context")
			return
		}
		_ = c.Text(http.StatusOK, "ok")
	})

	res := serve(app, newRequest(http.MethodGet, "/test"))
	if res.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", res.Code)
	}
}

func TestContext_XML(t *testing.T) {
	app := New()
	app.GET("/test", func(c *Context) {
		type Payload struct {
			Name string `xml:"name"`
		}
		_ = c.XML(http.StatusOK, Payload{Name: "kern"})
	})

	res := serve(app, newRequest(http.MethodGet, "/test"))
	if res.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", res.Code)
	}
	if !strings.Contains(res.Body.String(), "kern") {
		t.Fatalf("expected XML with 'kern', got %q", res.Body.String())
	}
}

func TestContext_HTML(t *testing.T) {
	app := New()
	app.GET("/test", func(c *Context) {
		_ = c.HTML(http.StatusOK, "<h1>hello</h1>")
	})

	res := serve(app, newRequest(http.MethodGet, "/test"))
	if res.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", res.Code)
	}
	if got := res.Body.String(); got != "<h1>hello</h1>" {
		t.Fatalf("got %q, want %q", got, "<h1>hello</h1>")
	}
}

func TestContext_Data(t *testing.T) {
	app := New()
	app.GET("/test", func(c *Context) {
		_ = c.Data(http.StatusOK, "application/octet-stream", []byte("binary"))
	})

	res := serve(app, newRequest(http.MethodGet, "/test"))
	if res.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", res.Code)
	}
	if got := res.Body.String(); got != "binary" {
		t.Fatalf("got %q, want %q", got, "binary")
	}
	if ct := res.Header().Get("Content-Type"); ct != "application/octet-stream" {
		t.Fatalf("got Content-Type %q, want %q", ct, "application/octet-stream")
	}
}

func TestContext_Text_Variants(t *testing.T) {
	t.Run("format with single string value", func(t *testing.T) {
		app := New()
		app.GET("/test", func(c *Context) {
			_ = c.Text(http.StatusOK, "%s", "direct")
		})
		res := serve(app, newRequest(http.MethodGet, "/test"))
		if got := res.Body.String(); got != "direct" {
			t.Fatalf("got %q, want %q", got, "direct")
		}
	})

	t.Run("format with two string values using dash", func(t *testing.T) {
		app := New()
		app.GET("/test", func(c *Context) {
			_ = c.Text(http.StatusOK, "%s-%s", "left", "right")
		})
		res := serve(app, newRequest(http.MethodGet, "/test"))
		if got := res.Body.String(); got != "left-right" {
			t.Fatalf("got %q, want %q", got, "left-right")
		}
	})

	t.Run("format with mixed types", func(t *testing.T) {
		app := New()
		app.GET("/test", func(c *Context) {
			_ = c.Text(http.StatusOK, "value=%d", 42)
		})
		res := serve(app, newRequest(http.MethodGet, "/test"))
		if got := res.Body.String(); got != "value=42" {
			t.Fatalf("got %q, want %q", got, "value=42")
		}
	})

	t.Run("empty format string with no values", func(t *testing.T) {
		app := New()
		app.GET("/test", func(c *Context) {
			_ = c.Text(http.StatusOK, "")
		})
		res := serve(app, newRequest(http.MethodGet, "/test"))
		if got := res.Body.String(); got != "" {
			t.Fatalf("got %q, want empty", got)
		}
	})
}

func TestContext_ValidationError(t *testing.T) {
	err := ValidationErrors{}
	if got := err.Error(); got != "validation failed" {
		t.Fatalf("got %q, want %q", got, "validation failed")
	}

	err = ValidationErrors{{
		Field: "name",
		Tag:   "required",
		Param: "",
	}}
	if got := err.Error(); got != "validation failed for name: required" {
		t.Fatalf("got %q, want %q", got, "validation failed for name: required")
	}

	err = ValidationErrors{{
		Field: "age",
		Tag:   "min",
		Param: "18",
	}}
	if got := err.Error(); got != "validation failed for age: min=18" {
		t.Fatalf("got %q, want %q", got, "validation failed for age: min=18")
	}
}

func TestContext_SaveFile(t *testing.T) {
	dir := t.TempDir()
	data := []byte("file content")

	app := New()
	app.POST("/upload", func(c *Context) {
		file, err := c.File("upload")
		if err != nil {
			_ = c.Error(http.StatusBadRequest, err.Error())
			return
		}
		if err := c.SaveFile(file, dir+"/saved.txt"); err != nil {
			_ = c.Error(http.StatusInternalServerError, err.Error())
			return
		}
		_ = c.Text(http.StatusOK, "saved")
	})

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("upload", "test.txt")
	_, _ = part.Write(data)
	_ = writer.Close()

	req := newRequestWithBody(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	res := serve(app, req)

	if res.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", res.Code)
	}

	saved, err := os.ReadFile(dir + "/saved.txt")
	if err != nil {
		t.Fatalf("read saved file: %v", err)
	}
	if string(saved) != string(data) {
		t.Fatalf("got %q, want %q", string(saved), string(data))
	}
}

func TestContext_StreamFile(t *testing.T) {
	dir := t.TempDir()
	filePath := dir + "/stream.txt"
	data := []byte("stream content")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	app := New()
	app.GET("/stream", func(c *Context) {
		if err := c.StreamFile(filePath); err != nil {
			_ = c.Error(http.StatusInternalServerError, err.Error())
		}
	})

	res := serve(app, newRequest(http.MethodGet, "/stream"))
	if res.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", res.Code)
	}
	if got := res.Body.String(); got != "stream content" {
		t.Fatalf("got %q, want %q", got, "stream content")
	}
}

func TestContext_DownloadFile(t *testing.T) {
	dir := t.TempDir()
	filePath := dir + "/download.txt"
	data := []byte("download content")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	app := New()
	app.GET("/download", func(c *Context) {
		if err := c.DownloadFile(filePath, "out.txt"); err != nil {
			_ = c.Error(http.StatusInternalServerError, err.Error())
		}
	})

	res := serve(app, newRequest(http.MethodGet, "/download"))
	if res.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", res.Code)
	}
	if got := res.Body.String(); got != "download content" {
		t.Fatalf("got %q, want %q", got, "download content")
	}
	if cd := res.Header().Get("Content-Disposition"); !strings.Contains(cd, "out.txt") {
		t.Fatalf("expected Content-Disposition with out.txt, got %q", cd)
	}
}

func TestContext_DownloadFile_DefaultName(t *testing.T) {
	dir := t.TempDir()
	filePath := dir + "/mydata.bin"
	data := []byte("binary data")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	app := New()
	app.GET("/download", func(c *Context) {
		if err := c.DownloadFile(filePath, ""); err != nil {
			_ = c.Error(http.StatusInternalServerError, err.Error())
		}
	})

	res := serve(app, newRequest(http.MethodGet, "/download"))
	if res.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", res.Code)
	}
	if cd := res.Header().Get("Content-Disposition"); !strings.Contains(cd, "mydata.bin") {
		t.Fatalf("expected Content-Disposition with mydata.bin, got %q", cd)
	}
}

func TestContext_ServeStatic(t *testing.T) {
	dir := t.TempDir()
	// ServeStatic does filepath.Join(dir, r.URL.Path) so put files at dir + "/files/..."
	staticDir := dir + "/files"
	if err := os.MkdirAll(staticDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(staticDir+"/index.html", []byte("<h1>index</h1>"), 0644); err != nil {
		t.Fatalf("write index: %v", err)
	}
	if err := os.WriteFile(staticDir+"/static.txt", []byte("static content"), 0644); err != nil {
		t.Fatalf("write static: %v", err)
	}

	app := New()
	app.GET("/files/", func(c *Context) {
		if err := c.ServeStatic(dir); err != nil {
			_ = c.Error(http.StatusNotFound, "not found")
		}
	})

	t.Run("serves file", func(t *testing.T) {
		req := newRequest(http.MethodGet, "/files/static.txt")
		res := serve(app, req)
		if res.Code != http.StatusOK {
			t.Fatalf("got %d, want 200", res.Code)
		}
		if got := res.Body.String(); got != "static content" {
			t.Fatalf("got %q, want %q", got, "static content")
		}
	})

	t.Run("serves index.html for directory", func(t *testing.T) {
		req := newRequest(http.MethodGet, "/files/")
		res := serve(app, req)
		if res.Code != http.StatusOK {
			t.Fatalf("got %d, want 200", res.Code)
		}
		if got := res.Body.String(); got != "<h1>index</h1>" {
			t.Fatalf("got %q, want %q", got, "<h1>index</h1>")
		}
	})

	t.Run("returns 404 for missing file", func(t *testing.T) {
		req := newRequest(http.MethodGet, "/files/nonexistent")
		res := serve(app, req)
		if res.Code != http.StatusNotFound {
			t.Fatalf("got %d, want 404", res.Code)
		}
	})
}
