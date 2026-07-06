package main

import (
	"log"
	"strconv"
	"sync"

	"github.com/mobentum/kern"
)

// data store to hold all todos
type Store struct {
	todos   map[int]*Todo
	counter int
	mu      sync.RWMutex
}

type Todo struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

// setup data store
var store = &Store{todos: make(map[int]*Todo)}

func main() {
	app := kern.Default()

	// cors setup
	app.Use(kern.CORS([]string{"*"}))

	api := app.Group("/api/v1")
	{
		api.GET("/todos", getTodos)
		api.POST("/todos", createTodo)
		api.GET("/todos/{id}", getTodo)
		api.PUT("/todos/{id}", updateTodo)
		api.DELETE("/todos/{id}", deleteTodo)
	}

	log.Fatal(app.Run("localhost:8000"))
}

func getTodos(c *kern.Context) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	list := make([]*Todo, 0, len(store.todos))
	for _, todo := range store.todos {
		list = append(list, todo)
	}

	_ = c.JSON(200, list)
}

func createTodo(c *kern.Context) {
	var todo Todo
	if err := c.DecodeJSON(&todo); err != nil {
		_ = c.JSON(400, map[string]string{"error": "Invalid JSON"})
		return
	}

	if todo.Title == "" {
		_ = c.JSON(400, map[string]string{"error": "Title is required"})
		return
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	todo.ID = store.counter
	store.counter++
	store.todos[todo.ID] = &todo

	_ = c.JSON(201, todo)
}

func getTodo(c *kern.Context) {
	id := parseID(c.Param("id"))

	store.mu.RLock()
	defer store.mu.Unlock()

	todo, exists := store.todos[id]
	if !exists {
		_ = c.JSON(404, map[string]string{"error": "Todo not found"})
		return
	}

	_ = c.JSON(200, todo)
}

func updateTodo(c *kern.Context) {
	id := parseID(c.Param("id"))

	var updated Todo
	if err := c.DecodeJSON(&updated); err != nil {
		_ = c.JSON(400, map[string]string{"error": "Invalid JSON"})
		return
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	todo, exists := store.todos[id]
	if !exists {
		_ = c.JSON(404, map[string]string{"error": "Todo not found"})
		return
	}

	if updated.Title != "" {
		todo.Title = updated.Title
	}
	todo.Completed = updated.Completed

	_ = c.JSON(200, todo)
}

func deleteTodo(c *kern.Context) {
	id := parseID(c.Param("id"))

	store.mu.Lock()
	defer store.mu.Unlock()

	if _, exists := store.todos[id]; !exists {
		_ = c.JSON(404, map[string]string{"error": "Todo not found"})
		return
	}

	delete(store.todos, id)
	_ = c.JSON(200, map[string]string{"message": "Todo deleted successfully"})
}

func parseID(s string) int {
	id, _ := strconv.Atoi(s)
	return id
}
