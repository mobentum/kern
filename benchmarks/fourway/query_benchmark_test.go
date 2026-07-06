package fourway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-chi/chi/v5"
	fiber "github.com/gofiber/fiber/v3"
	kern "github.com/mobentum/kern"
	mach "github.com/mrshabel/mach"
	"github.com/valyala/fasthttp"
)

func benchmarkKernQueryAccess(b *testing.B) {
	app := kern.New()
	app.GET("/search", func(c *kern.Context) {
		q, page := c.QueryPairDefaultRaw("q", "", "page", "1")
		_ = c.TextPair(http.StatusOK, q, "-", page)
	})

	req := httptest.NewRequest(http.MethodGet, "/search?q=kern&page=2", nil)
	res := newDiscardResponseWriter()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res.Reset()
		app.ServeHTTP(res, req)
	}
}

func benchmarkMachQueryAccess(b *testing.B) {
	app := mach.New()
	app.GET("/search", func(c *mach.Context) {
		_ = c.Text(http.StatusOK, "%s-%s", c.Query("q"), c.DefaultQuery("page", "1"))
	})

	req := httptest.NewRequest(http.MethodGet, "/search?q=mach&page=2", nil)
	res := newDiscardResponseWriter()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res.Reset()
		app.ServeHTTP(res, req)
	}
}

func benchmarkChiQueryAccess(b *testing.B) {
	router := chi.NewRouter()
	router.Get("/search", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		page := query.Get("page")
		if page == "" {
			page = "1"
		}
		_, _ = w.Write([]byte(query.Get("q") + "-" + page))
	})

	req := httptest.NewRequest(http.MethodGet, "/search?q=chi&page=2", nil)
	res := newDiscardResponseWriter()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res.Reset()
		router.ServeHTTP(res, req)
	}
}

func benchmarkGinQueryAccess(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.GET("/search", func(c *gin.Context) {
		page := c.Query("page")
		if page == "" {
			page = "1"
		}
		c.String(http.StatusOK, c.Query("q")+"-"+page)
	})

	req := httptest.NewRequest(http.MethodGet, "/search?q=gin&page=2", nil)
	res := newDiscardResponseWriter()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res.Reset()
		router.ServeHTTP(res, req)
	}
}

func benchmarkFiberQueryAccess(b *testing.B) {
	app := fiber.New()
	app.Get("/search", func(c fiber.Ctx) error {
		page := c.Query("page")
		if page == "" {
			page = "1"
		}
		return c.SendString(c.Query("q") + "-" + page)
	})

	handler := app.Handler()
	ctx := newFastHTTPRequestContext(fiber.MethodGet, "/search?q=kern&page=2", nil)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.Response.Reset()
		handler(ctx)
	}
}

func benchmarkFastHTTPQueryAccess(b *testing.B) {
	handler := fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {
		query := ctx.QueryArgs()
		page := query.Peek("page")
		if len(page) == 0 {
			page = []byte("1")
		}
		ctx.SetStatusCode(fasthttp.StatusOK)
		ctx.SetBodyString(string(query.Peek("q")) + "-" + string(page))
	})
	ctx := newFastHTTPRequestContext(fasthttp.MethodGet, "/search?q=kern&page=2", nil)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.Response.Reset()
		handler(ctx)
	}
}
