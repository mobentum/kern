package middleware

import (
	"compress/gzip"
	"net/http"
	"strings"
)

func Gzip() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// proceed to next handler if request body can't be compressed
			if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("Vary", "Accept-Encoding")

			gw := &gzipWriter{ResponseWriter: w}
			gw.Header().Set("Content-Encoding", "gzip")

			next.ServeHTTP(gw, r)
		})
	}
}

type gzipWriter struct {
	http.ResponseWriter
	written bool
}

func (gw *gzipWriter) Write(data []byte) (int, error) {
	if gw.written {
		return gw.ResponseWriter.Write(data)
	}

	gw.written = true
	if len(data) == 0 {
		return 0, nil
	}

	gw.Header().Set("Content-Type", http.DetectContentType(data))
	gw.Header().Del("Content-Length")

	gz := gzip.NewWriter(gw.ResponseWriter)
	defer gz.Close()

	return gz.Write(data)
}

func (gw *gzipWriter) WriteHeader(status int) {
	gw.ResponseWriter.WriteHeader(status)
}
