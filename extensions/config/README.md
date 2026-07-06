# config - dotenv and environment loading for kern

`config` is an optional configuration module for kern for dotenv-style loading and environment access.

- Standalone module: `github.com/mobentum/kern/extensions/config`
- Loads `.env` files into the process environment
- Reads common values through a small loader API
- Fits best when your application owns its own `Config` struct
- Keeps `kern` core dependency-free

## Install

```bash
go get github.com/mobentum/kern/extensions/config
```

## Quick Start

```go
package main

import (
    "log"
    "time"

    "github.com/mobentum/kern/extensions/config"
)

type Config struct {
    Host    string
    Port    int
    Debug   bool
    Timeout time.Duration
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
    debug, err := loader.Bool("DEBUG", false)
    if err != nil {
        return nil, err
    }
    timeout, err := loader.Duration("TIMEOUT", 5*time.Second)
    if err != nil {
        return nil, err
    }

    return &Config{
        Host:    loader.String("HOST", "127.0.0.1"),
        Port:    port,
        Debug:   debug,
        Timeout: timeout,
    }, nil
}

func main() {
    cfg, err := LoadConfig()
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("host=%s port=%d debug=%t timeout=%s", cfg.Host, cfg.Port, cfg.Debug, cfg.Timeout)
}
```

## Loading `.env` Files

```go
err := config.Load()
err := config.Load(".env.local", ".env")
err := config.Overload(".env.test")
config.MustLoad(".env")
```

- `Load` preserves environment variables that already exist.
- `Overload` replaces existing environment variables.
- `MustLoad` panics on failure.

## Creating a Loader

```go
loader, err := config.New(
    config.WithPrefix("APP"),
    config.WithDotEnv(".env"),
)
if err != nil {
    return err
}
```

Available options:
- `WithPrefix("APP")` reads keys like `APP_HOST`, `APP_PORT`
- `WithDotEnv(".env")` loads dotenv files during loader construction and returns errors through `New`
- `WithLookup(fn)` replaces `os.LookupEnv`, useful in tests

## Reading Typed Values

```go
host := loader.String("HOST", "127.0.0.1")
port, err := loader.Int("PORT", 8080)
debug, err := loader.Bool("DEBUG", false)
timeout, err := loader.Duration("TIMEOUT", 5*time.Second)
origins := loader.Strings("ALLOWED_ORIGINS", []string{"*"})
```

Supported getters:
- `LookupString`
- `String`
- `RequiredString`
- `Bool`
- `Int`
- `Duration`
- `Strings`

The typed getters with defaults return the default when a key is absent, but still return an error when the value exists and cannot be parsed.

## Recommended Pattern

Keep app-level configuration in your own struct and use `config` only to read environment values explicitly:

```go
type Config struct {
    DatabaseURL string
    GRPCPort    string
}

func LoadConfig() (*Config, error) {
    loader, err := config.New(config.WithDotEnv())
    if err != nil {
        return nil, err
    }

    return &Config{
        DatabaseURL: loader.String("DATABASE_URL", "postgres://localhost/app"),
        GRPCPort:    loader.String("GRPC_PORT", "9090"),
    }, nil
}
```

## Errors

Use `errors.Is` to distinguish missing vs invalid values:

```go
port, err := loader.Int("PORT", 8080)
if err != nil {
    switch {
    case errors.Is(err, config.ErrInvalidValue):
        return err
    case errors.Is(err, config.ErrMissingRequired):
        return err
    }
}
```

## Runnable Example

A complete kern integration example lives in:

- `extensions/config/examples/kern-integration`

Run it locally:

```bash
cd extensions/config/examples/kern-integration
go run .
```
