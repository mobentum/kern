package fourway

import (
	"encoding/json"
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

func benchmarkKernDecodeJSON(b *testing.B) {
	app := kern.New()
	app.POST("/users", func(c *kern.Context) {
		var payload benchmarkPayload
		_ = c.DecodeJSON(&payload)
		c.NoContent(http.StatusCreated)
	})

	body := []byte(`{"name":"kern","email":"dev@mobentum.dev"}`)
	bodyReader := newRewindReadCloser(body)
	req := httptest.NewRequest(http.MethodPost, "/users", bodyReader)
	req.Header.Set("Content-Type", "application/json")
	res := newDiscardResponseWriter()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res.Reset()
		bodyReader.Reset(body)
		app.ServeHTTP(res, req)
	}
}

func benchmarkMachDecodeJSON(b *testing.B) {
	app := mach.New()
	app.POST("/users", func(c *mach.Context) {
		var payload benchmarkPayload
		_ = c.DecodeJSON(&payload)
		c.NoContent(http.StatusCreated)
	})

	body := []byte(`{"name":"mach","email":"mach@example.com"}`)
	bodyReader := newRewindReadCloser(body)
	req := httptest.NewRequest(http.MethodPost, "/users", bodyReader)
	req.Header.Set("Content-Type", "application/json")
	res := newDiscardResponseWriter()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res.Reset()
		bodyReader.Reset(body)
		app.ServeHTTP(res, req)
	}
}

func benchmarkChiDecodeJSON(b *testing.B) {
	router := chi.NewRouter()
	router.Post("/users", func(w http.ResponseWriter, r *http.Request) {
		var payload benchmarkPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
	})

	body := []byte(`{"name":"chi","email":"chi@example.com"}`)
	bodyReader := newRewindReadCloser(body)
	req := httptest.NewRequest(http.MethodPost, "/users", bodyReader)
	req.Header.Set("Content-Type", "application/json")
	res := newDiscardResponseWriter()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res.Reset()
		bodyReader.Reset(body)
		router.ServeHTTP(res, req)
	}
}

func benchmarkGinDecodeJSON(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.POST("/users", func(c *gin.Context) {
		var payload benchmarkPayload
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.Status(http.StatusBadRequest)
			return
		}
		c.Status(http.StatusCreated)
	})

	body := []byte(`{"name":"gin","email":"gin@example.com"}`)
	bodyReader := newRewindReadCloser(body)
	req := httptest.NewRequest(http.MethodPost, "/users", bodyReader)
	req.Header.Set("Content-Type", "application/json")
	res := newDiscardResponseWriter()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res.Reset()
		bodyReader.Reset(body)
		router.ServeHTTP(res, req)
	}
}

func benchmarkFiberDecodeJSON(b *testing.B) {
	app := fiber.New()
	app.Post("/users", func(c fiber.Ctx) error {
		var payload benchmarkPayload
		if err := json.Unmarshal(c.BodyRaw(), &payload); err != nil {
			c.Status(fiber.StatusBadRequest)
			return nil
		}
		c.Status(fiber.StatusCreated)
		return nil
	})

	body := []byte(`{"name":"fiber","email":"fiber@example.com"}`)
	handler := app.Handler()
	ctx := newFastHTTPRequestContext(fiber.MethodPost, "/users", body)
	ctx.Request.Header.SetContentType("application/json")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.Response.Reset()
		ctx.Request.SetBodyRaw(body)
		handler(ctx)
	}
}

func benchmarkFastHTTPDecodeJSON(b *testing.B) {
	body := []byte(`{"name":"fast","email":"fast@example.com"}`)
	handler := fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {
		var payload benchmarkPayload
		if err := json.Unmarshal(ctx.PostBody(), &payload); err != nil {
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			return
		}
		ctx.SetStatusCode(fasthttp.StatusCreated)
	})
	ctx := newFastHTTPRequestContext(fasthttp.MethodPost, "/users", body)
	ctx.Request.Header.SetContentType("application/json")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.Response.Reset()
		ctx.Request.SetBodyRaw(body)
		handler(ctx)
	}
}
