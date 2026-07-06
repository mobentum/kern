# Kern - The Web Kernel for Go

![kern logo](docs/public/branding/kern-logo-horizontal.svg)

[![Go Reference](https://pkg.go.dev/badge/github.com/mobentum/kern.svg)](https://pkg.go.dev/github.com/mobentum/kern)
[![Go Report Card](https://goreportcard.com/badge/github.com/mobentum/kern)](https://goreportcard.com/report/github.com/mobentum/kern)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![CI](https://github.com/mobentum/kern/actions/workflows/test.yml/badge.svg)](https://github.com/mobentum/kern/actions/workflows/test.yml)
[![Release](https://img.shields.io/github/release/mobentum/kern.svg)](https://github.com/mobentum/kern/releases)

Kern is a lightweight web framework for Go focused on a small, reliable core so application code stays fast, explicit, and maintainable.

## Why the name "kern"

"kern" comes from "kernel".

In operating systems, a kernel is the smallest trusted core that coordinates everything else. In the same spirit, Kern aims to be a small web kernel:

- strong primitives
- minimal policy
- clean extension points
- no unnecessary magic

The goal is not to be the biggest framework. The goal is to be a dependable core you can build on for years.

## Motivation

Kern was created to reduce the gap between everyday HTTP work and the Go standard library. It is designed for teams that want a practical framework without hidden runtime complexity:

- embraces `net/http` instead of hiding it
- avoids dependency sprawl in the core runtime
- stays explicit, testable, and easy to reason about
- performs well without clever abstractions everywhere

The framework stays intentionally small so engineers can understand internals quickly, debug behavior confidently, and extend only what they need.

## Inspiration

Kern is inspired by:

- **Go stdlib design**: composable interfaces and explicit behavior
- **Bottle (Python)**: minimal API surface with practical defaults
- **microkernel philosophy**: small trusted core with optional modules around it

## Features

- **Go 1.22+ native routing** via `http.ServeMux`
- **Named routes & path constraints** — typed URL params (`kern.UintPathConstraint`) and route lookup by name
- **Built-in auth** — `BearerAuth` and `BasicAuth` middleware ship in core
- **Route-specific middleware** — apply guards per route with `RouteWithMiddleware()`, no group nesting needed
- **Structured request binding** — `Bind()` / `BindQuery()` / `BindForm()` / `BindHeader()` with struct tags
- **File handling** — multipart upload (`SaveFile`), download (`DownloadFile`), streaming with range support (`StreamFile`)
- **Conditional requests** — built-in `ETag`, `LastModified`, `If-None-Match` / `If-Modified-Since` evaluation
- **Context pooling** for lower allocation pressure
- **Middleware chaining** with standard `func(http.Handler) http.Handler`
- **Route groups** for shared prefixes and middleware
- **Graceful shutdown options** for production servers
- **Zero core dependencies**

## Installation

```bash
go get github.com/mobentum/kern
```

## Quality Baseline (Go)

Industry-style local checks for this repository:

```bash
# one-time tool install
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install golang.org/x/vuln/cmd/govulncheck@latest

# quality + tests
make go-quality
go test ./...
```

CI check order:

1. `go vet` (correctness smells)
2. `golangci-lint` (static lint suite)
3. `govulncheck` (vulnerability scan)
4. `go build` + `go test`

## Test Status

| Package | Tests | Build | Coverage |
|---------|------:|-------|---------:|
| `kern` (core) | 104 | ✅ | 81.4% |
| `kern/middleware` | 55 | ✅ | 82.5% |
| **Total** | **159** | **✅** | **~82%** |

All tests pass with zero failures across both packages.

## Quick Start

```go
package main

import "github.com/mobentum/kern"

func main() {
    app := kern.Default()

    app.GET("/", func(c *kern.Context) {
        c.JSON(200, map[string]string{"message": "hello from kern"})
    })

    app.GET("/users/{id}", func(c *kern.Context) {
        c.JSON(200, map[string]string{"id": c.Param("id")})
    })

    _ = app.Run(":8080")
}
```

## Core API

### App

```go
app := kern.New()
app := kern.Default()

app.Use(mw1, mw2)
app.GET("/", handler)
app.POST("/users", handler)
app.Group("/api")
app.Static("/static/", "./public")

app.Run(":8080")
app.RunTLS(":8443", "cert.pem", "key.pem")

app.RouteWithConstraints("GET", "/users/{id}", kern.PathConstraints{
    "id": kern.UintPathConstraint,
}, handler)

app.RouteNamedWithConstraints("users_show", "GET", "/users/{id}", kern.PathConstraints{
    "id": kern.UintPathConstraint,
}, handler)
```

### Context

Request helpers:

- `Param(name)`
- `Query(name)` / `DefaultQuery(name, fallback)`
- `QueryPair(name1, name2)` / `QueryPairDefault(name1, fallback1, name2, fallback2)`
- `QueryPairRaw(name1, name2)` / `QueryPairDefaultRaw(name1, fallback1, name2, fallback2)`
- `QueryInt(name, fallback)` / `QueryBool(name, fallback)`
- `Form(name)`
- `Cookie(name)`
- `GetHeader(name)` / `HeaderInt(name, fallback)` / `HeaderBool(name, fallback)`
- `ClientIP()`
- `Body()`

Response helpers:

- `JSON(status, value)`
- `OK(value)` / `Created(value)` / `Accepted(value)`
- `Text(status, format, args...)`
- `HTML(status, html)`
- `XML(status, value)`
- `Data(status, contentType, bytes)`
- `NoContent(status)`
- `Status(status)`
- `JSONError(status, message, details...)`
- `ETag(value)` / `LastModified(time)` / `IsNotModified(etag, modTime)`
- `CheckPreconditions(etag, modTime)`
- `Redirect(status, location)`

Proxy safety options:

- `WithTrustedProxies(...)` limits which upstream hops can supply forwarding headers.
- `WithStrictProxyHeaders(true)` enforces strict parsing for `X-Forwarded-For` and `X-Real-IP`.
- `WithStrictRequestParsing(true)` rejects malformed query strings in bind helpers.

## Middleware

Built-in in core package:

- `kern.Logger()`
- `kern.Recovery()`
- `kern.CORS(...)`

Extra middleware package:

- `github.com/mobentum/kern/middleware`

Session middleware example:

```go
import kmw "github.com/mobentum/kern/middleware"

app.Use(kmw.Session(kmw.SessionConfig{SigningKey: []byte("replace-with-strong-secret")}))

app.GET("/login", func(c *kern.Context) {
    session, _ := kmw.GetSession(c.Context())
    session.Set("user_id", "123")
    _ = c.Text(http.StatusOK, "ok")
})
```

Route-level request guard example:

```go
app.RouteWithMiddleware(http.MethodPost, "/upload", uploadHandler,
    kmw.RequestGuard(kmw.RequestGuardConfig{
        MaxBodyBytes:      8 << 20,
        RequireBody:       true,
        RequireHeaders:    []string{"X-Tenant"},
        AllowContentTypes: []string{"application/json", "multipart/form-data"},
    }),
)
```

## Optional Packages

Core Kern remains dependency-free. Optional integrations can live in separate modules.

### Structured Logging (`github.com/mobentum/kern/extensions/xlog`)

`xlog` provides a `slog`-compatible logger backed by zerolog.

See full package docs and examples in [extensions/xlog/README.md](extensions/xlog/README.md).

Runnable integration example: [extensions/xlog/examples/kern-integration](extensions/xlog/examples/kern-integration).

Install:

```bash
go get github.com/mobentum/kern/extensions/xlog
```

Use for app lifecycle logs:

```go
import (
    "github.com/mobentum/kern"
    "github.com/mobentum/kern/extensions/xlog"
)

app := kern.New(
    kern.WithSlogLogger(xlog.NewLogger(xlog.Config{Format: "json"})),
)
```

Use for request middleware logs:

```go
import (
    "github.com/mobentum/kern"
    "github.com/mobentum/kern/extensions/xlog"
)

reqLogger := xlog.NewLogger(xlog.Config{Format: "console"})

app := kern.New()
app.Use(kern.Logger(kern.LoggerConfig{
    SLogger: reqLogger,
    Fields: map[string]interface{}{
        "service": "users-api",
        "env":     "prod",
    },
}))
```

### Configuration (`github.com/mobentum/kern/extensions/config`)

`config` provides dotenv-style loading and typed environment access through a small loader API.

See full package docs and examples in [extensions/config/README.md](extensions/config/README.md).

Runnable integration example: [extensions/config/examples/kern-integration](extensions/config/examples/kern-integration).

Install:

```bash
go get github.com/mobentum/kern/extensions/config
```

Use for app configuration:

```go
import (
    "github.com/mobentum/kern/extensions/config"
)

type Config struct {
    Host string
    Port int
}

func LoadConfig() (*Config, error) {
    loader, err := config.New(
        config.WithPrefix("APP"),
        config.WithDotEnv(".env"),
    )
    if err != nil {
        return nil, err
    }

    port, err := loader.Int("PORT", 8080)
    if err != nil {
        return nil, err
    }

    return &Config{
        Host: loader.String("HOST", "127.0.0.1"),
        Port: port,
    }, nil
}
```

### OpenAPI (`github.com/mobentum/kern/extensions/openapi`)

`openapi` exposes a simple, explicit OpenAPI JSON endpoint and Swagger UI page.

See full package docs and examples in [extensions/openapi/README.md](extensions/openapi/README.md).

Runnable integration example: [extensions/openapi/examples/kern-integration](extensions/openapi/examples/kern-integration).

Install:

```bash
go get github.com/mobentum/kern/extensions/openapi
```

Use for API docs endpoints:

```go
import (
    "net/http"

    "github.com/mobentum/kern/extensions/openapi"
)

openapi.Register(app, openapi.Config{
    Info: openapi.Info{Title: "Users API", Version: "1.0.0"},
    Routes: []openapi.Route{
        {
            Method:      http.MethodGet,
            Path:        "/users/{id}",
            Summary:     "Get user",
            OperationID: "getUser",
            Tags:        []string{"users"},
        },
    },
})
```

### gRPC (`github.com/mobentum/kern/extensions/grpc`)

`grpc` provides explicit gRPC server lifecycle management with optional health and reflection registration.

See full package docs and examples in [extensions/grpc/README.md](extensions/grpc/README.md).

Runnable integration example: [extensions/grpc/examples/kern-integration](extensions/grpc/examples/kern-integration).

Install:

```bash
go get github.com/mobentum/kern/extensions/grpc
```

Use for gRPC server startup:

```go
import (
    "context"
    "time"

    grpcx "github.com/mobentum/kern/extensions/grpc"
)

srv, err := grpcx.Register(grpcx.Config{
    Addr:             ":9090",
    EnableHealth:     true,
    EnableReflection: true,
})
if err != nil {
    panic(err)
}

go func() {
    _ = srv.Run()
}()

shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
_ = srv.Shutdown(shutdownCtx)
```

## Examples

Real runnable examples are in [examples](examples):

- [examples/basic](examples/basic)
- [examples/middleware](examples/middleware)
- [examples/nested-routes](examples/nested-routes)
- [examples/rest-api](examples/rest-api)
- [examples/file-upload](examples/file-upload)
- [examples/file-download](examples/file-download)

Run one:

```bash
cd examples/basic
go run .
```

## Design Principles

1. **Small trusted core**
2. **Standard-library first**
3. **Explicit over implicit**
4. **Performance with clarity**
5. **Composable middleware over built-in monoliths**

## Status

Kern is production-usable today for APIs and internal services, with an active roadmap focused on middleware depth, validation, and observability.

## Performance workflow

For the repeatable benchmark and pprof optimization process used in this repository, see [benchmarks/fourway/README.md](benchmarks/fourway/README.md).

## Roadmap tracking

Current implementation progress is tracked in [ROADMAP_STATUS.md](ROADMAP_STATUS.md).

### Hot handler query pattern

For handlers that repeatedly read a small fixed set of query fields, prefer the optional one-shot helpers to avoid repeated lookup and fallback branching in hot paths:

```go
q, page := c.QueryPairDefaultRaw("q", "", "page", "1")
_ = c.TextPair(http.StatusOK, q, "-", page)
```

Use `QueryPairDefault` instead when URL-decoding behavior is required.

## Contributing

Contributions are welcome. Please open an issue or pull request.

## License

MIT License. See [LICENSE](LICENSE).
