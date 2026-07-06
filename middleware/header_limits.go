package middleware

import (
	"net/http"

	"github.com/mobentum/kern"
)

// HeaderLimitsConfig configures request header guard middleware.
type HeaderLimitsConfig struct {
	MaxHeaderCount int
	MaxHeaderBytes int
	Skip           func(r *http.Request) bool

	StatusCode int
	Message    string
}

// HeaderLimits rejects requests that exceed configured header count or total header bytes.
func HeaderLimits(configs ...HeaderLimitsConfig) kern.MiddlewareFunc {
	config := defaultHeaderLimitsConfig()
	if len(configs) > 0 {
		provided := configs[0]
		if provided.MaxHeaderCount > 0 {
			config.MaxHeaderCount = provided.MaxHeaderCount
		}
		if provided.MaxHeaderBytes > 0 {
			config.MaxHeaderBytes = provided.MaxHeaderBytes
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

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if config.Skip != nil && config.Skip(r) {
				next.ServeHTTP(w, r)
				return
			}

			headerCount, headerBytes := headerStats(r.Header)
			if config.MaxHeaderCount > 0 && headerCount > config.MaxHeaderCount {
				http.Error(w, config.Message, config.StatusCode)
				return
			}
			if config.MaxHeaderBytes > 0 && headerBytes > config.MaxHeaderBytes {
				http.Error(w, config.Message, config.StatusCode)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func defaultHeaderLimitsConfig() HeaderLimitsConfig {
	return HeaderLimitsConfig{
		MaxHeaderCount: 100,
		MaxHeaderBytes: 16 * 1024,
		StatusCode:     http.StatusRequestHeaderFieldsTooLarge,
		Message:        "Request Header Fields Too Large",
	}
}

func headerStats(headers http.Header) (int, int) {
	count := 0
	bytes := 0
	for key, values := range headers {
		count++
		bytes += len(key)
		for _, value := range values {
			bytes += len(value)
		}
	}
	return count, bytes
}
