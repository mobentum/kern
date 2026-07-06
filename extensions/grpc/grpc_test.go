package grpcx

import (
	"context"
	"errors"
	"testing"
	"time"

	"google.golang.org/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func TestNewRequiresAddress(t *testing.T) {
	_, err := New(Config{})
	if !errors.Is(err, ErrEmptyAddress) {
		t.Fatalf("expected ErrEmptyAddress, got %v", err)
	}
}

func TestNewRejectsNilRegistration(t *testing.T) {
	_, err := New(Config{Addr: "127.0.0.1:0"}, Registration{Name: "bad", Register: nil})
	if !errors.Is(err, ErrNilRegistration) {
		t.Fatalf("expected ErrNilRegistration, got %v", err)
	}
}

func TestHealthToggle(t *testing.T) {
	srv, err := New(Config{Addr: "127.0.0.1:0"})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	defer srv.Stop()

	if err := srv.SetHealthStatus("", healthpb.HealthCheckResponse_SERVING); !errors.Is(err, ErrHealthNotEnabled) {
		t.Fatalf("expected ErrHealthNotEnabled, got %v", err)
	}
}

func TestRunAndShutdown(t *testing.T) {
	srv, err := New(Config{Addr: "127.0.0.1:0", EnableHealth: true})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Run()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			t.Fatalf("run error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not exit after shutdown")
	}

	if err := srv.SetHealthStatus("", healthpb.HealthCheckResponse_NOT_SERVING); err != nil {
		t.Fatalf("set health status: %v", err)
	}
}
