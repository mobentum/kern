package middleware

import (
	"net/http"

	"github.com/mobentum/kern"
)

// SecurityHeadersConfig configures helmet-style response headers.
type SecurityHeadersConfig struct {
	XContentTypeOptions       string
	XFrameOptions             string
	ReferrerPolicy            string
	XXSSProtection            string
	ContentSecurityPolicy     string
	PermissionsPolicy         string
	StrictTransportSecurity   string
	CrossOriginOpenerPolicy   string
	CrossOriginResourcePolicy string
	CrossOriginEmbedderPolicy string
}

// SecurityHeaders applies a safe default set of security-oriented response headers.
func SecurityHeaders(configs ...SecurityHeadersConfig) kern.MiddlewareFunc {
	config := defaultSecurityHeadersConfig()
	if len(configs) > 0 {
		provided := configs[0]
		if provided.XContentTypeOptions != "" {
			config.XContentTypeOptions = provided.XContentTypeOptions
		}
		if provided.XFrameOptions != "" {
			config.XFrameOptions = provided.XFrameOptions
		}
		if provided.ReferrerPolicy != "" {
			config.ReferrerPolicy = provided.ReferrerPolicy
		}
		if provided.XXSSProtection != "" {
			config.XXSSProtection = provided.XXSSProtection
		}
		if provided.ContentSecurityPolicy != "" {
			config.ContentSecurityPolicy = provided.ContentSecurityPolicy
		}
		if provided.PermissionsPolicy != "" {
			config.PermissionsPolicy = provided.PermissionsPolicy
		}
		if provided.StrictTransportSecurity != "" {
			config.StrictTransportSecurity = provided.StrictTransportSecurity
		}
		if provided.CrossOriginOpenerPolicy != "" {
			config.CrossOriginOpenerPolicy = provided.CrossOriginOpenerPolicy
		}
		if provided.CrossOriginResourcePolicy != "" {
			config.CrossOriginResourcePolicy = provided.CrossOriginResourcePolicy
		}
		if provided.CrossOriginEmbedderPolicy != "" {
			config.CrossOriginEmbedderPolicy = provided.CrossOriginEmbedderPolicy
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			setHeaderIfEmpty(w, "X-Content-Type-Options", config.XContentTypeOptions)
			setHeaderIfEmpty(w, "X-Frame-Options", config.XFrameOptions)
			setHeaderIfEmpty(w, "Referrer-Policy", config.ReferrerPolicy)
			setHeaderIfEmpty(w, "X-XSS-Protection", config.XXSSProtection)
			setHeaderIfEmpty(w, "Content-Security-Policy", config.ContentSecurityPolicy)
			setHeaderIfEmpty(w, "Permissions-Policy", config.PermissionsPolicy)
			setHeaderIfEmpty(w, "Cross-Origin-Opener-Policy", config.CrossOriginOpenerPolicy)
			setHeaderIfEmpty(w, "Cross-Origin-Resource-Policy", config.CrossOriginResourcePolicy)
			setHeaderIfEmpty(w, "Cross-Origin-Embedder-Policy", config.CrossOriginEmbedderPolicy)

			if config.StrictTransportSecurity != "" && r.TLS != nil {
				setHeaderIfEmpty(w, "Strict-Transport-Security", config.StrictTransportSecurity)
			}

			next.ServeHTTP(w, r)
		})
	}
}

func defaultSecurityHeadersConfig() SecurityHeadersConfig {
	return SecurityHeadersConfig{
		XContentTypeOptions:       "nosniff",
		XFrameOptions:             "DENY",
		ReferrerPolicy:            "no-referrer",
		XXSSProtection:            "0",
		CrossOriginOpenerPolicy:   "same-origin",
		CrossOriginResourcePolicy: "same-origin",
	}
}

func setHeaderIfEmpty(w http.ResponseWriter, key, value string) {
	if value == "" {
		return
	}
	if w.Header().Get(key) != "" {
		return
	}
	w.Header().Set(key, value)
}
