package middleware

import (
	"net/http"
	"time"

	"github.com/mobentum/kern"
)

// TimeoutConfig configures timeout middleware behavior.
type TimeoutConfig struct {
	Duration time.Duration
	Message  string
}

// Timeout applies a request timeout using net/http TimeoutHandler.
func Timeout(configs ...TimeoutConfig) kern.MiddlewareFunc {
	config := TimeoutConfig{
		Duration: 30 * time.Second,
		Message:  "Service Unavailable",
	}

	if len(configs) > 0 {
		provided := configs[0]
		if provided.Duration > 0 {
			config.Duration = provided.Duration
		}
		if provided.Message != "" {
			config.Message = provided.Message
		}
	}

	return func(next http.Handler) http.Handler {
		return http.TimeoutHandler(next, config.Duration, config.Message)
	}
}
