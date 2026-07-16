package xgrpc

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type requestIDContextKey struct{}

// LoggingConfig controls grpc access log behavior.
type LoggingConfig struct {
	Logger *slog.Logger
	Slow   time.Duration
}

// RecoveryConfig controls panic recovery logging.
type RecoveryConfig struct {
	Logger *slog.Logger
}

// RequestIDConfig controls request-id propagation.
type RequestIDConfig struct {
	MetadataKey string
}

// AuthFunc validates incoming calls.
type AuthFunc func(ctx context.Context, fullMethod string) error

// UnaryLogging emits a structured log per unary grpc call.
func UnaryLogging(cfg LoggingConfig) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		dur := time.Since(start)

		if cfg.Logger != nil {
			code := status.Code(err)
			attrs := []any{"method", info.FullMethod, "duration", dur.String(), "code", code.String()}
			if requestID := RequestIDFromContext(ctx); requestID != "" {
				attrs = append(attrs, "request_id", requestID)
			}
			if err != nil {
				cfg.Logger.Error("grpc_unary", append(attrs, "error", err.Error())...)
			} else {
				cfg.Logger.Info("grpc_unary", attrs...)
			}
		}

		return resp, err
	}
}

// StreamLogging emits start/end logs for stream grpc calls.
func StreamLogging(cfg LoggingConfig) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		err := handler(srv, ss)
		dur := time.Since(start)

		if cfg.Logger != nil {
			code := status.Code(err)
			attrs := []any{"method", info.FullMethod, "duration", dur.String(), "code", code.String()}
			if requestID := RequestIDFromContext(ss.Context()); requestID != "" {
				attrs = append(attrs, "request_id", requestID)
			}
			if err != nil {
				cfg.Logger.Error("grpc_stream", append(attrs, "error", err.Error())...)
			} else {
				cfg.Logger.Info("grpc_stream", attrs...)
			}
		}

		return err
	}
}

// UnaryRecovery recovers panics and returns internal grpc errors.
func UnaryRecovery(cfg RecoveryConfig) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (_ interface{}, err error) {
		defer func() {
			if recovered := recover(); recovered != nil {
				if cfg.Logger != nil {
					cfg.Logger.Error("grpc_unary_panic", "method", info.FullMethod, "panic", fmt.Sprint(recovered), "stack", string(debug.Stack()))
				}
				err = status.Error(codes.Internal, "internal server error")
			}
		}()
		return handler(ctx, req)
	}
}

// StreamRecovery recovers panics and returns internal grpc errors.
func StreamRecovery(cfg RecoveryConfig) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		defer func() {
			if recovered := recover(); recovered != nil {
				if cfg.Logger != nil {
					cfg.Logger.Error("grpc_stream_panic", "method", info.FullMethod, "panic", fmt.Sprint(recovered), "stack", string(debug.Stack()))
				}
				err = status.Error(codes.Internal, "internal server error")
			}
		}()
		return handler(srv, ss)
	}
}

// UnaryRequestID stores x-request-id metadata into context.
func UnaryRequestID(cfg RequestIDConfig) grpc.UnaryServerInterceptor {
	key := cfg.MetadataKey
	if key == "" {
		key = "x-request-id"
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		requestID := metadataValue(ctx, key)
		if requestID != "" {
			ctx = context.WithValue(ctx, requestIDContextKey{}, requestID)
		}
		return handler(ctx, req)
	}
}

// StreamRequestID stores x-request-id metadata into stream context.
func StreamRequestID(cfg RequestIDConfig) grpc.StreamServerInterceptor {
	key := cfg.MetadataKey
	if key == "" {
		key = "x-request-id"
	}

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()
		requestID := metadataValue(ctx, key)
		if requestID == "" {
			return handler(srv, ss)
		}

		wrapped := &contextServerStream{ServerStream: ss, ctx: context.WithValue(ctx, requestIDContextKey{}, requestID)}
		return handler(srv, wrapped)
	}
}

// RequestIDFromContext returns request id propagated by request-id interceptors.
func RequestIDFromContext(ctx context.Context) string {
	value, _ := ctx.Value(requestIDContextKey{}).(string)
	return value
}

// UnaryAuth validates unary calls before handler execution.
func UnaryAuth(fn AuthFunc) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if fn != nil {
			if err := fn(ctx, info.FullMethod); err != nil {
				return nil, err
			}
		}
		return handler(ctx, req)
	}
}

// StreamAuth validates stream calls before handler execution.
func StreamAuth(fn AuthFunc) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if fn != nil {
			if err := fn(ss.Context(), info.FullMethod); err != nil {
				return err
			}
		}
		return handler(srv, ss)
	}
}

type contextServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *contextServerStream) Context() context.Context {
	return s.ctx
}

func metadataValue(ctx context.Context, key string) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	values := md.Get(key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
