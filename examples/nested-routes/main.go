package main

import (
	"log"

	"github.com/mobentum/kern"
)

func main() {
	app := kern.Default()

	// home
	app.GET("/", func(c *kern.Context) {
		_ = c.JSON(200, map[string]string{
			"message": "API with nested route groups",
			"docs":    "/api/v1/docs",
		})
	})

	// API v1
	v1 := app.Group("/api/v1", kern.CORS([]string{"*"}))
	{
		v1.GET("/docs", func(c *kern.Context) {
			_ = c.JSON(200, map[string]interface{}{
				"version": "1.0.0",
				"endpoints": map[string][]string{
					"public": {"/api/v1/health", "/api/v1/docs"},
					"users":  {"/api/v1/users", "/api/v1/users/{id}"},
					"posts":  {"/api/v1/posts", "/api/v1/posts/{id}"},
				},
			})
		})

		v1.GET("/health", func(c *kern.Context) {
			_ = c.JSON(200, map[string]string{"status": "healthy"})
		})

		// users nested group
		users := v1.Group("/users")
		{
			users.GET("", listUsers)
			users.GET("/{id}", getUser)
			users.POST("", createUser)

			// user posts nested group
			userPosts := users.Group("/{id}/posts")
			{
				userPosts.GET("", getUserPosts)
				userPosts.POST("", createUserPost)
			}
		}

		// posts nested group
		posts := v1.Group("/posts")
		{
			posts.GET("", listPosts)
			posts.GET("/{id}", getPost)
		}

	}

	log.Println("Server at http://localhost:8000")
	log.Println("Try:")
	log.Println("  curl http://localhost:8000/api/v1/docs")
	log.Println("  curl http://localhost:8000/api/v1/users")
	log.Println("  curl http://localhost:8000/api/v1/users/1/posts")

	log.Fatal(app.Run("localhost:8000"))
}

func listUsers(c *kern.Context) {
	_ = c.JSON(200, []map[string]interface{}{
		{"id": 1, "name": "Alice", "email": "alice@example.com"},
		{"id": 2, "name": "Bob", "email": "bob@example.com"},
	})
}

func getUser(c *kern.Context) {
	id := c.Param("id")
	_ = c.JSON(200, map[string]interface{}{
		"id":    id,
		"name":  "User " + id,
		"email": "user" + id + "@example.com",
	})
}

func createUser(c *kern.Context) {
	var user map[string]string
	if err := c.DecodeJSON(&user); err != nil {
		_ = c.JSON(400, map[string]string{"error": "Invalid JSON"})
		return
	}
	_ = c.JSON(201, user)
}

func getUserPosts(c *kern.Context) {
	userID := c.Param("id")
	_ = c.JSON(200, []map[string]interface{}{
		{"id": 1, "userId": userID, "title": "First Post"},
		{"id": 2, "userId": userID, "title": "Second Post"},
	})
}

func createUserPost(c *kern.Context) {
	userID := c.Param("id")
	var post map[string]string
	if err := c.DecodeJSON(&post); err != nil {
		_ = c.JSON(400, map[string]string{"error": "Invalid JSON"})
		return
	}
	post["user_id"] = userID
	_ = c.JSON(201, post)
}

func listPosts(c *kern.Context) {
	_ = c.JSON(200, []map[string]interface{}{
		{"id": 1, "title": "Getting Started with Go", "author": "Alice"},
		{"id": 2, "title": "Building Web APIs", "author": "Bob"},
	})
}

func getPost(c *kern.Context) {
	id := c.Param("id")
	_ = c.JSON(200, map[string]interface{}{
		"id":      id,
		"title":   "Post " + id,
		"content": "Lorem ipsum dolor sit amet...",
	})
}

func getStats(c *kern.Context) {
	_ = c.JSON(200, map[string]interface{}{
		"totalUsers":    42,
		"totalPosts":    128,
		"activeUsers":   38,
		"totalRequests": 1337,
	})
}

func adminListUsers(c *kern.Context) {
	_ = c.JSON(200, []map[string]interface{}{
		{"id": 1, "name": "Alice", "role": "admin"},
		{"id": 2, "name": "Bob", "role": "user"},
		{"id": 3, "name": "Charlie", "role": "user"},
	})
}
