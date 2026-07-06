package middleware

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/mobentum/kern"
)

type csrfTokenContextKey struct{}

// CSRFConfig configures CSRF middleware behavior.
type CSRFConfig struct {
	CookieName     string
	HeaderName     string
	FormField      string
	TokenLength    int
	CookiePath     string
	CookieDomain   string
	CookieSecure   bool
	CookieHTTPOnly bool
	CookieSameSite http.SameSite
	CookieMaxAge   int
	SkipSafe       bool
	ErrorHandler   func(w http.ResponseWriter, r *http.Request)
}

// CSRF protects unsafe HTTP methods using double-submit cookie tokens.
func CSRF(configs ...CSRFConfig) kern.MiddlewareFunc {
	config := defaultCSRFConfig()
	if len(configs) > 0 {
		provided := configs[0]
		if provided.CookieName != "" {
			config.CookieName = provided.CookieName
		}
		if provided.HeaderName != "" {
			config.HeaderName = provided.HeaderName
		}
		if provided.FormField != "" {
			config.FormField = provided.FormField
		}
		if provided.TokenLength > 0 {
			config.TokenLength = provided.TokenLength
		}
		if provided.CookiePath != "" {
			config.CookiePath = provided.CookiePath
		}
		if provided.CookieDomain != "" {
			config.CookieDomain = provided.CookieDomain
		}
		config.CookieSecure = provided.CookieSecure
		config.CookieHTTPOnly = provided.CookieHTTPOnly
		if provided.CookieSameSite != 0 {
			config.CookieSameSite = provided.CookieSameSite
		}
		if provided.CookieMaxAge != 0 {
			config.CookieMaxAge = provided.CookieMaxAge
		}
		if provided.ErrorHandler != nil {
			config.ErrorHandler = provided.ErrorHandler
		}
		config.SkipSafe = provided.SkipSafe
	}

	if config.ErrorHandler == nil {
		config.ErrorHandler = defaultCSRFFailure
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := ""
			if cookie, err := r.Cookie(config.CookieName); err == nil {
				token = cookie.Value
			}

			if token == "" {
				generated, err := generateCSRFToken(config.TokenLength)
				if err != nil {
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
				token = generated
				setCSRFCookie(w, config, token)
			}

			ctx := context.WithValue(r.Context(), csrfTokenContextKey{}, token)
			r = r.WithContext(ctx)

			if config.SkipSafe && isSafeMethod(r.Method) {
				next.ServeHTTP(w, r)
				return
			}

			requestToken := strings.TrimSpace(r.Header.Get(config.HeaderName))
			if requestToken == "" {
				requestToken = strings.TrimSpace(r.FormValue(config.FormField))
			}

			if requestToken == "" || subtle.ConstantTimeCompare([]byte(requestToken), []byte(token)) != 1 {
				config.ErrorHandler(w, r)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// CSRFToken returns the token attached to the request context by CSRF middleware.
func CSRFToken(ctx context.Context) (string, bool) {
	token, ok := ctx.Value(csrfTokenContextKey{}).(string)
	return token, ok && token != ""
}

func defaultCSRFConfig() CSRFConfig {
	return CSRFConfig{
		CookieName:     "_csrf",
		HeaderName:     "X-CSRF-Token",
		FormField:      "_csrf",
		TokenLength:    32,
		CookiePath:     "/",
		CookieHTTPOnly: false,
		CookieSameSite: http.SameSiteLaxMode,
		SkipSafe:       true,
	}
}

func defaultCSRFFailure(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "Forbidden", http.StatusForbidden)
}

func generateCSRFToken(length int) (string, error) {
	if length <= 0 {
		length = 32
	}

	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("csrf: token generation failed: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func setCSRFCookie(w http.ResponseWriter, config CSRFConfig, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     config.CookieName,
		Value:    token,
		Path:     config.CookiePath,
		Domain:   config.CookieDomain,
		Secure:   config.CookieSecure,
		HttpOnly: config.CookieHTTPOnly,
		SameSite: config.CookieSameSite,
		MaxAge:   config.CookieMaxAge,
	})
}

func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
}
