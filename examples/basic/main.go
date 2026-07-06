package main

import (
	"log"

	"github.com/mobentum/kern"
)

func main() {
	// create default instance
	app := kern.Default()

	// plain text response
	app.GET("/", func(c *kern.Context) {
		c.Text(200, "Welcome to the basic tutorial")
	})

	// json response
	app.GET("/hello/{name}", func(c *kern.Context) {
		name := c.Param("name")

		c.JSON(200, map[string]any{
			"message": "Hello " + name,
		})
	})

	// run app
	log.Fatal(app.Run("localhost:8000"))
}
