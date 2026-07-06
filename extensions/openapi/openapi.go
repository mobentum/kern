package openapi

import (
	"net/http"
	"strings"

	"github.com/mobentum/kern"
)

// Info describes high-level API metadata for the OpenAPI document.
type Info struct {
	Title       string
	Version     string
	Description string
}

// Server declares a server entry in the OpenAPI document.
type Server struct {
	URL         string
	Description string
}

// Schema is a manual JSON schema object.
// Keep this explicit and app-owned.
type Schema map[string]interface{}

// MediaType describes content for a single MIME type.
type MediaType struct {
	Schema Schema
}

// RequestBody describes the request body for an operation.
type RequestBody struct {
	Description string
	Required    bool
	Content     map[string]MediaType
}

// Response describes a single operation response.
type Response struct {
	Description string
	Content     map[string]MediaType
}

// Parameter describes a single OpenAPI operation parameter.
type Parameter struct {
	Name        string
	In          string // path, query, header, cookie
	Description string
	Required    bool
	Schema      Schema
	Example     interface{}
}

// Route describes one API operation. Keep this explicit and app-owned.
type Route struct {
	Method      string
	Path        string
	Summary     string
	Description string
	OperationID string
	Tags        []string
	Parameters  []Parameter
	RequestBody *RequestBody
	Responses   map[string]Response
}

// Config defines how docs endpoints are exposed and what is included in the document.
type Config struct {
	JSONPath  string
	DocsPath  string
	DocsTitle string
	Info      Info
	Servers   []Server
	Routes    []Route
}

// Register mounts an OpenAPI JSON endpoint and a Swagger UI endpoint.
//
// Defaults:
//   - JSONPath: /openapi.json
//   - DocsPath: /docs
//   - Info.Title: API
//   - Info.Version: 1.0.0
func Register(app *kern.App, cfg Config) {
	if app == nil {
		panic("openapi: app cannot be nil")
	}

	cfg = withDefaults(cfg)
	document := buildDocument(cfg)

	app.GET(cfg.JSONPath, func(c *kern.Context) {
		_ = c.JSON(http.StatusOK, document)
	})

	docsHTML := renderSwaggerUIHTML(cfg)
	app.GET(cfg.DocsPath, func(c *kern.Context) {
		_ = c.HTML(http.StatusOK, docsHTML)
	})
}

func withDefaults(cfg Config) Config {
	if cfg.JSONPath == "" {
		cfg.JSONPath = "/openapi.json"
	}
	if cfg.DocsPath == "" {
		cfg.DocsPath = "/docs"
	}
	if cfg.Info.Title == "" {
		cfg.Info.Title = "API"
	}
	if cfg.Info.Version == "" {
		cfg.Info.Version = "1.0.0"
	}
	if cfg.DocsTitle == "" {
		cfg.DocsTitle = cfg.Info.Title + " - API Docs"
	}
	return cfg
}

func buildDocument(cfg Config) map[string]interface{} {
	info := map[string]interface{}{
		"title":   cfg.Info.Title,
		"version": cfg.Info.Version,
	}
	if cfg.Info.Description != "" {
		info["description"] = cfg.Info.Description
	}

	servers := make([]map[string]interface{}, 0, len(cfg.Servers))
	for _, server := range cfg.Servers {
		item := map[string]interface{}{"url": server.URL}
		if server.Description != "" {
			item["description"] = server.Description
		}
		servers = append(servers, item)
	}

	paths := make(map[string]map[string]interface{})
	for _, route := range cfg.Routes {
		method := normalizeMethod(route.Method)
		if method == "" || route.Path == "" {
			continue
		}

		operations, ok := paths[route.Path]
		if !ok {
			operations = make(map[string]interface{})
			paths[route.Path] = operations
		}

		op := map[string]interface{}{}

		if len(route.Parameters) > 0 {
			params := buildParameters(route.Parameters)
			if len(params) > 0 {
				op["parameters"] = params
			}
		}

		if len(route.Responses) > 0 {
			op["responses"] = buildResponses(route.Responses)
		} else {
			op["responses"] = map[string]interface{}{
				"200": map[string]interface{}{"description": "OK"},
			}
		}

		if route.RequestBody != nil {
			reqBody := map[string]interface{}{}
			if route.RequestBody.Description != "" {
				reqBody["description"] = route.RequestBody.Description
			}
			if route.RequestBody.Required {
				reqBody["required"] = true
			}
			if len(route.RequestBody.Content) > 0 {
				reqBody["content"] = buildMediaTypes(route.RequestBody.Content)
			}
			op["requestBody"] = reqBody
		}
		if route.Summary != "" {
			op["summary"] = route.Summary
		}
		if route.Description != "" {
			op["description"] = route.Description
		}
		if route.OperationID != "" {
			op["operationId"] = route.OperationID
		}
		if len(route.Tags) > 0 {
			op["tags"] = route.Tags
		}

		operations[method] = op
	}

	doc := map[string]interface{}{
		"openapi": "3.0.3",
		"info":    info,
		"paths":   paths,
	}
	if len(servers) > 0 {
		doc["servers"] = servers
	}
	return doc
}

func normalizeMethod(method string) string {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case http.MethodGet:
		return "get"
	case http.MethodPost:
		return "post"
	case http.MethodPut:
		return "put"
	case http.MethodPatch:
		return "patch"
	case http.MethodDelete:
		return "delete"
	case http.MethodOptions:
		return "options"
	case http.MethodHead:
		return "head"
	default:
		return ""
	}
}

func buildResponses(responses map[string]Response) map[string]interface{} {
	out := make(map[string]interface{}, len(responses))
	for statusCode, response := range responses {
		item := map[string]interface{}{}
		description := strings.TrimSpace(response.Description)
		if description == "" {
			description = "Response"
		}
		item["description"] = description
		if len(response.Content) > 0 {
			item["content"] = buildMediaTypes(response.Content)
		}
		out[statusCode] = item
	}
	return out
}

func buildMediaTypes(content map[string]MediaType) map[string]interface{} {
	out := make(map[string]interface{}, len(content))
	for contentType, media := range content {
		item := map[string]interface{}{}
		if len(media.Schema) > 0 {
			item["schema"] = media.Schema
		}
		out[contentType] = item
	}
	return out
}

func buildParameters(params []Parameter) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(params))
	for _, parameter := range params {
		in := normalizeParameterIn(parameter.In)
		name := strings.TrimSpace(parameter.Name)
		if in == "" || name == "" {
			continue
		}

		item := map[string]interface{}{
			"name": name,
			"in":   in,
		}

		required := parameter.Required
		if in == "path" {
			required = true
		}
		if required {
			item["required"] = true
		}
		if parameter.Description != "" {
			item["description"] = parameter.Description
		}
		if len(parameter.Schema) > 0 {
			item["schema"] = parameter.Schema
		}
		if parameter.Example != nil {
			item["example"] = parameter.Example
		}

		out = append(out, item)
	}
	return out
}

func normalizeParameterIn(in string) string {
	switch strings.ToLower(strings.TrimSpace(in)) {
	case "path", "query", "header", "cookie":
		return strings.ToLower(strings.TrimSpace(in))
	default:
		return ""
	}
}

func renderSwaggerUIHTML(cfg Config) string {
	return "<!doctype html>\n" +
		"<html lang=\"en\">\n" +
		"<head>\n" +
		"  <meta charset=\"utf-8\" />\n" +
		"  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1\" />\n" +
		"  <title>" + escapeHTML(cfg.DocsTitle) + "</title>\n" +
		"  <link rel=\"stylesheet\" href=\"https://unpkg.com/swagger-ui-dist@5/swagger-ui.css\" />\n" +
		"  <style>body{margin:0;background:#fafafa}#swagger-ui{max-width:1200px;margin:0 auto}</style>\n" +
		"</head>\n" +
		"<body>\n" +
		"  <div id=\"swagger-ui\"></div>\n" +
		"  <script src=\"https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js\"></script>\n" +
		"  <script>window.ui=SwaggerUIBundle({url:'" + jsString(cfg.JSONPath) + "',dom_id:'#swagger-ui'});</script>\n" +
		"</body>\n" +
		"</html>\n"
}

func escapeHTML(s string) string {
	replacer := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;", "'", "&#39;")
	return replacer.Replace(s)
}

func jsString(s string) string {
	replacer := strings.NewReplacer("\\", "\\\\", "'", "\\'", "\n", "", "\r", "")
	return replacer.Replace(s)
}
