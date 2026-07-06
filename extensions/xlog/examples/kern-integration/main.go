package main

import (
	"log"
	"log/slog"
	"net/http"

	"github.com/mobentum/kern"
	"github.com/mobentum/kern/extensions/xlog"
)

func main() {
	lifecycle := xlog.NewLogger(xlog.Config{Format: "json", Level: slog.LevelInfo})
	request := xlog.NewLogger(xlog.Config{Format: "console", Level: slog.LevelInfo})

	app := kern.New(
		kern.WithSlogLogger(lifecycle),
	)

	app.Use(kern.Logger(kern.LoggerConfig{
		SLogger: request,
		Fields: map[string]interface{}{
			"service": "xlog-example",
			"env":     "local",
		},
	}))
	app.Use(kern.Recovery())

	app.GET("/", func(c *kern.Context) {
		_ = c.JSON(http.StatusOK, map[string]string{"message": "xlog + kern example"})
	})

	app.GET("/users/{id}", func(c *kern.Context) {
		_ = c.JSON(http.StatusOK, map[string]string{
			"id":      c.Param("id"),
			"request": c.GetHeader("X-Request-ID"),
		})
	})

	log.Fatal(app.Run(":8080"))
}
