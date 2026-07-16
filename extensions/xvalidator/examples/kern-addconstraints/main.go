package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/mobentum/kern"
	"github.com/mobentum/kern/extensions/xvalidator"
)

type CreateUserRequest struct {
	Name  string `json:"name"  validate:"required,min=3,max=50"`
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age"   validate:"gte=18,lte=120"`
}

func main() {
	app := kern.New()

	// Pattern 1: In-handler validation (simple, explicit)
	app.POST("/users-v1", func(c *kern.Context) {
		var req CreateUserRequest
		if err := c.DecodeJSON(&req); err != nil {
			_ = c.Error(http.StatusBadRequest, "invalid request body")
			return
		}
		if err := xvalidator.Validate(req); err != nil {
			_ = c.JSON(http.StatusUnprocessableEntity, map[string]any{
				"error":  "validation failed",
				"fields": err,
			})
			return
		}
		_ = c.JSON(http.StatusCreated, map[string]string{
			"message": fmt.Sprintf("user %s created", req.Name),
		})
	})

	// Pattern 2: AddConstraints + xvalidator middleware factory
	app.AddConstraints(http.MethodPost, "/users-v2", kern.Constraints{
		Validate: xvalidator.BodyValidator[CreateUserRequest](),
	}, func(c *kern.Context) {
		req, ok := xvalidator.Validated[CreateUserRequest](c.Context())
		if !ok {
			c.NoContent(http.StatusInternalServerError)
			return
		}
		_ = c.JSON(http.StatusCreated, map[string]string{
			"message": fmt.Sprintf("user %s created", req.Name),
		})
	})

	log.Fatal(app.Run(":8080"))
}
