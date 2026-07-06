package kern

import (
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"runtime/debug"
	"sort"
	"strings"
	"time"
)

// BearerAuthConfig configures bearer token authentication middleware.
type BearerAuthConfig struct {
	// Realm is used in the WWW-Authenticate response header.
	Realm string
	// Token is compared against incoming bearer tokens when ValidateToken is nil.
	Token string
	// ValidateToken provides custom token verification logic.
	ValidateToken func(token string, r *http.Request) bool
}

// BasicAuthConfig configures HTTP basic authentication middleware.
type BasicAuthConfig struct {
	// Realm is used in the WWW-Authenticate response header.
	Realm string
	// Username is compared against incoming basic-auth username when ValidateCredentials is nil.
	Username string
	// Password is compared against incoming basic-auth password when ValidateCredentials is nil.
	Password string
	// ValidateCredentials provides custom username/password verification logic.
	ValidateCredentials func(username, password string, r *http.Request) bool
}

// responseWriter extends the http response writer to capture additional details
type responseWriter struct {
	http.ResponseWriter

	status int
	size   int
	// write header only once
	isHeaderWritten bool
}

func (rw *responseWriter) WriteHeader(status int) {
	if rw.isHeaderWritten {
		return
	}

	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
	rw.isHeaderWritten = true
}

func (rw *responseWriter) Write(data []byte) (int, error) {
	// write status header if not done
	if !rw.isHeaderWritten {
		rw.WriteHeader(http.StatusOK)
	}

	size, err := rw.ResponseWriter.Write(data)
	rw.size += size

	return size, err
}

// LoggerConfig configures request logging middleware output.
type LoggerConfig struct {
	Format  string
	Logger  *log.Logger
	SLogger *slog.Logger
	Output  io.Writer
	Fields  map[string]interface{}
}

func Logger(configs ...LoggerConfig) MiddlewareFunc {
	config := defaultLoggerConfig()
	if len(configs) > 0 {
		provided := configs[0]
		if configs[0].Format != "" {
			config.Format = strings.ToLower(configs[0].Format)
		}
		if provided.Logger != nil {
			config.Logger = provided.Logger
		}
		if provided.SLogger != nil {
			config.SLogger = provided.SLogger
		}
		if provided.Output != nil && provided.Logger == nil {
			flags := log.LstdFlags
			if config.Format == "json" {
				flags = 0
			}
			config.Logger = log.New(provided.Output, "", flags)
		}
		if len(provided.Fields) > 0 {
			config.Fields = make(map[string]interface{}, len(provided.Fields))
			for k, v := range provided.Fields {
				config.Fields[k] = v
			}
		}
	}

	if config.Format == "json" && config.Logger == nil {
		config.Logger = log.New(os.Stdout, "", 0)
	}
	if config.Logger == nil {
		config.Logger = log.New(os.Stdout, "", log.LstdFlags)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

			// call next handler in chain
			next.ServeHTTP(rw, r)

			duration := time.Since(start)
			if config.SLogger != nil {
				attrs := []slog.Attr{
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.String("remote_addr", r.RemoteAddr),
					slog.Int("status", rw.status),
					slog.Int64("duration_ms", duration.Milliseconds()),
					slog.Int("size", rw.size),
				}

				if requestID := r.Header.Get("X-Request-ID"); requestID != "" {
					attrs = append(attrs, slog.String("request_id", requestID))
				}

				for _, attr := range mapToSlogAttrs(config.Fields) {
					attrs = append(attrs, attr)
				}

				config.SLogger.LogAttrs(r.Context(), slog.LevelInfo, "http_request", attrs...)
				return
			}

			if config.Format == "json" {
				entry := map[string]interface{}{
					"timestamp":   time.Now().UTC().Format(time.RFC3339Nano),
					"method":      r.Method,
					"path":        r.URL.Path,
					"remote_addr": r.RemoteAddr,
					"status":      rw.status,
					"duration_ms": duration.Milliseconds(),
					"size":        rw.size,
				}

				if requestID := r.Header.Get("X-Request-ID"); requestID != "" {
					entry["request_id"] = requestID
				}

				for key, value := range config.Fields {
					if _, exists := entry[key]; !exists {
						entry[key] = value
					}
				}

				payload, err := json.Marshal(entry)
				if err != nil {
					config.Logger.Printf("[%s] %s %s - %d (%v) %d bytes", r.Method, r.URL.Path, r.RemoteAddr,
						rw.status, duration, rw.size)
					return
				}

				config.Logger.Println(string(payload))
				return
			}

			// log details. [method] /path address status duration size
			textLine := fmt.Sprintf("[%s] %s %s - %d (%v) %d bytes", r.Method, r.URL.Path, r.RemoteAddr,
				rw.status, duration, rw.size)
			if len(config.Fields) > 0 {
				textLine += " " + formatLogFields(config.Fields)
			}

			config.Logger.Print(textLine)
		})
	}
}

func defaultLoggerConfig() LoggerConfig {
	return LoggerConfig{
		Format: "text",
		Logger: log.New(os.Stdout, "", log.LstdFlags),
		Fields: map[string]interface{}{},
	}
}

func formatLogFields(fields map[string]interface{}) string {
	if len(fields) == 0 {
		return ""
	}

	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", key, fields[key]))
	}

	return strings.Join(parts, " ")
}

func mapToSlogAttrs(fields map[string]interface{}) []slog.Attr {
	if len(fields) == 0 {
		return nil
	}

	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	attrs := make([]slog.Attr, 0, len(keys))
	for _, key := range keys {
		attrs = append(attrs, slog.Any(key, fields[key]))
	}

	return attrs
}

func Recovery() MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					// log stack trace
					log.Printf("PANIC: %v\n%s", err, debug.Stack())
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

type CORSConfig struct {
	AllowOrigins      []string
	AllowMethods      []string
	AllowHeaders      []string
	ExposeHeaders     []string
	AllowCredentials  bool
	MaxAge            int
	PreflightContinue bool
}

func CORS(allowOrigins []string) MiddlewareFunc {
	return CORSWithConfig(CORSConfig{
		AllowOrigins: allowOrigins,
	})
}

func CORSWithConfig(config CORSConfig) MiddlewareFunc {
	allowAll := false
	origins := make(map[string]struct{}, len(config.AllowOrigins))

	// validate origin
	for _, origin := range config.AllowOrigins {
		if origin == "*" {
			allowAll = true
			break
		}
		origins[origin] = struct{}{}
	}

	defaultMethods := []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH", "HEAD"}
	defaultHeaders := []string{"Content-Type", "Authorization"}

	allowMethods := config.AllowMethods
	if len(allowMethods) == 0 {
		allowMethods = defaultMethods
	}

	allowHeaders := config.AllowHeaders
	if len(allowHeaders) == 0 {
		allowHeaders = defaultHeaders
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			if allowAll {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else if _, ok := origins[origin]; ok {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
			}

			if config.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			if len(config.ExposeHeaders) > 0 {
				w.Header().Set("Access-Control-Expose-Headers", strings.Join(config.ExposeHeaders, ", "))
			}

			w.Header().Set("Access-Control-Allow-Methods", strings.Join(allowMethods, ", "))
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(allowHeaders, ", "))

			if config.MaxAge > 0 {
				w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", config.MaxAge))
			}

			if r.Method == "OPTIONS" {
				if config.PreflightContinue {
					next.ServeHTTP(w, r)
				} else {
					w.WriteHeader(http.StatusNoContent)
				}
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// BearerAuth validates Authorization: Bearer <token> using a static token.
func BearerAuth(token string) MiddlewareFunc {
	return BearerAuthWithConfig(BearerAuthConfig{Token: token})
}

// BearerAuthWithConfig validates bearer tokens using either Token or ValidateToken.
func BearerAuthWithConfig(config BearerAuthConfig) MiddlewareFunc {
	realm := config.Realm
	if realm == "" {
		realm = "restricted"
	}

	validate := config.ValidateToken
	if validate == nil {
		expected := config.Token
		validate = func(token string, _ *http.Request) bool {
			return expected != "" && token == expected
		}
	}

	challenge := fmt.Sprintf(`Bearer realm=%q`, realm)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := parseBearerToken(r.Header.Get("Authorization"))
			if !ok || !validate(token, r) {
				w.Header().Set("WWW-Authenticate", challenge)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func parseBearerToken(authorization string) (string, bool) {
	if authorization == "" {
		return "", false
	}

	parts := strings.SplitN(strings.TrimSpace(authorization), " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", false
	}

	return token, true
}

// BasicAuth validates Authorization: Basic <base64(username:password)>.
func BasicAuth(username, password string) MiddlewareFunc {
	return BasicAuthWithConfig(BasicAuthConfig{Username: username, Password: password})
}

// BasicAuthWithConfig validates basic-auth credentials using either static credentials or ValidateCredentials.
func BasicAuthWithConfig(config BasicAuthConfig) MiddlewareFunc {
	realm := config.Realm
	if realm == "" {
		realm = "restricted"
	}

	validate := config.ValidateCredentials
	if validate == nil {
		expectedUser := config.Username
		expectedPass := config.Password
		validate = func(username, password string, _ *http.Request) bool {
			if expectedUser == "" || expectedPass == "" {
				return false
			}
			userOK := subtle.ConstantTimeCompare([]byte(username), []byte(expectedUser)) == 1
			passOK := subtle.ConstantTimeCompare([]byte(password), []byte(expectedPass)) == 1
			return userOK && passOK
		}
	}

	challenge := fmt.Sprintf(`Basic realm=%q`, realm)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			username, password, ok := parseBasicCredentials(r.Header.Get("Authorization"))
			if !ok || !validate(username, password, r) {
				w.Header().Set("WWW-Authenticate", challenge)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func parseBasicCredentials(authorization string) (string, string, bool) {
	if authorization == "" {
		return "", "", false
	}

	parts := strings.SplitN(strings.TrimSpace(authorization), " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Basic") {
		return "", "", false
	}

	encoded := strings.TrimSpace(parts[1])
	if encoded == "" {
		return "", "", false
	}

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", "", false
	}

	username, password, ok := strings.Cut(string(decoded), ":")
	if !ok || username == "" {
		return "", "", false
	}

	return username, password, true
}
