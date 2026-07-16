package kern

import (
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
