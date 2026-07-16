package kern

import (
	"net/http"
	"testing"
)

func TestTestClient_GetAndDefaultHeaders(t *testing.T) {
	app := New()
	app.GET("/whoami", func(c *Context) {
		_ = c.Text(http.StatusOK, "%s", c.GetHeader("X-User"))
	})

	client := NewTestClient(app).WithHeader("X-User", "alice")
	res := client.Get("/whoami")

	if res.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", res.Code)
	}
	if got := res.Body.String(); got != "alice" {
		t.Fatalf("got %q, want %q", got, "alice")
	}
}

func TestTestClient_PostJSON(t *testing.T) {
	type input struct {
		Name string `json:"name"`
	}

	app := New()
	app.POST("/users", func(c *Context) {
		var in input
		if err := c.DecodeJSON(&in); err != nil {
			_ = c.Error(http.StatusBadRequest, "invalid json")
			return
		}
		_ = c.Text(http.StatusCreated, "%s", in.Name)
	})

	client := NewTestClient(app)
	res := client.PostJSON("/users", input{Name: "mobentum"})

	if res.Code != http.StatusCreated {
		t.Fatalf("got %d, want 201", res.Code)
	}
	if got := res.Body.String(); got != "mobentum" {
		t.Fatalf("got %q, want %q", got, "mobentum")
	}
}
