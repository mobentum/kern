package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/mobentum/kern"
)

func main() {
	app := kern.New()

	// global middleware
	app.Use(kern.Logger())
	app.Use(kern.Recovery())

	// custom middleware
	app.Use(APIVersionMiddleware("1.0.0"))

	// public routes
	app.GET("/", func(c *kern.Context) {
		c.JSON(200, map[string]string{
			"message": "Welcome to the API",
		})
	})

	// API routes with CORS
	api := app.Group("/api", kern.CORS([]string{"*"}))
	{
		api.GET("/health", func(c *kern.Context) {
			c.JSON(200, map[string]string{"status": "healthy"})
		})

		api.GET("/products", func(c *kern.Context) {
			c.JSON(200, []map[string]interface{}{
				{"id": 1, "name": "Product 1"},
				{"id": 2, "name": "Product 2"},
			})
		})
	}

	log.Println("Server running at http://localhost:8000")
	log.Println("Try:")
	log.Println("  curl -i http://localhost:8000/")
	log.Println("  curl -i http://localhost:8000/api/health")

	log.Fatal(app.Run("localhost:8000"))
}

// APIVersionMiddleware adds an API version header to all responses
func APIVersionMiddleware(version string) kern.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-API-Version", version)

			// call next handler
			next.ServeHTTP(w, r)
		})
	}
}

// Optional: Simple logging middleware example
func SimpleLogger() kern.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("→ %s %s", r.Method, r.URL.Path)

			next.ServeHTTP(w, r)

			log.Printf("← %s %s completed", r.Method, r.URL.Path)
		})
	}
}

// Optional: Content-Type enforcement middleware
func RequireJSON() kern.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only check POST, PUT, PATCH requests
			if r.Method == "POST" || r.Method == "PUT" || r.Method == "PATCH" {
				contentType := r.Header.Get("Content-Type")
				if !strings.Contains(contentType, "application/json") {
					http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
