package xotel

import (
	"context"
	"net/http"
	"time"

	"github.com/mobentum/kern"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// SetupConfig configures the global OpenTelemetry tracer provider.
type SetupConfig struct {
	// ServiceName is included as the resource service.name attribute.
	ServiceName string
	// Endpoint is the OTLP HTTP collector endpoint (host:port). Defaults to localhost:4318.
	Endpoint string
	// Insecure disables TLS for the OTLP exporter. Defaults to true (local dev).
	Insecure bool
	// BatchTimeout controls the batch span processor flush interval. Defaults to 5s.
	BatchTimeout time.Duration
}

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

// Setup configures the global tracer provider with an OTLP HTTP exporter and
// the W3C TraceContext + Baggage propagators. It returns a shutdown function
// that flushes pending spans. Call it once at process startup and defer the
// returned function.
func Setup(cfg SetupConfig) (func(), error) {
	if cfg.Endpoint == "" {
		cfg.Endpoint = "localhost:4318"
	}
	if cfg.BatchTimeout <= 0 {
		cfg.BatchTimeout = 5 * time.Second
	}

	opts := []otlptracehttp.Option{otlptracehttp.WithEndpoint(cfg.Endpoint)}
	if cfg.Insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	exporter, err := otlptracehttp.New(context.Background(), opts...)
	if err != nil {
		return nil, err
	}

	res, err := resource.New(context.Background(),
		resource.WithAttributes(semconv.ServiceName(cfg.ServiceName)),
	)
	if err != nil {
		return nil, err
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter, sdktrace.WithBatchTimeout(cfg.BatchTimeout)),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = provider.Shutdown(ctx)
	}, nil
}
