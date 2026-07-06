package fourway

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-chi/chi/v5"
	fiber "github.com/gofiber/fiber/v3"
	kern "github.com/mobentum/kern"
	mach "github.com/mrshabel/mach"
	"github.com/valyala/fasthttp"
)

func benchmarkKernPathParams(b *testing.B) {
	app := kern.New()
	app.GET("/users/{id}/posts/{postID}", func(c *kern.Context) {
		_ = c.TextPair(http.StatusOK, c.Param("id"), "-", c.Param("postID"))
	})

	req := httptest.NewRequest(http.MethodGet, "/users/123/posts/456", nil)
	res := newDiscardResponseWriter()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res.Reset()
		app.ServeHTTP(res, req)
	}
}

func benchmarkMachPathParams(b *testing.B) {
	app := mach.New()
	app.GET("/users/{id}/posts/{postID}", func(c *mach.Context) {
		_ = c.Text(http.StatusOK, "%s-%s", c.Param("id"), c.Param("postID"))
	})

	req := httptest.NewRequest(http.MethodGet, "/users/123/posts/456", nil)
	res := newDiscardResponseWriter()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res.Reset()
		app.ServeHTTP(res, req)
	}
}

func benchmarkChiPathParams(b *testing.B) {
	router := chi.NewRouter()
	router.Get("/users/{id}/posts/{postID}", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(chi.URLParam(r, "id") + "-" + chi.URLParam(r, "postID")))
	})

	req := httptest.NewRequest(http.MethodGet, "/users/123/posts/456", nil)
	res := newDiscardResponseWriter()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res.Reset()
		router.ServeHTTP(res, req)
	}
}

func benchmarkGinPathParams(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.GET("/users/:id/posts/:postID", func(c *gin.Context) {
		c.String(http.StatusOK, c.Param("id")+"-"+c.Param("postID"))
	})

	req := httptest.NewRequest(http.MethodGet, "/users/123/posts/456", nil)
	res := newDiscardResponseWriter()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res.Reset()
		router.ServeHTTP(res, req)
	}
}

func benchmarkFiberPathParams(b *testing.B) {
	app := fiber.New()
	app.Get("/users/:id/posts/:postID", func(c fiber.Ctx) error {
		return c.SendString(c.Params("id") + "-" + c.Params("postID"))
	})

	handler := app.Handler()
	ctx := newFastHTTPRequestContext(fiber.MethodGet, "/users/123/posts/456", nil)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.Response.Reset()
		handler(ctx)
	}
}

func benchmarkFastHTTPPathParams(b *testing.B) {
	handler := fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {
		path := string(ctx.Path())
		const usersPrefix = "/users/"
		const postsToken = "/posts/"
		if !strings.HasPrefix(path, usersPrefix) {
			ctx.SetStatusCode(fasthttp.StatusNotFound)
			return
		}

		start := len(usersPrefix)
		midRel := strings.Index(path[start:], postsToken)
		if midRel < 0 {
			ctx.SetStatusCode(fasthttp.StatusNotFound)
			return
		}

		mid := start + midRel
		id := path[start:mid]
		postID := path[mid+len(postsToken):]

		ctx.SetStatusCode(fasthttp.StatusOK)
		ctx.SetBodyString(id + "-" + postID)
	})

	ctx := newFastHTTPRequestContext(fasthttp.MethodGet, "/users/123/posts/456", nil)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.Response.Reset()
		handler(ctx)
	}
}
