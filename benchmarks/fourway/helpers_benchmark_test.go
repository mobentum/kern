package fourway

import (
	"bytes"
	"net/http"

	"github.com/valyala/fasthttp"
)

type discardResponseWriter struct {
	header http.Header
	code   int
	size   int
}

type rewindReadCloser struct {
	reader *bytes.Reader
}

type benchmarkPayload struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

var middlewareHeaderValue = []string{"middleware"}

type fastHTTPMiddleware func(fasthttp.RequestHandler) fasthttp.RequestHandler

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

func chainFastHTTPMiddlewares(handler fasthttp.RequestHandler, middlewares ...fastHTTPMiddleware) fasthttp.RequestHandler {
	for idx := len(middlewares) - 1; idx >= 0; idx-- {
		handler = middlewares[idx](handler)
	}
	return handler
}

func newFastHTTPRequestContext(method, uri string, body []byte) *fasthttp.RequestCtx {
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod(method)
	ctx.Request.SetRequestURI(uri)
	if body != nil {
		ctx.Request.SetBodyRaw(body)
	}
	return ctx
}
