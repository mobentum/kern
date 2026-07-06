package kern

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

type discardResponseWriter struct {
	header http.Header
	code   int
	size   int
}

type rewindReadCloser struct {
	reader *bytes.Reader
}

func newDiscardResponseWriter() *discardResponseWriter {
	return &discardResponseWriter{
		header: make(http.Header),
		code:   http.StatusOK,
	}
}

func (w *discardResponseWriter) Header() http.Header {
	return w.header
}

func (w *discardResponseWriter) Write(data []byte) (int, error) {
	w.size += len(data)
	return len(data), nil
}

func (w *discardResponseWriter) WriteString(data string) (int, error) {
	w.size += len(data)
	return len(data), nil
}

func (w *discardResponseWriter) WriteHeader(status int) {
	w.code = status
}

func (w *discardResponseWriter) Reset() {
	for key := range w.header {
		delete(w.header, key)
	}
	w.code = http.StatusOK
	w.size = 0
}

func newRewindReadCloser(body []byte) *rewindReadCloser {
	return &rewindReadCloser{reader: bytes.NewReader(body)}
}

func (r *rewindReadCloser) Read(data []byte) (int, error) {
	return r.reader.Read(data)
}

func (r *rewindReadCloser) Close() error {
	return nil
}

func (r *rewindReadCloser) Reset(body []byte) {
	r.reader.Reset(body)
}

func BenchmarkRouteGET(b *testing.B) {
	app := New()
	app.GET("/ping", func(c *Context) {
		_ = c.Text(http.StatusOK, "pong")
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	res := newDiscardResponseWriter()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res.Reset()
		app.ServeHTTP(res, req)
	}
}

func BenchmarkRouteGETWithMiddleware(b *testing.B) {
	app := New()
	app.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	})
	app.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	})
	app.GET("/ping", func(c *Context) {
		_ = c.Text(http.StatusOK, "pong")
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	res := newDiscardResponseWriter()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res.Reset()
		app.ServeHTTP(res, req)
	}
}

func BenchmarkRouteGETHTTptest(b *testing.B) {
	app := New()
	app.GET("/ping", func(c *Context) {
		_ = c.Text(http.StatusOK, "pong")
	})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := newRequest(http.MethodGet, "/ping")
		_ = serve(app, req)
	}
}

func BenchmarkBindQuery(b *testing.B) {
	type payload struct {
		Limit int      `query:"limit"`
		Tags  []string `query:"tag"`
	}

	app := New()
	app.GET("/bind", func(c *Context) {
		var in payload
		_ = c.BindQuery(&in)
		c.NoContent(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/bind?limit=10&tag=a&tag=b", nil)
	res := newDiscardResponseWriter()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res.Reset()
		app.ServeHTTP(res, req)
	}
}

func BenchmarkDecodeJSON(b *testing.B) {
	type payload struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	app := New()
	app.POST("/json", func(c *Context) {
		var in payload
		_ = c.DecodeJSON(&in)
		c.NoContent(http.StatusNoContent)
	})

	body := []byte(`{"name":"mobentum","email":"dev@mobentum.dev"}`)
	bodyReader := newRewindReadCloser(body)
	req := httptest.NewRequest(http.MethodPost, "/json", bodyReader)
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
