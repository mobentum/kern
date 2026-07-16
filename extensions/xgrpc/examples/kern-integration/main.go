package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mobentum/kern"
	xgrpc "github.com/mobentum/kern/extensions/xgrpc"
)

func main() {
	logger := slog.Default()

	app := kern.New(kern.WithSlogLogger(logger))
	app.GET("/health", func(c *kern.Context) {
		_ = c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: app,
	}

	grpcServer, err := xgrpc.Register(xgrpc.Config{
		Addr:             ":9090",
		Logger:           logger,
		EnableHealth:     true,
		EnableReflection: true,
	})
	if err != nil {
		panic(err)
	}

	errCh := make(chan error, 2)
	go func() {
		errCh <- httpServer.ListenAndServe()
	}()
	go func() {
		errCh <- grpcServer.Run()
	}()

	stopSignals := make(chan os.Signal, 1)
	signal.Notify(stopSignals, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-stopSignals:
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_ = httpServer.Shutdown(shutdownCtx)
		_ = grpcServer.Shutdown(shutdownCtx)
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server exited", "error", err.Error())
		}
	}
}
