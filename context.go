package kern

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	ErrEmptyRequestBody  = errors.New("request body is empty")
	ErrInvalidBindTarget = errors.New("bind target must be a non-nil pointer to struct")
	bindPlanCache        sync.Map
	jsonBodyBufferPool   = sync.Pool{
		New: func() interface{} {
			buf := make([]byte, 0, 1024)
			return &buf
		},
	}
	jsonContentType = []string{"application/json"}
	xmlContentType  = []string{"application/xml"}
	textContentType = []string{"text/plain; charset=utf-8"}
	htmlContentType = []string{"text/html; charset=utf-8"}
)

const maxRetainedJSONBodyBuffer = 64 << 10

type bindPlanKey struct {
	typeOf  reflect.Type
	tagName string
}

type bindFieldPlan struct {
	index  int
	name   string
	binder bindFieldValueFunc
}

type bindPlan struct {
	fields []bindFieldPlan
}

type bindFieldValueFunc func(field reflect.Value, values []string) error

// FieldValidationError describes a single field validation failure.
type FieldValidationError struct {
	Field string
	Tag   string
	Param string
	Value interface{}
}

// ValidationErrors contains all field validation errors.
type ValidationErrors []FieldValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return "validation failed"
	}

	err := e[0]
	if err.Param != "" {
		return fmt.Sprintf("validation failed for %s: %s=%s", err.Field, err.Tag, err.Param)
	}

	return fmt.Sprintf("validation failed for %s: %s", err.Field, err.Tag)
}

type stringWriter interface {
	WriteString(string) (int, error)
}

// JSONErrorPayload is the default structured payload for JSONError responses.
type JSONErrorPayload struct {
	Error   string      `json:"error"`
	Details interface{} `json:"details,omitempty"`
}

// Context adds helpful methods to the ongoing request
type Context struct {
	Request  *http.Request
	Response http.ResponseWriter

	app *App

	// per-request slog logger derived from app configuration.
	slogger *slog.Logger

	// cached url data
	query      url.Values
	formParsed bool

	// tiny hot-path cache for repeated path param lookups.
	param1Name  string
	param1Value string
	param1Set   bool
	param2Name  string
	param2Value string
	param2Set   bool
}

// reset prepares a context to be reused by a new request
func (ctx *Context) reset(w http.ResponseWriter, r *http.Request) {
	ctx.Response = w
	ctx.Request = r
	ctx.slogger = ctx.app.slogger
	ctx.query = nil
	ctx.formParsed = false
	ctx.param1Set = false
	ctx.param2Set = false
}

// Logger returns the request logger configured for this context.
//
// It returns nil when no slog logger is configured on the app.
func (c *Context) Logger() *slog.Logger {
	return c.slogger
}

// SetLogger overrides the logger for the current request context.
func (c *Context) SetLogger(logger *slog.Logger) {
	c.slogger = logger
}

// Param gets a path parameter by name.
// For example, this returns the value of id from /users/{id}
func (c *Context) Param(name string) string {
	if c.param1Set && c.param1Name == name {
		return c.param1Value
	}
	if c.param2Set && c.param2Name == name {
		return c.param2Value
	}

	value := c.Request.PathValue(name)
	if !c.param1Set {
		c.param1Name = name
		c.param1Value = value
		c.param1Set = true
		return value
	}

	c.param2Name = c.param1Name
	c.param2Value = c.param1Value
	c.param2Set = c.param1Set
	c.param1Name = name
	c.param1Value = value
	c.param1Set = true

	return value
}

// Query returns a named query parameter
func (c *Context) Query(name string) string {
	if c.query != nil {
		return c.query.Get(name)
	}

	value, ok := lookupRawQueryValue(c.Request.URL.RawQuery, name)
	if !ok {
		return ""
	}

	return value
}

// QueryPair returns two named query parameters using a single lookup path.
// This is useful in hot handlers that consistently read the same pair.
func (c *Context) QueryPair(name1, name2 string) (string, string) {
	if name1 == name2 {
		value := c.Query(name1)
		return value, value
	}

	if c.query != nil {
		return c.query.Get(name1), c.query.Get(name2)
	}

	v1, v2, _, _ := lookupRawQueryPair(c.Request.URL.RawQuery, name1, name2)
	return v1, v2
}

// QueryPairDefault returns two query parameters with independent defaults.
func (c *Context) QueryPairDefault(name1, default1, name2, default2 string) (string, string) {
	v1, v2 := c.QueryPair(name1, name2)
	if v1 == "" {
		v1 = default1
	}
	if v2 == "" {
		v2 = default2
	}

	return v1, v2
}

// QueryPairRaw returns two query parameters without URL decoding.
// This is a faster path for hot handlers when keys/values are plain ASCII.
func (c *Context) QueryPairRaw(name1, name2 string) (string, string) {
	if name1 == name2 {
		value, _ := lookupRawQueryValueRaw(c.Request.URL.RawQuery, name1)
		return value, value
	}

	v1, v2, _, _ := lookupRawQueryPairRaw(c.Request.URL.RawQuery, name1, name2)
	return v1, v2
}

// QueryPairDefaultRaw returns two raw query parameters with independent defaults.
func (c *Context) QueryPairDefaultRaw(name1, default1, name2, default2 string) (string, string) {
	v1, v2 := c.QueryPairRaw(name1, name2)
	if v1 == "" {
		v1 = default1
	}
	if v2 == "" {
		v2 = default2
	}

	return v1, v2
}

// DefaultQuery gets query param with default value
func (c *Context) DefaultQuery(name, defaultValue string) string {
	val := c.Query(name)
	if val == "" {
		return defaultValue
	}

	return val
}

// QueryInt reads an integer query parameter, returning defaultValue when absent.
func (c *Context) QueryInt(name string, defaultValue int) (int, error) {
	value := c.Query(name)
	if value == "" {
		return defaultValue, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}

	return parsed, nil
}

// QueryBool reads a boolean query parameter, returning defaultValue when absent.
func (c *Context) QueryBool(name string, defaultValue bool) (bool, error) {
	value := c.Query(name)
	if value == "" {
		return defaultValue, nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, err
	}

	return parsed, nil
}

// Form gets a form value
func (c *Context) Form(name string) string {
	// parse form values only once. its values cached by default once parsed
	if !c.formParsed {
		c.Request.ParseForm()
		c.formParsed = true
	}

	return c.Request.FormValue(name)
}

// File gets an uploaded file by key name. The file header containing the file is returned
func (c *Context) File(name string) (*multipart.FileHeader, error) {
	_, header, err := c.Request.FormFile(name)
	return header, err
}

// Cookie gets a request cookie by name
func (c *Context) Cookie(name string) (*http.Cookie, error) {
	return c.Request.Cookie(name)
}

// GetHeader retrieves a request header by key
func (c *Context) GetHeader(key string) string {
	return c.Request.Header.Get(key)
}

// HeaderInt reads an integer request header, returning defaultValue when absent.
func (c *Context) HeaderInt(key string, defaultValue int) (int, error) {
	value := c.GetHeader(key)
	if value == "" {
		return defaultValue, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}

	return parsed, nil
}

// HeaderBool reads a boolean request header, returning defaultValue when absent.
func (c *Context) HeaderBool(key string, defaultValue bool) (bool, error) {
	value := c.GetHeader(key)
	if value == "" {
		return defaultValue, nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, err
	}

	return parsed, nil
}

// Method returns the request method
func (c *Context) Method() string {
	return c.Request.Method
}

// Path retrieves the request path
func (c *Context) Path() string {
	return c.Request.URL.Path
}

// ClientIP returns the client IP address.
// Use this if you trust request headers passed to the server (ie: reverse proxy sits before server)
// else use c.Request.RemoteAddr()
func (c *Context) ClientIP() string {
	trustProxy := c.app == nil || c.app.isTrustedProxy(c.Request.RemoteAddr)
	if trustProxy {
		if forwarded := c.Request.Header.Get("X-Forwarded-For"); forwarded != "" {
			if c.app != nil && c.app.strictProxyHeaders {
				if ip, ok := parseForwardedForStrict(forwarded); ok {
					return ip
				}
			} else {
				if ip, ok := parseForwardedForBestEffort(forwarded); ok {
					return ip
				}
				ips := strings.Split(forwarded, ",")
				return strings.TrimSpace(ips[0])
			}
		}
		if realIP := c.Request.Header.Get("X-Real-IP"); realIP != "" {
			if c.app != nil && c.app.strictProxyHeaders {
				if ip, ok := parseIPCandidate(realIP); ok {
					return ip
				}
			} else {
				if ip, ok := parseIPCandidate(realIP); ok {
					return ip
				}
				return realIP
			}
		}
	}

	ip := c.Request.RemoteAddr
	if host, _, err := net.SplitHostPort(ip); err == nil {
		return host
	}

	// Fallback for non host:port inputs.
	ip = strings.TrimPrefix(ip, "[")
	ip = strings.TrimSuffix(ip, "]")
	return ip
}

func parseForwardedForStrict(forwarded string) (string, bool) {
	parts := strings.Split(forwarded, ",")
	if len(parts) == 0 {
		return "", false
	}

	first := ""
	for _, part := range parts {
		ip, ok := parseIPCandidate(part)
		if !ok {
			return "", false
		}
		if first == "" {
			first = ip
		}
	}

	return first, first != ""
}

func parseForwardedForBestEffort(forwarded string) (string, bool) {
	parts := strings.Split(forwarded, ",")
	for _, part := range parts {
		if ip, ok := parseIPCandidate(part); ok {
			return ip, true
		}
	}

	return "", false
}

func parseIPCandidate(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false
	}

	if ip := net.ParseIP(strings.Trim(value, "[]")); ip != nil {
		return ip.String(), true
	}

	host, _, err := net.SplitHostPort(value)
	if err != nil {
		return "", false
	}

	ip := net.ParseIP(strings.Trim(host, "[]"))
	if ip == nil {
		return "", false
	}

	return ip.String(), true
}

// Body reads the request body
func (c *Context) Body() ([]byte, error) {
	return io.ReadAll(c.Request.Body)
}

// Context returns the request original context from context.Context
func (c *Context) Context() context.Context {
	return c.Request.Context()
}

// binding methods

// DecodeJSON decodes a request body into a struct
func (c *Context) DecodeJSON(data interface{}) error {
	if c.Request.Body == nil {
		return ErrEmptyRequestBody
	}

	bufPtr := jsonBodyBufferPool.Get().(*[]byte)
	buf := (*bufPtr)[:0]
	if contentLength := c.Request.ContentLength; contentLength > 0 && int64(cap(buf)) < contentLength {
		buf = make([]byte, 0, contentLength)
	}

	body, err := readRequestBody(c.Request.Body, buf)
	if err != nil {
		releaseJSONBodyBuffer(bufPtr, body)
		return err
	}
	if len(body) == 0 {
		releaseJSONBodyBuffer(bufPtr, body)
		return io.EOF
	}

	err = json.Unmarshal(body, data)
	releaseJSONBodyBuffer(bufPtr, body)
	return err
}

// DecodeXML decodes a request body into a struct
func (c *Context) DecodeXML(data interface{}) error {
	if c.Request.Body == nil {
		return ErrEmptyRequestBody
	}

	return xml.NewDecoder(c.Request.Body).Decode(data)
}

// BindQuery binds URL query params into a struct using `query` tags.
func (c *Context) BindQuery(data interface{}) error {
	if c.query == nil {
		if c.app != nil && c.app.strictRequestParse {
			parsed, err := url.ParseQuery(c.Request.URL.RawQuery)
			if err != nil {
				return err
			}
			c.query = parsed
		} else {
			c.query = c.Request.URL.Query()
		}
	}

	if err := bindValues(c.query, "query", data); err != nil {
		return err
	}

	return validateStructTags(data)
}

// BindForm binds request form values into a struct using `form` tags.
func (c *Context) BindForm(data interface{}) error {
	if !c.formParsed {
		if err := c.Request.ParseForm(); err != nil {
			return err
		}
		c.formParsed = true
	}

	if err := bindValues(c.Request.Form, "form", data); err != nil {
		return err
	}

	return validateStructTags(data)
}

// BindHeader binds request headers into a struct using `header` tags.
func (c *Context) BindHeader(data interface{}) error {
	if err := bindHeader(c.Request.Header, data); err != nil {
		return err
	}

	return validateStructTags(data)
}

// Bind automatically binds request data based on method and Content-Type,
// then validates the result using `validate` struct tags.
func (c *Context) Bind(data interface{}) error {
	method := c.Request.Method
	if method == http.MethodGet || method == http.MethodHead || method == http.MethodDelete || method == http.MethodOptions {
		return c.BindQuery(data)
	}

	contentType := c.Request.Header.Get("Content-Type")
	mediaType, _, _ := mime.ParseMediaType(contentType)

	switch mediaType {
	case "application/json":
		if err := c.DecodeJSON(data); err != nil {
			return err
		}
	case "application/xml", "text/xml":
		if err := c.DecodeXML(data); err != nil {
			return err
		}
	case "application/x-www-form-urlencoded", "multipart/form-data":
		if err := c.BindForm(data); err != nil {
			return err
		}
	default:
		if err := c.BindQuery(data); err != nil {
			return err
		}
	}

	return validateStructTags(data)
}

func readRequestBody(body io.Reader, buf []byte) ([]byte, error) {
	for {
		if len(buf) == cap(buf) {
			newCap := cap(buf) * 2
			if newCap == 0 {
				newCap = 512
			}
			next := make([]byte, len(buf), newCap)
			copy(next, buf)
			buf = next
		}

		n, err := body.Read(buf[len(buf):cap(buf)])
		buf = buf[:len(buf)+n]
		if err == nil {
			continue
		}
		if err == io.EOF {
			return buf, nil
		}
		return buf, err
	}
}

func releaseJSONBodyBuffer(bufPtr *[]byte, body []byte) {
	if cap(body) > maxRetainedJSONBodyBuffer {
		resized := make([]byte, 0, 1024)
		*bufPtr = resized
		jsonBodyBufferPool.Put(bufPtr)
		return
	}

	*bufPtr = body[:0]
	jsonBodyBufferPool.Put(bufPtr)
}

// response methods

// JSON sends a JSON response
func (c *Context) JSON(status int, data interface{}) error {
	c.Response.Header()["Content-Type"] = jsonContentType
	c.Response.WriteHeader(status)

	return json.NewEncoder(c.Response).Encode(data)
}

// OK sends a JSON 200 response.
func (c *Context) OK(data interface{}) error {
	return c.JSON(http.StatusOK, data)
}

// Created sends a JSON 201 response.
func (c *Context) Created(data interface{}) error {
	return c.JSON(http.StatusCreated, data)
}

// Accepted sends a JSON 202 response.
func (c *Context) Accepted(data interface{}) error {
	return c.JSON(http.StatusAccepted, data)
}

// XML sends an XML response
func (c *Context) XML(status int, data interface{}) error {
	c.Response.Header()["Content-Type"] = xmlContentType
	c.Response.WriteHeader(status)

	return xml.NewEncoder(c.Response).Encode(data)
}

// Text sends a plain text response
func (c *Context) Text(status int, format string, values ...interface{}) error {
	c.Response.Header()["Content-Type"] = textContentType
	c.Response.WriteHeader(status)
	if len(values) == 0 {
		if sw, ok := c.Response.(stringWriter); ok {
			_, err := sw.WriteString(format)
			return err
		}
		_, err := fmt.Fprint(c.Response, format)
		return err
	}

	if len(values) == 1 && format == "%s" {
		if v, ok := values[0].(string); ok {
			if sw, ok := c.Response.(stringWriter); ok {
				_, err := sw.WriteString(v)
				return err
			}
			_, err := fmt.Fprint(c.Response, v)
			return err
		}
	}

	if len(values) == 2 && format == "%s-%s" {
		v0, ok0 := values[0].(string)
		v1, ok1 := values[1].(string)
		if ok0 && ok1 {
			if sw, ok := c.Response.(stringWriter); ok {
				if _, err := sw.WriteString(v0); err != nil {
					return err
				}
				if _, err := sw.WriteString("-"); err != nil {
					return err
				}
				_, err := sw.WriteString(v1)
				return err
			}
			_, err := fmt.Fprint(c.Response, v0, "-", v1)
			return err
		}
	}

	_, err := fmt.Fprintf(c.Response, format, values...)

	return err
}

// TextPair sends a plain text response by joining left + sep + right without variadic formatting.
func (c *Context) TextPair(status int, left, sep, right string) error {
	c.Response.Header()["Content-Type"] = textContentType
	c.Response.WriteHeader(status)

	if sw, ok := c.Response.(stringWriter); ok {
		if _, err := sw.WriteString(left); err != nil {
			return err
		}
		if _, err := sw.WriteString(sep); err != nil {
			return err
		}
		_, err := sw.WriteString(right)
		return err
	}

	total := len(left) + len(sep) + len(right)
	if total == 0 {
		return nil
	}
	if total <= 256 {
		var small [256]byte
		n := copy(small[:], left)
		n += copy(small[n:], sep)
		n += copy(small[n:], right)
		_, err := c.Response.Write(small[:n])
		return err
	}

	_, err := fmt.Fprint(c.Response, left, sep, right)
	return err
}

func (c *Context) HTML(status int, html string) error {
	c.Response.Header()["Content-Type"] = htmlContentType
	c.Response.WriteHeader(status)

	_, err := c.Response.Write([]byte(html))
	return err
}

// Data sends raw bytes
func (c *Context) Data(status int, contentType string, data []byte) error {
	c.SetHeader("Content-Type", contentType)
	c.Response.WriteHeader(status)

	_, err := c.Response.Write(data)
	return err
}

// NoContent sends a response with no body
func (c *Context) NoContent(status int) {
	c.Response.WriteHeader(status)
}

// Status sends a response status code with no body.
func (c *Context) Status(status int) {
	c.Response.WriteHeader(status)
}

// SetHeader sets a response header
func (c *Context) SetHeader(key, value string) {
	c.Response.Header().Set(key, value)
}

// ETag sets the ETag response header and returns the normalized value.
func (c *Context) ETag(tag string) string {
	normalized := normalizeETag(tag)
	if normalized != "" {
		c.SetHeader("ETag", normalized)
	}

	return normalized
}

// LastModified sets the Last-Modified response header.
func (c *Context) LastModified(modTime time.Time) {
	if modTime.IsZero() {
		return
	}

	c.SetHeader("Last-Modified", modTime.UTC().Format(http.TimeFormat))
}

// IsNotModified checks request cache validators and writes 304 when fresh.
// It also sets ETag and Last-Modified response headers when provided.
func (c *Context) IsNotModified(etag string, modTime time.Time) bool {
	normalizedETag := c.ETag(etag)
	c.LastModified(modTime)
	status := c.conditionalStatus(normalizedETag, modTime)
	if status == http.StatusNotModified {
		c.Status(http.StatusNotModified)
		return true
	}

	return false
}

// CheckPreconditions evaluates HTTP conditional headers and writes 304/412 when applicable.
// It sets ETag and Last-Modified response headers when values are provided.
func (c *Context) CheckPreconditions(etag string, modTime time.Time) bool {
	normalizedETag := c.ETag(etag)
	c.LastModified(modTime)

	status := c.conditionalStatus(normalizedETag, modTime)
	if status == 0 {
		return false
	}

	c.Status(status)
	return true
}

func (c *Context) conditionalStatus(normalizedETag string, modTime time.Time) int {
	method := c.Request.Method

	if ifMatch := c.GetHeader("If-Match"); ifMatch != "" {
		if !etagStrongMatch(ifMatch, normalizedETag) {
			return http.StatusPreconditionFailed
		}
	}

	if ifUnmodifiedSince := c.GetHeader("If-Unmodified-Since"); ifUnmodifiedSince != "" && !modTime.IsZero() {
		if t, ok := parseHTTPTime(ifUnmodifiedSince); ok {
			if modTime.UTC().Truncate(time.Second).After(t.UTC().Truncate(time.Second)) {
				return http.StatusPreconditionFailed
			}
		}
	}

	if ifNoneMatch := c.GetHeader("If-None-Match"); ifNoneMatch != "" {
		if etagMatches(ifNoneMatch, normalizedETag) {
			if method == http.MethodGet || method == http.MethodHead {
				return http.StatusNotModified
			}

			return http.StatusPreconditionFailed
		}

		// If-None-Match takes precedence over If-Modified-Since.
		return 0
	}

	if method != http.MethodGet && method != http.MethodHead {
		return 0
	}

	if modTime.IsZero() {
		return 0
	}

	ifModifiedSince := c.GetHeader("If-Modified-Since")
	if ifModifiedSince == "" {
		return 0
	}

	if t, ok := parseHTTPTime(ifModifiedSince); ok {
		if !modTime.UTC().Truncate(time.Second).After(t.UTC().Truncate(time.Second)) {
			return http.StatusNotModified
		}
	}

	return 0
}

// SetCookie sets a response cookie
func (c *Context) SetCookie(cookie *http.Cookie) {
	http.SetCookie(c.Response, cookie)
}

// Redirect redirects to a URL
func (c *Context) Redirect(status int, url string) {
	http.Redirect(c.Response, c.Request, url, status)
}

// Error sends a JSON error response with a stable shape.
func (c *Context) Error(status int, message string) error {
	return c.JSONError(status, message)
}

// JSONError sends a structured JSON error payload.
// Pass an optional details value to attach machine-readable context.
func (c *Context) JSONError(status int, message string, details ...interface{}) error {
	err := NewError(status, message)
	payload := JSONErrorPayload{Error: err.Message}
	if len(details) > 0 {
		payload.Details = details[0]
	}

	return c.JSON(status, payload)
}

// utilities

// SaveFile saves an uploaded file to the specified destination path.
func (c *Context) SaveFile(file *multipart.FileHeader, path string) error {
	// copy file to destination
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	dest, err := os.Create(path)
	if err != nil {
		return err
	}
	defer dest.Close()

	_, err = io.Copy(dest, src)
	return err
}

// StreamFile streams the content of a file in chunks to the client
func (c *Context) StreamFile(filepath string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return err
	}

	contentType := mime.TypeByExtension(filepath)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	rangeHeader := c.GetHeader("Range")

	if rangeHeader == "" {
		c.SetHeader("Content-Type", contentType)
		c.SetHeader("Content-Length", strconv.FormatInt(stat.Size(), 10))
		c.Response.WriteHeader(http.StatusOK)
		_, err = io.Copy(c.Response, file)
		return err
	}

	return c.serveRange(file, stat, rangeHeader, contentType)
}

// DownloadFile sends a downloadable file response with the specified filename
func (c *Context) DownloadFile(filepath string, downloadName string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return err
	}

	if downloadName == "" {
		downloadName = stat.Name()
	}

	contentType := mime.TypeByExtension(filepath)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	c.SetHeader("Content-Type", contentType)
	c.SetHeader("Content-Disposition", fmt.Sprintf("attachment; filename=%q", downloadName))
	c.SetHeader("Content-Length", strconv.FormatInt(stat.Size(), 10))
	c.Response.WriteHeader(http.StatusOK)

	_, err = io.Copy(c.Response, file)
	return err
}

func (c *Context) ServeStatic(dir string) error {
	path := filepath.Join(dir, c.Request.URL.Path)

	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if info.IsDir() {
		indexPath := filepath.Join(path, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			path = indexPath
		} else {
			return os.ErrNotExist
		}
	}

	return c.StreamFile(path)
}

// serveRange determines the exact range of the file content to serve on the current request context
func (c *Context) serveRange(file *os.File, stat os.FileInfo, rangeHeader, contentType string) error {
	rangePart := strings.TrimPrefix(rangeHeader, "bytes=")
	parts := strings.Split(rangePart, "-")

	start, end := int64(0), stat.Size()-1
	if len(parts) > 0 && parts[0] != "" {
		start, _ = strconv.ParseInt(parts[0], 10, 64)
	}
	if len(parts) > 1 && parts[1] != "" {
		end, _ = strconv.ParseInt(parts[1], 10, 64)
	}

	if start > end || start >= stat.Size() {
		c.SetHeader("Content-Range", fmt.Sprintf("bytes */%d", stat.Size()))
		c.Response.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
		return nil
	}

	if end >= stat.Size() {
		end = stat.Size() - 1
	}

	contentLength := end - start + 1

	file.Seek(start, io.SeekStart)

	c.SetHeader("Content-Type", contentType)
	c.SetHeader("Content-Length", strconv.FormatInt(contentLength, 10))
	c.SetHeader("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, stat.Size()))
	c.SetHeader("Accept-Ranges", "bytes")
	c.Response.WriteHeader(http.StatusPartialContent)

	_, err := io.CopyN(c.Response, file, contentLength)
	return err
}

func bindHeader(header http.Header, data interface{}) error {
	target, err := validateBindTarget(data)
	if err != nil {
		return err
	}

	plan := getBindPlan(target.Type(), "header")
	for _, fieldPlan := range plan.fields {
		values := header.Values(fieldPlan.name)
		if len(values) == 0 {
			continue
		}

		if err := fieldPlan.binder(target.Field(fieldPlan.index), values); err != nil {
			return fmt.Errorf("bind header %q: %w", fieldPlan.name, err)
		}
	}

	return nil
}

func bindValues(values url.Values, tagName string, data interface{}) error {
	target, err := validateBindTarget(data)
	if err != nil {
		return err
	}

	plan := getBindPlan(target.Type(), tagName)
	for _, fieldPlan := range plan.fields {
		raw, ok := values[fieldPlan.name]
		if !ok || len(raw) == 0 {
			continue
		}

		if err := fieldPlan.binder(target.Field(fieldPlan.index), raw); err != nil {
			return fmt.Errorf("bind %s %q: %w", tagName, fieldPlan.name, err)
		}
	}

	return nil
}

func getBindPlan(typeOf reflect.Type, tagName string) bindPlan {
	key := bindPlanKey{typeOf: typeOf, tagName: tagName}
	if plan, ok := bindPlanCache.Load(key); ok {
		return plan.(bindPlan)
	}

	fields := make([]bindFieldPlan, 0, typeOf.NumField())
	for idx := 0; idx < typeOf.NumField(); idx++ {
		field := typeOf.Field(idx)
		if !field.IsExported() {
			continue
		}

		name := tagOrDefault(field, tagName)
		if name == "" {
			continue
		}

		if tagName == "header" {
			name = textproto.CanonicalMIMEHeaderKey(name)
		}

		fields = append(fields, bindFieldPlan{
			index:  idx,
			name:   name,
			binder: compileBindFieldValueFunc(field.Type),
		})
	}

	plan := bindPlan{fields: fields}
	actual, _ := bindPlanCache.LoadOrStore(key, plan)
	return actual.(bindPlan)
}

func validateBindTarget(data interface{}) (reflect.Value, error) {
	v := reflect.ValueOf(data)
	if !v.IsValid() || v.Kind() != reflect.Ptr || v.IsNil() {
		return reflect.Value{}, ErrInvalidBindTarget
	}

	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return reflect.Value{}, ErrInvalidBindTarget
	}

	return v, nil
}

func validateStructTags(data interface{}) error {
	target, err := validateBindTarget(data)
	if err != nil {
		return err
	}

	var errs ValidationErrors
	typeOf := target.Type()
	for idx := 0; idx < target.NumField(); idx++ {
		field := target.Field(idx)
		fieldType := typeOf.Field(idx)
		if !fieldType.IsExported() {
			continue
		}

		tag := strings.TrimSpace(fieldType.Tag.Get("validate"))
		if tag == "" || tag == "-" {
			continue
		}

		if err := validateFieldRules(fieldType.Name, field, tag); err != nil {
			errs = append(errs, err...)
		}
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}

func normalizeETag(tag string) string {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return ""
	}

	if strings.HasPrefix(tag, "W/\"") && strings.HasSuffix(tag, "\"") {
		return tag
	}
	if strings.HasPrefix(tag, "\"") && strings.HasSuffix(tag, "\"") {
		return tag
	}

	return "\"" + tag + "\""
}

func etagMatches(ifNoneMatchHeader, normalizedETag string) bool {
	if normalizedETag == "" {
		return false
	}

	header := strings.TrimSpace(ifNoneMatchHeader)
	if header == "" {
		return false
	}

	if header == "*" {
		return true
	}

	parts := strings.Split(header, ",")
	for _, part := range parts {
		candidate := strings.TrimSpace(part)
		if candidate == normalizedETag {
			return true
		}

		if strings.HasPrefix(candidate, "W/") {
			if strings.TrimSpace(strings.TrimPrefix(candidate, "W/")) == normalizedETag {
				return true
			}
		}
	}

	return false
}

func etagStrongMatch(ifMatchHeader, normalizedETag string) bool {
	if strings.TrimSpace(ifMatchHeader) == "*" {
		return normalizedETag != ""
	}

	if strings.HasPrefix(normalizedETag, "W/") || normalizedETag == "" {
		return false
	}

	parts := strings.Split(ifMatchHeader, ",")
	for _, part := range parts {
		candidate := strings.TrimSpace(part)
		if candidate == "" {
			continue
		}
		if strings.HasPrefix(candidate, "W/") {
			continue
		}
		if candidate == normalizedETag {
			return true
		}
	}

	return false
}

func parseHTTPTime(value string) (time.Time, bool) {
	t, err := http.ParseTime(value)
	if err != nil {
		return time.Time{}, false
	}

	return t, true
}

func validateFieldRules(fieldName string, value reflect.Value, ruleTag string) ValidationErrors {
	rules := strings.Split(ruleTag, ",")
	required := false

	for _, rule := range rules {
		r := strings.TrimSpace(rule)
		if r == "required" {
			required = true
			break
		}
	}

	if isEmptyValidationValue(value) {
		if required {
			return ValidationErrors{{Field: fieldName, Tag: "required", Value: value.Interface()}}
		}
		return nil
	}

	var errs ValidationErrors
	for _, rule := range rules {
		r := strings.TrimSpace(rule)
		if r == "" || r == "required" {
			continue
		}

		name, param, hasParam := strings.Cut(r, "=")
		name = strings.TrimSpace(name)
		param = strings.TrimSpace(param)

		if !hasParam {
			if name == "email" {
				s := valueToString(value)
				if s == "" || !strings.Contains(s, "@") || strings.HasPrefix(s, "@") || strings.HasSuffix(s, "@") {
					errs = append(errs, FieldValidationError{Field: fieldName, Tag: "email", Value: value.Interface()})
				}
			}
			continue
		}

		switch name {
		case "len":
			expected, err := strconv.Atoi(param)
			if err != nil || validationLength(value) != expected {
				errs = append(errs, FieldValidationError{Field: fieldName, Tag: "len", Param: param, Value: value.Interface()})
			}
		case "min":
			minValue, err := strconv.ParseFloat(param, 64)
			if err != nil || validationNumericOrLength(value) < minValue {
				errs = append(errs, FieldValidationError{Field: fieldName, Tag: "min", Param: param, Value: value.Interface()})
			}
		case "max":
			maxValue, err := strconv.ParseFloat(param, 64)
			if err != nil || validationNumericOrLength(value) > maxValue {
				errs = append(errs, FieldValidationError{Field: fieldName, Tag: "max", Param: param, Value: value.Interface()})
			}
		case "oneof":
			allowed := strings.Fields(param)
			current := valueToString(value)
			ok := false
			for _, v := range allowed {
				if current == v {
					ok = true
					break
				}
			}
			if !ok {
				errs = append(errs, FieldValidationError{Field: fieldName, Tag: "oneof", Param: param, Value: value.Interface()})
			}
		}
	}

	return errs
}

func isEmptyValidationValue(v reflect.Value) bool {
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return true
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.String, reflect.Array, reflect.Slice, reflect.Map:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface:
		return v.IsNil()
	}

	return false
}

func validationLength(v reflect.Value) int {
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return 0
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.String, reflect.Array, reflect.Slice, reflect.Map:
		return v.Len()
	default:
		return 0
	}
}

func validationNumericOrLength(v reflect.Value) float64 {
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return 0
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.String, reflect.Array, reflect.Slice, reflect.Map:
		return float64(v.Len())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(v.Uint())
	case reflect.Float32, reflect.Float64:
		return v.Float()
	default:
		return 0
	}
}

func valueToString(v reflect.Value) string {
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}

	if v.Kind() == reflect.String {
		return v.String()
	}

	return fmt.Sprint(v.Interface())
}

func tagOrDefault(field reflect.StructField, tagName string) string {
	name := field.Tag.Get(tagName)
	if name == "-" {
		return ""
	}
	if name != "" {
		return name
	}

	if field.Name == "" {
		return ""
	}

	return strings.ToLower(field.Name[:1]) + field.Name[1:]
}

func lookupRawQueryValue(rawQuery, name string) (string, bool) {
	for {
		part := rawQuery
		next := ""
		if idx := strings.IndexByte(rawQuery, '&'); idx >= 0 {
			part = rawQuery[:idx]
			next = rawQuery[idx+1:]
		}

		key, value, _ := strings.Cut(part, "=")

		if key == name {
			decodedValue, valueErr := decodeQueryComponent(value)
			if valueErr == nil {
				return decodedValue, true
			}
		} else if strings.IndexByte(key, '%') >= 0 || strings.IndexByte(key, '+') >= 0 {
			decodedKey, err := decodeQueryComponent(key)
			if err == nil && decodedKey == name {
				decodedValue, valueErr := decodeQueryComponent(value)
				if valueErr == nil {
					return decodedValue, true
				}
			}
		}

		if next == "" {
			return "", false
		}

		rawQuery = next
	}
}

func lookupRawQueryPair(rawQuery, name1, name2 string) (string, string, bool, bool) {
	var value1, value2 string
	var found1, found2 bool

	for start := 0; start <= len(rawQuery); {
		end := strings.IndexByte(rawQuery[start:], '&')
		if end < 0 {
			end = len(rawQuery)
		} else {
			end += start
		}

		eq := strings.IndexByte(rawQuery[start:end], '=')
		key := ""
		value := ""
		if eq < 0 {
			key = rawQuery[start:end]
		} else {
			eq += start
			key = rawQuery[start:eq]
			value = rawQuery[eq+1 : end]
		}

		matchedName := ""

		switch key {
		case name1:
			matchedName = name1
		case name2:
			matchedName = name2
		default:
			if strings.IndexByte(key, '%') >= 0 || strings.IndexByte(key, '+') >= 0 {
				decodedKey, err := decodeQueryComponent(key)
				if err == nil {
					switch decodedKey {
					case name1:
						matchedName = name1
					case name2:
						matchedName = name2
					}
				}
			}
		}

		if matchedName != "" {
			decodedValue, valueErr := decodeQueryComponent(value)
			if valueErr == nil {
				if matchedName == name1 && !found1 {
					value1 = decodedValue
					found1 = true
				}
				if matchedName == name2 && !found2 {
					value2 = decodedValue
					found2 = true
				}
			}
		}

		if found1 && found2 {
			return value1, value2, true, true
		}

		if end == len(rawQuery) {
			return value1, value2, found1, found2
		}

		start = end + 1
	}

	return value1, value2, found1, found2
}

func lookupRawQueryPairRaw(rawQuery, name1, name2 string) (string, string, bool, bool) {
	var value1, value2 string
	var found1, found2 bool
	n := len(rawQuery)
	segmentStart := 0
	equalIndex := -1

	for i := 0; i <= n; i++ {
		if i < n {
			b := rawQuery[i]
			if b == '=' {
				if equalIndex < 0 {
					equalIndex = i
				}
				continue
			}
			if b != '&' {
				continue
			}
		}

		segmentEnd := i
		keyStart := segmentStart
		keyEnd := segmentEnd
		valueStart := segmentEnd
		if equalIndex >= 0 {
			keyEnd = equalIndex
			valueStart = equalIndex + 1
		}

		key := rawQuery[keyStart:keyEnd]
		if !found1 && key == name1 {
			value1 = rawQuery[valueStart:segmentEnd]
			found1 = true
		}
		if !found2 && key == name2 {
			value2 = rawQuery[valueStart:segmentEnd]
			found2 = true
		}

		if found1 && found2 {
			return value1, value2, true, true
		}

		segmentStart = i + 1
		equalIndex = -1
	}

	return value1, value2, found1, found2
}

func lookupRawQueryValueRaw(rawQuery, name string) (string, bool) {
	value, _, found, _ := lookupRawQueryPairRaw(rawQuery, name, "")
	return value, found
}

func decodeQueryComponent(value string) (string, error) {
	if strings.IndexByte(value, '%') < 0 && strings.IndexByte(value, '+') < 0 {
		return value, nil
	}

	return url.QueryUnescape(value)
}

func compileBindFieldValueFunc(typeOf reflect.Type) bindFieldValueFunc {
	if typeOf.Kind() == reflect.Ptr {
		elemType := typeOf.Elem()
		elemBinder := compileBindFieldValueFunc(elemType)
		return func(field reflect.Value, values []string) error {
			if field.IsNil() {
				field.Set(reflect.New(elemType))
			}
			return elemBinder(field.Elem(), values)
		}
	}

	if typeOf.Kind() == reflect.Slice {
		elemBinder := compileBindScalarStringFunc(typeOf.Elem())
		return func(field reflect.Value, values []string) error {
			if len(values) == 0 {
				return nil
			}

			result := reflect.MakeSlice(field.Type(), len(values), len(values))
			for idx, raw := range values {
				if err := elemBinder(result.Index(idx), raw); err != nil {
					return err
				}
			}

			field.Set(result)
			return nil
		}
	}

	scalarBinder := compileBindScalarStringFunc(typeOf)
	return func(field reflect.Value, values []string) error {
		if len(values) == 0 {
			return nil
		}

		return scalarBinder(field, values[0])
	}
}

func compileBindScalarStringFunc(typeOf reflect.Type) func(field reflect.Value, value string) error {
	if typeOf.Kind() == reflect.Ptr {
		elemType := typeOf.Elem()
		elemBinder := compileBindScalarStringFunc(elemType)
		return func(field reflect.Value, value string) error {
			if field.IsNil() {
				field.Set(reflect.New(elemType))
			}
			return elemBinder(field.Elem(), value)
		}
	}

	switch typeOf.Kind() {
	case reflect.String:
		return func(field reflect.Value, value string) error {
			field.SetString(value)
			return nil
		}
	case reflect.Bool:
		return func(field reflect.Value, value string) error {
			parsed, err := strconv.ParseBool(value)
			if err != nil {
				return err
			}
			field.SetBool(parsed)
			return nil
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return func(field reflect.Value, value string) error {
			parsed, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return err
			}
			field.SetInt(parsed)
			return nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return func(field reflect.Value, value string) error {
			parsed, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return err
			}
			field.SetUint(parsed)
			return nil
		}
	case reflect.Float32, reflect.Float64:
		return func(field reflect.Value, value string) error {
			parsed, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return err
			}
			field.SetFloat(parsed)
			return nil
		}
	default:
		return func(field reflect.Value, value string) error {
			return fmt.Errorf("unsupported kind %s", field.Kind())
		}
	}
}
