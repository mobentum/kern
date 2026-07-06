# openapi - simple OpenAPI endpoints for kern

`openapi` is an optional module that serves an OpenAPI JSON document and Swagger UI from a kern app.

- Standalone module: `github.com/mobentum/kern/extensions/openapi`
- No reflection or auto-schema generation
- Explicit route metadata (you own your API contract)

## Install

```bash
go get github.com/mobentum/kern/extensions/openapi
```

## Quick Start

```go
package main

import (
    "net/http"

    "github.com/mobentum/kern"
    "github.com/mobentum/kern/extensions/openapi"
)

func main() {
    app := kern.New()

    app.GET("/users/{id}", func(c *kern.Context) {
        _ = c.JSON(http.StatusOK, map[string]string{"id": c.Param("id")})
    })

    openapi.Register(app, openapi.Config{
        Info: openapi.Info{
            Title:       "Users API",
            Version:     "1.0.0",
            Description: "Public API for user operations",
        },
        Routes: []openapi.Route{
            {
                Method:      http.MethodGet,
                Path:        "/users/{id}",
                Summary:     "Get user",
                OperationID: "getUser",
                Tags:        []string{"users"},
                Responses: map[string]openapi.Response{
                    "200": {
                        Description: "User",
                        Content: map[string]openapi.MediaType{
                            "application/json": {
                                Schema: openapi.Schema{
                                    "type": "object",
                                    "properties": map[string]interface{}{
                                        "id": map[string]interface{}{"type": "string"},
                                    },
                                },
                            },
                        },
                    },
                },
            },
        },
    })

    _ = app.Run(":8080")
}
```

Then open:
- `http://localhost:8080/openapi.json`
- `http://localhost:8080/docs`

## Config

```go
type Config struct {
    JSONPath  string // default: /openapi.json
    DocsPath  string // default: /docs
    DocsTitle string // default: <Info.Title> - API Docs
    Info      Info
    Servers   []Server
    Routes    []Route
}
```

Optional schema fields on each route:

```go
type Route struct {
    Method      string
    Path        string
    Summary     string
    Description string
    OperationID string
    Tags        []string
    Parameters  []Parameter
    RequestBody *RequestBody
    Responses   map[string]Response
}

type Parameter struct {
    Name        string
    In          string // path, query, header, cookie
    Description string
    Required    bool
    Schema      Schema
    Example     interface{}
}

type RequestBody struct {
    Description string
    Required    bool
    Content     map[string]MediaType // e.g. "application/json"
}

type Response struct {
    Description string
    Content     map[string]MediaType
}

type MediaType struct {
    Schema Schema
}

type Schema map[string]interface{}
```

Path parameters are always marked `required=true` in generated output.

The module intentionally keeps request/response schemas manual. You can add schemas later by extending this module or generating JSON separately.

## Runnable Example

A complete kern integration example lives in:

- `extensions/openapi/examples/kern-integration`

Run it locally:

```bash
cd extensions/openapi/examples/kern-integration
go run .
```
