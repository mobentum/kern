# xlog - zerolog-backed slog for kern

`xlog` is an optional logger module for kern that provides a `slog`-compatible interface backed by zerolog.

- Standalone module: `github.com/mobentum/kern/extensions/xlog`
- Works with kern app lifecycle logs via `kern.WithSlogLogger`
- Works with request logs via `kern.LoggerConfig{SLogger: ...}`

## Install

```bash
go get github.com/mobentum/kern/extensions/xlog
```

## Quick Start

```go
package main

import (
    "github.com/mobentum/kern"
    "github.com/mobentum/kern/extensions/xlog"
)

func main() {
    app := kern.New(
        kern.WithSlogLogger(xlog.NewLogger(xlog.Config{Format: "json"})),
    )

    app.Use(kern.Logger(kern.LoggerConfig{
        SLogger: xlog.NewLogger(xlog.Config{Format: "console"}),
        Fields: map[string]interface{}{
            "service": "users-api",
            "env":     "dev",
        },
    }))

    app.GET("/health", func(c *kern.Context) {
        _ = c.Text(200, "ok")
    })

    _ = app.Run(":8080")
}
```

## Runnable Example

This repository includes a complete kern + xlog sample app in:

- `extensions/xlog/examples/kern-integration`

Run it locally:

```bash
cd extensions/xlog/examples/kern-integration
go run .
```

Then call:

```bash
curl -i http://localhost:8080/
curl -i -H "X-Request-ID: req-123" http://localhost:8080/users/42
```

## Configuration

```go
type Config struct {
    Level      slog.Level
    Format     string   // "json" (default) or "console"
    Output     io.Writer
    TimeFormat string
}
```

Defaults:
- `Level`: `slog.LevelInfo`
- `Format`: `json`
- `Output`: `os.Stdout`
- `TimeFormat`: `time.RFC3339Nano`

## App Lifecycle Logging

`kern.WithSlogLogger(...)` controls app lifecycle events such as server start, listen errors, and shutdown events.

```go
app := kern.New(
    kern.WithSlogLogger(xlog.NewLogger(xlog.Config{Format: "json"})),
)
```

## Request Logging Middleware

Use `SLogger` to route request logs through slog/xlog.

```go
reqLogger := xlog.NewLogger(xlog.Config{
    Format: "json",
    Level:  slog.LevelInfo,
})

app.Use(kern.Logger(kern.LoggerConfig{
    SLogger: reqLogger,
    Fields: map[string]interface{}{
        "service": "billing-api",
        "region":  "us-east-1",
    },
}))
```

Expected fields include:
- `method`
- `path`
- `remote_addr`
- `status`
- `duration_ms`
- `size`
- `request_id` (when `X-Request-ID` exists)
- static fields from `Fields`

## JSON vs Console

JSON output:

```go
logger := xlog.NewLogger(xlog.Config{Format: "json"})
```

Console output:

```go
logger := xlog.NewLogger(xlog.Config{Format: "console"})
```

## Building a slog Logger Manually

If you need direct handler composition:

```go
handler := xlog.NewHandler(xlog.Config{Format: "json"})
logger := slog.New(handler)
```

## Notes

- `xlog` is separate from `kern` core to keep the core dependency-free.
- Import `xlog` only when you want zerolog-backed structured logging.
