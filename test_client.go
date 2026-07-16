package kern

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
)

// TestClient provides concise, reusable request helpers for app tests.
type TestClient struct {
	app            *App
	defaultHeaders map[string]string
}

// NewTestClient creates a test client for the given app.
func NewTestClient(app *App) *TestClient {
	return &TestClient{app: app, defaultHeaders: map[string]string{}}
}

// WithHeader sets a default header on all subsequent requests.
func (tc *TestClient) WithHeader(key, value string) *TestClient {
	tc.defaultHeaders[key] = value
	return tc
}

// Do sends a prepared request and returns the recorded response.
func (tc *TestClient) Do(req *http.Request) *httptest.ResponseRecorder {
	for key, value := range tc.defaultHeaders {
		if req.Header.Get(key) == "" {
			req.Header.Set(key, value)
		}
	}

	res := httptest.NewRecorder()
	tc.app.ServeHTTP(res, req)
	return res
}

// Request creates a request with the given method, path, and body, then sends it.
func (tc *TestClient) Request(method, path string, body io.Reader) *httptest.ResponseRecorder {
	return tc.Do(httptest.NewRequest(method, path, body))
}

// Get sends a GET request to the given path.
func (tc *TestClient) Get(path string) *httptest.ResponseRecorder {
	return tc.Request(http.MethodGet, path, nil)
}

// PostJSON sends a POST request with a JSON-encoded body.
func (tc *TestClient) PostJSON(path string, payload interface{}) *httptest.ResponseRecorder {
	body, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}

	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	return tc.Do(req)
}

// Post sends a POST request with an optional body.
func (tc *TestClient) Post(path string, body io.Reader) *httptest.ResponseRecorder {
	return tc.Request(http.MethodPost, path, body)
}

// Put sends a PUT request with an optional body.
func (tc *TestClient) Put(path string, body io.Reader) *httptest.ResponseRecorder {
	return tc.Request(http.MethodPut, path, body)
}

// Delete sends a DELETE request.
func (tc *TestClient) Delete(path string) *httptest.ResponseRecorder {
	return tc.Request(http.MethodDelete, path, nil)
}
