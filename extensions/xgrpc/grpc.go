package xgrpc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

var (
	ErrEmptyAddress      = errors.New("xgrpc: addr is required")
	ErrNilRegistration   = errors.New("xgrpc: registration function is nil")
	ErrHealthNotEnabled  = errors.New("xgrpc: health service is not enabled")
)

// Config controls grpc server setup.
type Config struct {
	Addr              string
	Network           string
	Logger            *slog.Logger
	Creds             credentials.TransportCredentials
	EnableHealth      bool
	HealthService     string
	EnableReflection  bool
	ShutdownTimeout   time.Duration
	UnaryInterceptors []grpc.UnaryServerInterceptor
	StreamInterceptors []grpc.StreamServerInterceptor
}

// Registration registers one grpc service against the server.
type Registration struct {
	Name     string
	Register func(grpc.ServiceRegistrar)
}

// Server is a managed grpc server lifecycle wrapper.
type Server struct {
	cfg    Config
	gs     *grpc.Server
	lis    net.Listener
	health *health.Server
}

// Register constructs a grpc server, registers services, and wires optional features.
func Register(cfg Config, regs ...Registration) (*Server, error) {
	return New(cfg, regs...)
}

// New constructs a grpc server, registers services, and wires optional features.
func New(cfg Config, regs ...Registration) (*Server, error) {
	cfg = withDefaults(cfg)
	if cfg.Addr == "" {
		return nil, ErrEmptyAddress
	}

	listener, err := net.Listen(cfg.Network, cfg.Addr)
	if err != nil {
		return nil, fmt.Errorf("xgrpc: listen: %w", err)
	}

	unary := make([]grpc.UnaryServerInterceptor, 0, len(cfg.UnaryInterceptors)+3)
	unary = append(unary,
		UnaryRecovery(RecoveryConfig{Logger: cfg.Logger}),
		UnaryRequestID(RequestIDConfig{}),
		UnaryLogging(LoggingConfig{Logger: cfg.Logger}),
	)
	unary = append(unary, cfg.UnaryInterceptors...)

	stream := make([]grpc.StreamServerInterceptor, 0, len(cfg.StreamInterceptors)+3)
	stream = append(stream,
		StreamRecovery(RecoveryConfig{Logger: cfg.Logger}),
		StreamRequestID(RequestIDConfig{}),
		StreamLogging(LoggingConfig{Logger: cfg.Logger}),
	)
	stream = append(stream, cfg.StreamInterceptors...)

	opts := make([]grpc.ServerOption, 0, 3)
	if cfg.Creds != nil {
		opts = append(opts, grpc.Creds(cfg.Creds))
	}
	if len(unary) > 0 {
		opts = append(opts, grpc.ChainUnaryInterceptor(unary...))
	}
	if len(stream) > 0 {
		opts = append(opts, grpc.ChainStreamInterceptor(stream...))
	}

	gs := grpc.NewServer(opts...)

	for _, reg := range regs {
		if reg.Register == nil {
			listener.Close()
			return nil, ErrNilRegistration
		}
		reg.Register(gs)
	}

	var hs *health.Server
	if cfg.EnableHealth {
		hs = health.NewServer()
		hs.SetServingStatus(cfg.HealthService, healthpb.HealthCheckResponse_SERVING)
		healthpb.RegisterHealthServer(gs, hs)
	}
	if cfg.EnableReflection {
		reflection.Register(gs)
	}

	return &Server{cfg: cfg, gs: gs, lis: listener, health: hs}, nil
}

func withDefaults(cfg Config) Config {
	if cfg.Network == "" {
		cfg.Network = "tcp"
	}
	if cfg.ShutdownTimeout <= 0 {
		cfg.ShutdownTimeout = 10 * time.Second
	}
	return cfg
}

// GRPC exposes the underlying grpc server.
func (s *Server) GRPC() *grpc.Server {
	return s.gs
}

// ListenerAddr returns the bound listener address.
func (s *Server) ListenerAddr() net.Addr {
	return s.lis.Addr()
}

// Run starts serving grpc requests and blocks until server exits.
func (s *Server) Run() error {
	return s.gs.Serve(s.lis)
}

// Shutdown gracefully stops the grpc server until ctx is done, then forces stop.
func (s *Server) Shutdown(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		s.gs.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		s.gs.Stop()
		return ctx.Err()
	}
}

// Stop immediately stops the grpc server.
func (s *Server) Stop() {
	s.gs.Stop()
}

// SetHealthStatus updates grpc health for a given service.
func (s *Server) SetHealthStatus(service string, status healthpb.HealthCheckResponse_ServingStatus) error {
	if s.health == nil {
		return ErrHealthNotEnabled
	}
	s.health.SetServingStatus(service, status)
	return nil
}
