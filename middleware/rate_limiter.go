package middleware

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mobentum/kern"
)

const rateLimiterBucketCount = 16

// RateLimiterConfig configures rate limiter middleware.
type RateLimiterConfig struct {
	Requests int
	Window   time.Duration
	KeyFunc  func(r *http.Request) string
	Skip     func(r *http.Request) bool

	StatusCode int
	Message    string
}

type rateLimitEntry struct {
	count           int
	resetAtUnixNano int64
}

type rateLimiterBucket struct {
	mu      sync.Mutex
	entries map[string]rateLimitEntry
}

// RateLimiter applies fixed-window rate limiting per request key.
func RateLimiter(configs ...RateLimiterConfig) kern.MiddlewareFunc {
	config := defaultRateLimiterConfig()
	if len(configs) > 0 {
		provided := configs[0]
		if provided.Requests > 0 {
			config.Requests = provided.Requests
		}
		if provided.Window > 0 {
			config.Window = provided.Window
		}
		if provided.KeyFunc != nil {
			config.KeyFunc = provided.KeyFunc
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

	var (
		windowNs = int64(config.Window)
		buckets  [rateLimiterBucketCount]rateLimiterBucket
		hits     int64
	)
	if windowNs <= 0 {
		windowNs = int64(time.Second)
	}

	for i := range buckets {
		buckets[i].entries = make(map[string]rateLimitEntry)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if config.Skip != nil && config.Skip(r) {
				next.ServeHTTP(w, r)
				return
			}

			nowUnixNano := time.Now().UnixNano()
			key := config.KeyFunc(r)
			if key == "" {
				key = "global"
			}

			bucketIdx := hashKey(key) % rateLimiterBucketCount
			bucket := &buckets[bucketIdx]
			bucket.mu.Lock()
			entry, ok := bucket.entries[key]
			if !ok || nowUnixNano > entry.resetAtUnixNano {
				entry = rateLimitEntry{count: 0, resetAtUnixNano: nowUnixNano + windowNs}
			}

			if entry.count >= config.Requests {
				remaining := 0
				limit := config.Requests
				resetUnix := entry.resetAtUnixNano / int64(time.Second)
				bucket.mu.Unlock()

				setRateLimitHeaders(w, limit, remaining, resetUnix)
				http.Error(w, config.Message, config.StatusCode)
				return
			}

			entry.count++
			bucket.entries[key] = entry
			remaining := config.Requests - entry.count
			limit := config.Requests
			resetUnix := entry.resetAtUnixNano / int64(time.Second)

			hits++
			if hits%256 == 0 {
				for k, v := range bucket.entries {
					if nowUnixNano > v.resetAtUnixNano+windowNs {
						delete(bucket.entries, k)
					}
				}
			}
			bucket.mu.Unlock()

			setRateLimitHeaders(w, limit, remaining, resetUnix)
			next.ServeHTTP(w, r)
		})
	}
}

func defaultRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		Requests:   100,
		Window:     time.Minute,
		KeyFunc:    rateLimitKeyFromRequest,
		StatusCode: http.StatusTooManyRequests,
		Message:    "Too Many Requests",
	}
}

func setRateLimitHeaders(w http.ResponseWriter, limit, remaining int, resetUnix int64) {
	w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
	w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
	w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetUnix, 10))
}

func rateLimitKeyFromRequest(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		if idx := strings.IndexByte(forwarded, ','); idx > 0 {
			return strings.TrimSpace(forwarded[:idx])
		}
		return strings.TrimSpace(forwarded)
	}

	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}

	if r.RemoteAddr != "" {
		return r.RemoteAddr
	}

	return "global"
}

func hashKey(key string) int {
	h := 5381
	for _, c := range key {
		h = ((h << 5) + h) ^ int(c)
	}
	if h < 0 {
		h = -h
	}
	return h
}
