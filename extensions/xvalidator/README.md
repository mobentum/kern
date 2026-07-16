# xvalidator - struct validation for kern

`xvalidator` wraps [go-playground/validator](https://github.com/go-playground/validator) and provides convenient struct validation for Kern applications.

- Standalone module: `github.com/mobentum/kern/extensions/xvalidator`
- Built-in validation tags (`required`, `min`, `max`, `email`, `uuid4`, `oneof`, etc.)
- Custom validation rules
- Custom error messages with per-tag defaults
- Struct-level validation (cross-field checks)
- Nested struct and slice validation via `dive`
- Works with `c.DecodeJSON()` / `c.Bind()` — validate the decoded struct in a separate step
- `BodyValidator[T]()` middleware — decode + validate in one step via `AddConstraints`
- `Validated[T]()` — retrieve the validated struct from the request context

## Install

```bash
go get github.com/mobentum/kern/extensions/xvalidator
```

## Quick Start

```go
package main

import (
    "net/http"

    "github.com/mobentum/kern"
    "github.com/mobentum/kern/extensions/xvalidator"
)

type createUserRequest struct {
    Name     string `json:"name"     validate:"required,min=3,max=50"`
    Email    string `json:"email"    validate:"required,email"`
    Age      int    `json:"age"      validate:"gte=18,lte=120"`
    Password string `json:"password" validate:"required,min=8"`
}

func main() {
    app := kern.New()

    app.POST("/users", func(c *kern.Context) {
        var req createUserRequest
        if err := c.DecodeJSON(&req); err != nil {
            _ = c.Error(http.StatusBadRequest, "invalid request body")
            return
        }

        if err := xvalidator.Validate(req); err != nil {
            _ = c.JSON(http.StatusUnprocessableEntity, map[string]interface{}{
                "error":  "validation failed",
                "fields": err,
            })
            return
        }

        _ = c.JSON(http.StatusCreated, map[string]string{"status": "ok"})
    })

    _ = app.Run(":8080")
}
```

## Custom Messages

```go
xvalidator.Default().SetMessages(map[string]string{
    "required": "is required",
    "email":    "must be a valid email address",
    "min":      "must be at least %s characters",
})

// Or per-call:
err := xvalidator.ValidateWithMessages(req, map[string]string{
    "email": "provide a valid email address",
})
```

## Custom Validation Rules

```go
import "github.com/go-playground/validator/v10"

xvalidator.RegisterValidation("even", func(fl validator.FieldLevel) bool {
    n, ok := fl.Field().Interface().(int)
    if !ok { return false }
    return n%2 == 0
})

type request struct {
    Value int `validate:"even"`
}
```

## Struct-Level Validation (cross-field)

```go
v := xvalidator.New()

type passwordRequest struct {
    Password        string `validate:"required,min=8"`
    ConfirmPassword string `validate:"required"`
}

v.RegisterStructValidation(func(sl validator.StructLevel) {
    r := sl.Current().Interface().(passwordRequest)
    if r.Password != r.ConfirmPassword {
        sl.ReportError(r.ConfirmPassword, "ConfirmPassword", "ConfirmPassword", "eqfield", "Password")
    }
}, passwordRequest{})

err := v.Validate(req)
```

## Nested Validation

```go
type orderItem struct {
    ProductID string  `validate:"required,uuid4"`
    Quantity  int     `validate:"required,min=1,max=100"`
    Price     float64 `validate:"required,gt=0"`
}

type orderRequest struct {
    Items []orderItem `validate:"required,min=1,dive"`
}
```

## Middleware

`xvalidator.BodyValidator[T]()` decodes and validates the request body in a single middleware step. The validated struct is stored in the request context and can be retrieved with `xvalidator.Validated[T]()`.

```go
type createUserRequest struct {
    Name  string `json:"name"  validate:"required,min=3,max=50"`
    Email string `json:"email" validate:"required,email"`
    Age   int    `json:"age"   validate:"gte=18,lte=120"`
}

app.AddConstraints(http.MethodPost, "/users", kern.Constraints{
    Validate: xvalidator.BodyValidator[createUserRequest](),
}, func(c *kern.Context) {
    req, ok := xvalidator.Validated[createUserRequest](c.Context())
    if !ok {
        c.NoContent(http.StatusInternalServerError)
        return
    }
    _ = c.JSON(http.StatusCreated, map[string]string{"name": req.Name})
})
```

## Runnable Examples

- `extensions/xvalidator/examples/kern-integration` — in-handler validation
- `extensions/xvalidator/examples/kern-addconstraints` — middleware validation with `AddConstraints`

Run them locally:

```bash
cd extensions/xvalidator/examples/<name>
go run .
```
