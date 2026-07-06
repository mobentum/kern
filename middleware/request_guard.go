package middleware

import (
	"mime"
	"net/http"
	"strings"

	"github.com/mobentum/kern"
)

// RequestGuardConfig configures request validation guards.
type RequestGuardConfig struct {
	MaxBodyBytes      int64
	RequireBody       bool
	RequireHeaders    []string
	AllowContentTypes []string
	Skip              func(r *http.Request) bool
	StatusCode        int
	Message           string
}

// RequestGuard applies route-level request guards before invoking handlers.
func RequestGuard(configs ...RequestGuardConfig) kern.MiddlewareFunc {
	config := defaultRequestGuardConfig()
	if len(configs) > 0 {
		provided := configs[0]
		if provided.MaxBodyBytes > 0 {
			config.MaxBodyBytes = provided.MaxBodyBytes
		}
		if provided.RequireBody {
			config.RequireBody = true
		}
		if len(provided.RequireHeaders) > 0 {
			config.RequireHeaders = append([]string(nil), provided.RequireHeaders...)
		}
		if len(provided.AllowContentTypes) > 0 {
			config.AllowContentTypes = normalizeContentTypes(provided.AllowContentTypes)
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

			if config.MaxBodyBytes > 0 {
				r.Body = http.MaxBytesReader(w, r.Body, config.MaxBodyBytes)
			}

			hasBody := requestHasBody(r)
			if config.RequireBody && !hasBody {
				http.Error(w, config.Message, config.StatusCode)
				return
			}

			for _, headerName := range config.RequireHeaders {
				if strings.TrimSpace(r.Header.Get(headerName)) == "" {
					http.Error(w, config.Message, config.StatusCode)
					return
				}
			}

			if hasBody && len(config.AllowContentTypes) > 0 {
				mediaType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
				if !containsContentType(config.AllowContentTypes, mediaType) {
					http.Error(w, config.Message, config.StatusCode)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func defaultRequestGuardConfig() RequestGuardConfig {
	return RequestGuardConfig{
		StatusCode: http.StatusBadRequest,
		Message:    "Bad Request",
	}
}

func requestHasBody(r *http.Request) bool {
	if r.ContentLength > 0 {
		return true
	}
	if len(r.TransferEncoding) > 0 {
		return true
	}
	return false
}

func normalizeContentTypes(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		mediaType, _, err := mime.ParseMediaType(value)
		if err != nil {
			mediaType = strings.TrimSpace(strings.ToLower(value))
		}
		if mediaType == "" {
			continue
		}
		normalized = append(normalized, strings.ToLower(mediaType))
	}
	return normalized
}

func containsContentType(allowed []string, mediaType string) bool {
	mediaType = strings.ToLower(strings.TrimSpace(mediaType))
	if mediaType == "" {
		return false
	}
	for _, allowedType := range allowed {
		if mediaType == allowedType {
			return true
		}
	}
	return false
}
