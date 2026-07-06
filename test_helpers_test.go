package kern

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
)

func newRequest(method, path string) *http.Request {
	return httptest.NewRequest(method, path, nil)
}

func newRequestWithBody(method, path string, body io.Reader) *http.Request {
	return httptest.NewRequest(method, path, body)
}

func serve(app *App, req *http.Request) *httptest.ResponseRecorder {
	res := httptest.NewRecorder()
	app.ServeHTTP(res, req)
	return res
}

// TestClient provides concise, reusable request helpers for app tests.
type TestClient struct {
	app            *App
	defaultHeaders map[string]string
}

func newTestClient(app *App) *TestClient {
	return &TestClient{app: app, defaultHeaders: map[string]string{}}
}

func (tc *TestClient) WithHeader(key, value string) *TestClient {
	tc.defaultHeaders[key] = value
	return tc
}

func (tc *TestClient) Do(req *http.Request) *httptest.ResponseRecorder {
	for key, value := range tc.defaultHeaders {
		if req.Header.Get(key) == "" {
			req.Header.Set(key, value)
		}
	}

	return serve(tc.app, req)
}

func (tc *TestClient) Request(method, path string, body io.Reader) *httptest.ResponseRecorder {
	return tc.Do(newRequestWithBody(method, path, body))
}

func (tc *TestClient) Get(path string) *httptest.ResponseRecorder {
	return tc.Request(http.MethodGet, path, nil)
}

func (tc *TestClient) PostJSON(path string, payload interface{}) *httptest.ResponseRecorder {
	body, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}

	req := newRequestWithBody(http.MethodPost, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	return tc.Do(req)
}
