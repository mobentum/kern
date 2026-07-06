package middleware

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/mobentum/kern"
)

var ErrResponseTooLarge = errors.New("response body too large")

// ResponseLimitConfig configures response size limiting.
type ResponseLimitConfig struct {
	MaxBytes   int64
	Skip       func(r *http.Request) bool
	StatusCode int
	Message    string
}

// ResponseLimit limits response body bytes for routes where it is applied.
func ResponseLimit(configs ...ResponseLimitConfig) kern.MiddlewareFunc {
	config := defaultResponseLimitConfig()
	if len(configs) > 0 {
		provided := configs[0]
		if provided.MaxBytes > 0 {
			config.MaxBytes = provided.MaxBytes
		}
		if provided.Skip != nil {
			config.Skip = provided.Skip
		}
		if provided.StatusCode > 0 {
			config.StatusCode = provided.StatusCode
		}
		if provided.Message != "" {
			config.Message = provided.Message
		}
	}

	if config.MaxBytes <= 0 {
		panic("middleware.ResponseLimit: MaxBytes must be > 0")
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if config.Skip != nil && config.Skip(r) {
				next.ServeHTTP(w, r)
				return
			}

			lw := &responseLimitWriter{
				ResponseWriter: w,
				limit:          config.MaxBytes,
				statusCode:     config.StatusCode,
				message:        config.Message,
			}
			next.ServeHTTP(lw, r)
		})
	}
}

func defaultResponseLimitConfig() ResponseLimitConfig {
	return ResponseLimitConfig{
		StatusCode: http.StatusInternalServerError,
		Message:    "Internal Server Error",
	}
}

type responseLimitWriter struct {
	http.ResponseWriter
	limit       int64
	written     int64
	statusCode  int
	message     string
	blocked     bool
	headWritten bool
}

func (w *responseLimitWriter) WriteHeader(statusCode int) {
	if w.blocked {
		return
	}

	if !w.headWritten {
		if contentLength := w.Header().Get("Content-Length"); contentLength != "" {
			if n, err := strconv.ParseInt(contentLength, 10, 64); err == nil && n > w.limit {
				w.blocked = true
				http.Error(w.ResponseWriter, w.message, w.statusCode)
				w.headWritten = true
				return
			}
		}
	}

	w.headWritten = true
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseLimitWriter) Write(data []byte) (int, error) {
	if w.blocked {
		return 0, ErrResponseTooLarge
	}

	n := int64(len(data))
	if w.written+n > w.limit {
		w.blocked = true
		if w.written == 0 && !w.headWritten {
			http.Error(w.ResponseWriter, w.message, w.statusCode)
			w.headWritten = true
		}
		return 0, ErrResponseTooLarge
	}

	w.headWritten = true
	w.written += n
	return w.ResponseWriter.Write(data)
}
