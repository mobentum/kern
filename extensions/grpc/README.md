# grpc - lightweight gRPC lifecycle for kern

`grpcx` is an optional module that reduces repetitive gRPC server setup while keeping contracts explicit.

- Standalone module: `github.com/mobentum/kern/extensions/grpc`
- Explicit service registration (you own your protobuf contracts)
- Built-in interceptors for recovery, request-id propagation, and logging
- Optional health and reflection registration

## Install

```bash
go get github.com/mobentum/kern/extensions/grpc
```

## Quick Start

```go
package main

import (
    "context"
    "log/slog"

    grpcx "github.com/mobentum/kern/extensions/grpc"
)

func main() {
    logger := slog.Default()

    srv, err := grpcx.Register(grpcx.Config{
        Addr:             ":9090",
        Logger:           logger,
        EnableHealth:     true,
        EnableReflection: true,
    })
    if err != nil {
        panic(err)
    }

    go func() {
        _ = srv.Run()
    }()

    // ... signal handling, app lifecycle, etc.
    _ = srv.Shutdown(context.Background())
}
```

## Config

```go
type Config struct {
    Addr               string
    Network            string // default: tcp
    Logger             *slog.Logger
    Creds              credentials.TransportCredentials
    EnableHealth       bool
    HealthService      string
    EnableReflection   bool
    ShutdownTimeout    time.Duration
    UnaryInterceptors  []grpc.UnaryServerInterceptor
    StreamInterceptors []grpc.StreamServerInterceptor
}
```

## Interceptors

Built-in interceptors are applied automatically:

- `UnaryRecovery` / `StreamRecovery`
- `UnaryRequestID` / `StreamRequestID` (metadata key defaults to `x-request-id`)
- `UnaryLogging` / `StreamLogging`

Additional interceptors can be appended using `Config.UnaryInterceptors` and `Config.StreamInterceptors`.

## Health and Reflection

- Set `EnableHealth: true` to register the standard grpc health service.
- Set `EnableReflection: true` to register grpc reflection.
- Update health at runtime using `SetHealthStatus(service, status)`.

## Runnable Example

A complete kern + grpc lifecycle sample app lives in:

- `extensions/grpc/examples/kern-integration`

Run it locally:

```bash
cd extensions/grpc/examples/kern-integration
go run .
```
