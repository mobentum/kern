package xotel

import (
	"net/http"

	"github.com/mobentum/kern"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// Config configures the OpenTelemetry middleware.
type Config struct {
	// TracerProvider is the global tracer provider. Defaults to otel.GetTracerProvider().
	TracerProvider trace.TracerProvider
	// Propagator propagates trace context across process boundaries. Defaults to otel.GetTextMapPropagator().
	Propagator propagation.TextMapPropagator
	// ServiceName is included as a span attribute on every request span.
	ServiceName string
	// Skip allows skipping tracing for specific requests.
	Skip func(r *http.Request) bool
}

// Middleware returns a kern middleware that creates a tracing span for each request.
func Middleware(configs ...Config) kern.MiddlewareFunc {
	cfg := defaultConfig()
	if len(configs) > 0 {
		provided := configs[0]
		if provided.TracerProvider != nil {
			cfg.TracerProvider = provided.TracerProvider
		}
		if provided.Propagator != nil {
			cfg.Propagator = provided.Propagator
		}
		if provided.ServiceName != "" {
			cfg.ServiceName = provided.ServiceName
		}
		if provided.Skip != nil {
			cfg.Skip = provided.Skip
		}
	}

	tracer := cfg.TracerProvider.Tracer("github.com/mobentum/kern/extensions/xotel")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.Skip != nil && cfg.Skip(r) {
				next.ServeHTTP(w, r)
				return
			}

			ctx := cfg.Propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
			spanName := r.Method + " " + r.URL.Path

			ctx, span := tracer.Start(ctx, spanName,
				trace.WithAttributes(
					attribute.String("http.request.method", r.Method),
					attribute.String("url.path", r.URL.Path),
					attribute.String("url.query", r.URL.RawQuery),
					attribute.String("network.protocol.version", r.Proto),
					attribute.String("user_agent.original", r.UserAgent()),
				),
				trace.WithSpanKind(trace.SpanKindServer),
			)
			defer span.End()

			if cfg.ServiceName != "" {
				span.SetAttributes(attribute.String("service.name", cfg.ServiceName))
			}

			rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rw, r.WithContext(ctx))

			span.SetAttributes(
				attribute.Int("http.response.status_code", rw.status),
			)
			if rw.status >= 500 {
				span.SetAttributes(attribute.String("error.type", "http_error"))
			}
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func defaultConfig() Config {
	return Config{
		TracerProvider: otel.GetTracerProvider(),
		Propagator:     otel.GetTextMapPropagator(),
	}
}
