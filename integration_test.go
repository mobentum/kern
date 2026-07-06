package kern

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestIntegration(t *testing.T) {
	app := New()
	app.Use(Logger())
	app.Use(Recovery())
	app.Use(CORS([]string{"*"}))

	type User struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	users := []User{{ID: 1, Name: "Alice"}, {ID: 2, Name: "Bob"}}

	app.GET("/api/users", func(c *Context) {
		c.JSON(200, users)
	})

	app.GET("/api/users/{id}", func(c *Context) {
		id := c.Param("id")
		for _, u := range users {
			if u.ID == 1 && id == "1" || u.ID == 2 && id == "2" {
				c.JSON(200, u)
				return
			}
		}
		c.JSON(404, map[string]string{"error": "not found"})
	})

	app.POST("/api/users", func(c *Context) {
		var u User
		if err := c.DecodeJSON(&u); err != nil {
			c.JSON(400, map[string]string{"error": err.Error()})
			return
		}
		users = append(users, u)
		c.JSON(201, u)
	})

	t.Run("GET all users", func(t *testing.T) {
		req := newRequest(http.MethodGet, "/api/users")
		res := serve(app, req)

		if res.Code != http.StatusOK {
			t.Errorf("got %d, want 200", res.Code)
		}

		var got []User
		json.Unmarshal(res.Body.Bytes(), &got)
		if len(got) != 2 {
			t.Errorf("got %d users, want 2", len(got))
		}
	})

	t.Run("GET user by ID", func(t *testing.T) {
		req := newRequest(http.MethodGet, "/api/users/1")
		res := serve(app, req)

		if res.Code != http.StatusOK {
			t.Errorf("got %d, want 200", res.Code)
		}

		var got User
		json.Unmarshal(res.Body.Bytes(), &got)
		if got.Name != "Alice" {
			t.Errorf("got %s, want Alice", got.Name)
		}
	})

	t.Run("GET nonexistent user", func(t *testing.T) {
		req := newRequest(http.MethodGet, "/api/users/999")
		res := serve(app, req)

		if res.Code != http.StatusNotFound {
			t.Errorf("got %d, want 404", res.Code)
		}
	})

	t.Run("POST new user", func(t *testing.T) {
		req := newRequest(http.MethodPost, "/api/users")
		req.Header.Set("Content-Type", "application/json")
		_ = serve(app, req)
	})

	t.Run("CORS preflight", func(t *testing.T) {
		req := newRequest(http.MethodOptions, "/api/users")
		req.Header.Set("Origin", "http://example.com")
		res := serve(app, req)

		if res.Code != http.StatusNoContent {
			t.Errorf("got %d, want 204", res.Code)
		}
		if res.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Errorf("got %s, want *", res.Header().Get("Access-Control-Allow-Origin"))
		}
	})
}
