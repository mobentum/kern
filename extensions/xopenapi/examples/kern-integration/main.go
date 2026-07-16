package main

import (
	"net/http"

	"github.com/mobentum/kern"
	"github.com/mobentum/kern/extensions/xopenapi"
)

func main() {
	app := kern.New()

	app.GET("/health", func(c *kern.Context) {
		_ = c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})
	app.GET("/users/{id}", func(c *kern.Context) {
		_ = c.JSON(http.StatusOK, map[string]string{"id": c.Param("id")})
	})

	xopenapi.Register(app, xopenapi.Config{
		Info: xopenapi.Info{
			Title:       "Kern OpenAPI Demo",
			Version:     "1.0.0",
			Description: "Example API docs served by extensions/xopenapi",
		},
		Servers: []xopenapi.Server{{URL: "http://localhost:8080", Description: "local"}},
		Routes: []xopenapi.Route{
			{
				Method:      http.MethodGet,
				Path:        "/health",
				Summary:     "Health check",
				OperationID: "healthCheck",
				Tags:        []string{"system"},
				Responses: map[string]xopenapi.Response{
					"200": {
						Description: "OK",
						Content: map[string]xopenapi.MediaType{
							"application/json": {
								Schema: xopenapi.Schema{
									"type": "object",
									"properties": map[string]interface{}{
										"status": map[string]interface{}{"type": "string"},
									},
								},
							},
						},
					},
				},
			},
			{
				Method:      http.MethodGet,
				Path:        "/users/{id}",
				Summary:     "Get user",
				OperationID: "getUser",
				Tags:        []string{"users"},
				Parameters: []xopenapi.Parameter{
					{
						Name:        "id",
						In:          "path",
						Description: "User identifier",
						Schema:      xopenapi.Schema{"type": "string"},
					},
					{
						Name:        "include",
						In:          "query",
						Description: "Optional expansion",
						Schema:      xopenapi.Schema{"type": "string"},
						Example:     "profile",
					},
				},
				Responses: map[string]xopenapi.Response{
					"200": {
						Description: "User",
						Content: map[string]xopenapi.MediaType{
							"application/json": {
								Schema: xopenapi.Schema{
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
