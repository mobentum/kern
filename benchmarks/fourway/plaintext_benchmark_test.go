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

func benchmarkKernPlaintext(b *testing.B) {
	app := kern.New()
	app.GET("/hello", func(c *kern.Context) {
		_ = c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	res := newDiscardResponseWriter()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res.Reset()
		app.ServeHTTP(res, req)
	}
}

func benchmarkMachPlaintext(b *testing.B) {
	app := mach.New()
	app.GET("/hello", func(c *mach.Context) {
		_ = c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	res := newDiscardResponseWriter()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res.Reset()
		app.ServeHTTP(res, req)
	}
}

func benchmarkChiPlaintext(b *testing.B) {
	router := chi.NewRouter()
	router.Get("/hello", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	res := newDiscardResponseWriter()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res.Reset()
		router.ServeHTTP(res, req)
	}
}

func benchmarkGinPlaintext(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.GET("/hello", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	res := newDiscardResponseWriter()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res.Reset()
		router.ServeHTTP(res, req)
	}
}

func benchmarkFiberPlaintext(b *testing.B) {
	app := fiber.New()
	app.Get("/hello", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	handler := app.Handler()
	ctx := newFastHTTPRequestContext(fiber.MethodGet, "/hello", nil)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.Response.Reset()
		handler(ctx)
	}
}

func benchmarkFastHTTPPlaintext(b *testing.B) {
	handler := fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(fasthttp.StatusOK)
		ctx.SetBodyString("ok")
	})
	ctx := newFastHTTPRequestContext(fasthttp.MethodGet, "/hello", nil)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.Response.Reset()
		handler(ctx)
	}
}

func benchmarkKernPlaintextMiddleware(b *testing.B) {
	app := kern.New()
	app.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header()["X-Test"] = middlewareHeaderValue
			next.ServeHTTP(w, r)
		})
	})
	app.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	})
	app.GET("/hello", func(c *kern.Context) {
		_ = c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	res := newDiscardResponseWriter()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res.Reset()
		app.ServeHTTP(res, req)
	}
}

func benchmarkMachPlaintextMiddleware(b *testing.B) {
	app := mach.New()
	app.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header()["X-Test"] = middlewareHeaderValue
			next.ServeHTTP(w, r)
		})
	})
	app.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	})
	app.GET("/hello", func(c *mach.Context) {
		_ = c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	res := newDiscardResponseWriter()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res.Reset()
		app.ServeHTTP(res, req)
	}
}

func benchmarkChiPlaintextMiddleware(b *testing.B) {
	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header()["X-Test"] = middlewareHeaderValue
			next.ServeHTTP(w, r)
		})
	})
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	})
	router.Get("/hello", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	res := newDiscardResponseWriter()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res.Reset()
		router.ServeHTTP(res, req)
	}
}

func benchmarkGinPlaintextMiddleware(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Header("X-Test", "middleware")
		c.Next()
	})
	router.Use(func(c *gin.Context) {
		c.Next()
	})
	router.GET("/hello", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	res := newDiscardResponseWriter()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res.Reset()
		router.ServeHTTP(res, req)
	}
}

func benchmarkFiberPlaintextMiddleware(b *testing.B) {
	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Set("X-Test", "middleware")
		return c.Next()
	})
	app.Use(func(c fiber.Ctx) error {
		return c.Next()
	})
	app.Get("/hello", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	handler := app.Handler()
	ctx := newFastHTTPRequestContext(fiber.MethodGet, "/hello", nil)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.Response.Reset()
		handler(ctx)
	}
}

func benchmarkFastHTTPPlaintextMiddleware(b *testing.B) {
	handler := chainFastHTTPMiddlewares(
		fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {
			ctx.SetStatusCode(fasthttp.StatusOK)
			ctx.SetBodyString("ok")
		}),
		func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
			return func(ctx *fasthttp.RequestCtx) {
				ctx.Response.Header.Set("X-Test", "middleware")
				next(ctx)
			}
		},
		func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
			return func(ctx *fasthttp.RequestCtx) {
				next(ctx)
			}
		},
	)
	ctx := newFastHTTPRequestContext(fasthttp.MethodGet, "/hello", nil)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.Response.Reset()
		handler(ctx)
	}
}
