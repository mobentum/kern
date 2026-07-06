package openapi

import (
	"net/http"
	"testing"
)

func TestWithDefaults(t *testing.T) {
	cfg := withDefaults(Config{})
	if cfg.JSONPath != "/openapi.json" {
		t.Fatalf("unexpected JSONPath: %s", cfg.JSONPath)
	}
	if cfg.DocsPath != "/docs" {
		t.Fatalf("unexpected DocsPath: %s", cfg.DocsPath)
	}
	if cfg.Info.Title != "API" {
		t.Fatalf("unexpected title: %s", cfg.Info.Title)
	}
	if cfg.Info.Version != "1.0.0" {
		t.Fatalf("unexpected version: %s", cfg.Info.Version)
	}
}

func TestBuildDocument(t *testing.T) {
	doc := buildDocument(withDefaults(Config{
		Info:    Info{Title: "Users API", Version: "1.2.3"},
		Servers: []Server{{URL: "http://localhost:8080"}},
		Routes: []Route{
			{
				Method:      http.MethodGet,
				Path:        "/users/{id}",
				Summary:     "Get user",
				OperationID: "getUser",
				Tags:        []string{"users"},
				Parameters: []Parameter{
					{
						Name:        "id",
						In:          "path",
						Description: "User ID",
						Schema:      Schema{"type": "string"},
					},
					{
						Name:     "include",
						In:       "query",
						Required: false,
						Schema:   Schema{"type": "string"},
						Example:  "profile",
					},
					{
						Name: "bad",
						In:   "not-valid",
					},
				},
			},
			{
				Method:      http.MethodPost,
				Path:        "/users",
				Summary:     "Create user",
				OperationID: "createUser",
				Tags:        []string{"users"},
				RequestBody: &RequestBody{
					Description: "User payload",
					Required:    true,
					Content: map[string]MediaType{
						"application/json": {
							Schema: Schema{
								"type": "object",
								"properties": map[string]interface{}{
									"name": map[string]interface{}{"type": "string"},
								},
							},
						},
					},
				},
				Responses: map[string]Response{
					"201": {
						Description: "Created",
						Content: map[string]MediaType{
							"application/json": {
								Schema: Schema{
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
			{Method: "INVALID", Path: "/ignored"},
		},
	}))

	paths, ok := doc["paths"].(map[string]map[string]interface{})
	if !ok {
		t.Fatalf("paths shape mismatch: %#v", doc["paths"])
	}
	if _, ok := paths["/ignored"]; ok {
		t.Fatal("expected invalid method route to be ignored")
	}
	if _, ok := paths["/users/{id}"]["get"]; !ok {
		t.Fatal("expected GET operation for /users/{id}")
	}
	if _, ok := paths["/users"]["post"]; !ok {
		t.Fatal("expected POST operation for /users")
	}

	get := paths["/users/{id}"]["get"].(map[string]interface{})
	parameters := get["parameters"].([]map[string]interface{})
	if len(parameters) != 2 {
		t.Fatalf("expected 2 valid parameters, got %d", len(parameters))
	}
	if got := parameters[0]["in"]; got != "path" {
		t.Fatalf("expected first parameter in=path, got %v", got)
	}
	if got := parameters[0]["required"]; got != true {
		t.Fatalf("expected path parameter required=true, got %v", got)
	}
	if got := parameters[1]["in"]; got != "query" {
		t.Fatalf("expected second parameter in=query, got %v", got)
	}

	post := paths["/users"]["post"].(map[string]interface{})
	if _, ok := post["requestBody"]; !ok {
		t.Fatal("expected requestBody for POST /users")
	}
	responses := post["responses"].(map[string]interface{})
	if _, ok := responses["201"]; !ok {
		t.Fatal("expected 201 response for POST /users")
	}
}
