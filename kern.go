// Package kern provides a lightweight web framework for Go.
//
// kern is built on Go 1.22's enhanced net/http router with zero dependencies.
// It provides a simple, intuitive API for building web applications while
// leveraging the standard library's performance and reliability.
//
// Example usage:
//
//	app := kern.Default()
//
//	app.GET("/", func(c *kern.Context) {
//	    c.JSON(200, map[string]string{"message": "Hello, kern!"})
//	})
//
//	app.Run(":8080")
//
// Features:
//   - Go 1.22+ native routing with method matching and path parameters
//   - Standard http.Handler middleware pattern
//   - Route groups for organization
//   - Zero dependencies
package kern

import (
	"context"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// App is the main application instance
type App struct {
	router             *http.ServeMux
	middlewares        []MiddlewareFunc
	pool               sync.Pool
	logger             *log.Logger
	slogger            *slog.Logger
	bodyLimit          int64
	handler            atomic.Value
	cachedHandler      http.Handler // fast-path: cached handler to avoid atomic load
	routesMu           sync.RWMutex
	routes             []RouteInfo
	routeNames         map[string]RouteInfo
	trustedProxyNets   []*net.IPNet
	trustedProxyIPs    map[string]struct{}
	strictProxyHeaders bool
	strictRequestParse bool
	// whether to run the app in debug mode or not
	debug bool

	onRouteHooks    []RouteHook
	onListenHooks   []ListenHook
	onShutdownHooks []ShutdownHook
	onErrorHooks    []ErrorHook
}

// HandlerFunc is the handler signature
type HandlerFunc func(c *Context)

// PathConstraint validates a single path parameter value.
type PathConstraint func(value string) bool

// PathConstraints maps path parameter names to validators.
type PathConstraints map[string]PathConstraint

// MiddlewareFunc is the middleware signature
type MiddlewareFunc func(http.Handler) http.Handler

// Option configures the app
type Option func(*App)

// RunOption configures the server
type RunOption func(*serverConfig)

type serverConfig struct {
	readTimeout       time.Duration
	readHeaderTimeout time.Duration
	writeTimeout      time.Duration
	idleTimeout       time.Duration
	maxHeaderBytes    int
	keepAlivesEnabled bool
	gracefulTimeout   time.Duration
}

type handlerRef struct {
	http.Handler
}

// New instantiates a new app instance
func New(opts ...Option) *App {
	app := &App{
		router:          http.NewServeMux(),
		logger:          log.New(os.Stdout, "[kern] ", log.LstdFlags),
		routeNames:      make(map[string]RouteInfo),
		trustedProxyIPs: make(map[string]struct{}),
	}
	app.handler.Store(handlerRef{Handler: http.Handler(app.router)})

	// setup context pool
	app.pool.New = func() interface{} {
		return &Context{app: app}
	}

	// apply server configuration
	for _, opt := range opts {
		opt(app)
	}

	return app
}

// Default instantiates an app with common settings
func Default() *App {
	app := New()
	app.Use(Logger())
	app.Use(Recovery())

	return app
}

// app configuration

// WithLogger adds logger middleware
func WithLogger() Option {
	return func(app *App) {
		app.Use(Logger())
	}
}

// WithSlogLogger configures app lifecycle logging with slog.
func WithSlogLogger(logger *slog.Logger) Option {
	return func(app *App) {
		app.slogger = logger
	}
}

// WithRecovery adds recovery middleware
func WithRecovery() Option {
	return func(app *App) {
		app.Use(Recovery())
	}
}

// WithDebug enables debug mode
func WithDebug() Option {
	return func(app *App) {
		app.debug = true
	}
}

// WithBodyLimit sets a max number of bytes readable from request bodies.
func WithBodyLimit(limit int64) Option {
	return func(app *App) {
		app.bodyLimit = limit
	}
}

// WithTrustedProxies configures reverse proxies whose forwarding headers are trusted.
// Entries may be plain IP addresses (e.g. "10.0.0.1") or CIDR blocks (e.g. "10.0.0.0/24").
func WithTrustedProxies(entries ...string) Option {
	return func(app *App) {
		app.trustedProxyNets = app.trustedProxyNets[:0]
		app.trustedProxyIPs = make(map[string]struct{})

		for _, entry := range entries {
			entry = strings.TrimSpace(entry)
			if entry == "" {
				continue
			}

			if strings.Contains(entry, "/") {
				_, ipNet, err := net.ParseCIDR(entry)
				if err != nil {
					panic("kern.WithTrustedProxies: invalid CIDR " + entry)
				}
				app.trustedProxyNets = append(app.trustedProxyNets, ipNet)
				continue
			}

			ip := net.ParseIP(entry)
			if ip == nil {
				panic("kern.WithTrustedProxies: invalid IP " + entry)
			}
			app.trustedProxyIPs[ip.String()] = struct{}{}
		}
	}
}

// WithStrictProxyHeaders enables strict validation of forwarding headers.
// When enabled, malformed X-Forwarded-For or X-Real-IP values are ignored.
func WithStrictProxyHeaders(enabled bool) Option {
	return func(app *App) {
		app.strictProxyHeaders = enabled
	}
}

// WithStrictRequestParsing enables strict query parsing for binding helpers.
// When enabled, malformed URL query strings return errors in BindQuery/Bind.
func WithStrictRequestParsing(enabled bool) Option {
	return func(app *App) {
		app.strictRequestParse = enabled
	}
}

func (app *App) isTrustedProxy(remoteAddr string) bool {
	if len(app.trustedProxyIPs) == 0 && len(app.trustedProxyNets) == 0 {
		return true
	}

	host := remoteAddr
	if parsedHost, _, err := net.SplitHostPort(remoteAddr); err == nil {
		host = parsedHost
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	if ip == nil {
		return false
	}

	if _, ok := app.trustedProxyIPs[ip.String()]; ok {
		return true
	}

	for _, n := range app.trustedProxyNets {
		if n.Contains(ip) {
			return true
		}
	}

	return false
}

// app core methods

// ServeHTTP implements the handler for serving each request
func (app *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Fast path: use cached handler to avoid atomic load overhead
	if app.cachedHandler != nil {
		app.cachedHandler.ServeHTTP(w, r)
		return
	}

	// Slow path: atomic load (for backwards compatibility)
	ref, _ := app.handler.Load().(handlerRef)
	handler := ref.Handler
	handler.ServeHTTP(w, r)
}

// buildHandler creates the middleware chain
func (app *App) buildHandler() http.Handler {
	handler := http.Handler(app.router)

	// chain middleware in reverse order so they execute in order which they were added
	for idx := len(app.middlewares) - 1; idx >= 0; idx-- {
		handler = app.middlewares[idx](handler)
	}

	// Cache the handler for fast-path access (eliminates atomic load overhead)
	app.cachedHandler = handler

	return handler
}

// Use adds a global middleware to the application
func (app *App) Use(middlewares ...MiddlewareFunc) {
	app.middlewares = append(app.middlewares, middlewares...)
	app.handler.Store(handlerRef{Handler: app.buildHandler()})
}

func (app *App) registerRoute(info RouteInfo) {
	app.routesMu.Lock()
	defer app.routesMu.Unlock()

	if info.Name != "" {
		if _, exists := app.routeNames[info.Name]; exists {
			panic("kern: duplicate route name: " + info.Name)
		}
		app.routeNames[info.Name] = info
	}

	app.routes = append(app.routes, info)
}

// Routes returns a snapshot of all registered routes.
func (app *App) Routes() []RouteInfo {
	app.routesMu.RLock()
	defer app.routesMu.RUnlock()

	routes := make([]RouteInfo, len(app.routes))
	copy(routes, app.routes)
	return routes
}

// RouteByName retrieves a route by its unique name.
func (app *App) RouteByName(name string) (RouteInfo, bool) {
	app.routesMu.RLock()
	defer app.routesMu.RUnlock()

	route, ok := app.routeNames[name]
	return route, ok
}

// internal routing method to fit the http.Handler method signature
func (app *App) handle(method, path string, handler HandlerFunc) {
	app.handleNamedWithConstraintsAndMiddleware(method, path, "", nil, handler, nil)
}

func (app *App) handleNamed(method, path, name string, handler HandlerFunc) {
	app.handleNamedWithConstraintsAndMiddleware(method, path, name, nil, handler, nil)
}

func (app *App) handleNamedWithConstraints(method, path, name string, constraints PathConstraints, handler HandlerFunc) {
	app.handleNamedWithConstraintsAndMiddleware(method, path, name, constraints, handler, nil)
}

func (app *App) handleNamedWithConstraintsAndMiddleware(
	method,
	path,
	name string,
	constraints PathConstraints,
	handler HandlerFunc,
	routeMiddlewares []MiddlewareFunc,
) {
	// std lib pattern is space delimiter of method and path
	pattern := method + " " + path
	routeInfo := RouteInfo{Method: method, Path: path, Name: name}
	app.registerRoute(routeInfo)
	app.emitRoute(routeInfo)
	constraints = clonePathConstraints(constraints)

	wrappedHandler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !validatePathConstraints(r, constraints) {
			http.NotFound(w, r)
			return
		}

		// Apply max body reader once at the framework boundary so bind/decode
		// helpers and handlers inherit the same request-size guarantees.
		if app.bodyLimit > 0 {
			r.Body = http.MaxBytesReader(w, r.Body, app.bodyLimit)
		}

		c, _ := app.pool.Get().(*Context)
		c.reset(w, r)
		handler(c)
		app.pool.Put(c)
	}))

	for idx := len(routeMiddlewares) - 1; idx >= 0; idx-- {
		wrappedHandler = routeMiddlewares[idx](wrappedHandler)
	}

	app.router.Handle(pattern, wrappedHandler)
}

// routing methods

// Route registers a handler for the given method and path
func (app *App) Route(method, path string, handler HandlerFunc) {
	app.handle(method, path, handler)
}

// RouteNamed registers a named handler for the given method and path.
func (app *App) RouteNamed(name, method, path string, handler HandlerFunc) {
	app.handleNamed(method, path, name, handler)
}

// RouteWithConstraints registers a handler with typed path constraints.
func (app *App) RouteWithConstraints(method, path string, constraints PathConstraints, handler HandlerFunc) {
	app.handleNamedWithConstraints(method, path, "", constraints, handler)
}

// RouteNamedWithConstraints registers a named handler with typed path constraints.
func (app *App) RouteNamedWithConstraints(name, method, path string, constraints PathConstraints, handler HandlerFunc) {
	app.handleNamedWithConstraints(method, path, name, constraints, handler)
}

// RouteWithMiddleware registers a handler with route-specific middleware.
func (app *App) RouteWithMiddleware(method, path string, handler HandlerFunc, middlewares ...MiddlewareFunc) {
	app.handleNamedWithConstraintsAndMiddleware(method, path, "", nil, handler, middlewares)
}

// RouteNamedWithMiddleware registers a named handler with route-specific middleware.
func (app *App) RouteNamedWithMiddleware(name, method, path string, handler HandlerFunc, middlewares ...MiddlewareFunc) {
	app.handleNamedWithConstraintsAndMiddleware(method, path, name, nil, handler, middlewares)
}

// GET registers a GET route
func (app *App) GET(path string, handler HandlerFunc) {
	app.handle(http.MethodGet, path, handler)
}

// GETNamed registers a named GET route.
func (app *App) GETNamed(name, path string, handler HandlerFunc) {
	app.handleNamed(http.MethodGet, path, name, handler)
}

// POST registers a POST route
func (app *App) POST(path string, handler HandlerFunc) {
	app.handle(http.MethodPost, path, handler)
}

// POSTNamed registers a named POST route.
func (app *App) POSTNamed(name, path string, handler HandlerFunc) {
	app.handleNamed(http.MethodPost, path, name, handler)
}

// PATCH registers a PATCH route
func (app *App) PATCH(path string, handler HandlerFunc) {
	app.handle(http.MethodPatch, path, handler)
}

// PATCHNamed registers a named PATCH route.
func (app *App) PATCHNamed(name, path string, handler HandlerFunc) {
	app.handleNamed(http.MethodPatch, path, name, handler)
}

// PUT registers a PUT route
func (app *App) PUT(path string, handler HandlerFunc) {
	app.handle(http.MethodPut, path, handler)
}

// PUTNamed registers a named PUT route.
func (app *App) PUTNamed(name, path string, handler HandlerFunc) {
	app.handleNamed(http.MethodPut, path, name, handler)
}

// DELETE registers a DELETE route
func (app *App) DELETE(path string, handler HandlerFunc) {
	app.handle(http.MethodDelete, path, handler)
}

// DELETENamed registers a named DELETE route.
func (app *App) DELETENamed(name, path string, handler HandlerFunc) {
	app.handleNamed(http.MethodDelete, path, name, handler)
}

// HEAD registers a HEAD route
func (app *App) HEAD(path string, handler HandlerFunc) {
	app.handle(http.MethodHead, path, handler)
}

// HEADNamed registers a named HEAD route.
func (app *App) HEADNamed(name, path string, handler HandlerFunc) {
	app.handleNamed(http.MethodHead, path, name, handler)
}

// OPTIONS registers a OPTIONS route
func (app *App) OPTIONS(path string, handler HandlerFunc) {
	app.handle(http.MethodOptions, path, handler)
}

// OPTIONSNamed registers a named OPTIONS route.
func (app *App) OPTIONSNamed(name, path string, handler HandlerFunc) {
	app.handleNamed(http.MethodOptions, path, name, handler)
}

func (app *App) Static(prefix, dir string) {
	fileServer := http.FileServer(http.Dir(dir))
	strippedPrefix := strings.TrimSuffix(prefix, "/")
	app.router.Handle("GET "+strippedPrefix+"/{path...}", http.StripPrefix(strippedPrefix+"/", fileServer))
}

// Group creates a route group with common prefix and middleware
func (app *App) Group(prefix string, middlewares ...MiddlewareFunc) *Group {
	return &Group{
		prefix:      prefix,
		middlewares: middlewares,
		app:         app,
	}
}

// run options

// WithReadTimeout sets the server read timeout
func WithReadTimeout(duration time.Duration) RunOption {
	return func(cfg *serverConfig) {
		cfg.readTimeout = duration
	}
}

// WithReadHeaderTimeout sets the server read header timeout.
func WithReadHeaderTimeout(duration time.Duration) RunOption {
	return func(cfg *serverConfig) {
		cfg.readHeaderTimeout = duration
	}
}

// WithWriteTimeout sets the server write timeout
func WithWriteTimeout(duration time.Duration) RunOption {
	return func(cfg *serverConfig) {
		cfg.writeTimeout = duration
	}
}

// WithIdleTimeout sets the server idle timeout.
func WithIdleTimeout(duration time.Duration) RunOption {
	return func(cfg *serverConfig) {
		cfg.idleTimeout = duration
	}
}

// WithMaxHeaderBytes sets the server MaxHeaderBytes value.
func WithMaxHeaderBytes(max int) RunOption {
	return func(cfg *serverConfig) {
		if max > 0 {
			cfg.maxHeaderBytes = max
		}
	}
}

// WithKeepAlivesEnabled enables or disables HTTP keep-alives.
func WithKeepAlivesEnabled(enabled bool) RunOption {
	return func(cfg *serverConfig) {
		cfg.keepAlivesEnabled = enabled
	}
}

// WithGracefulShutdown sets the graceful shutdown timeout.
func WithGracefulShutdown(timeout time.Duration) RunOption {
	return func(cfg *serverConfig) {
		cfg.gracefulTimeout = timeout
	}
}

// server methods

// Run starts the http server
func (app *App) Run(addr string, opts ...RunOption) error {
	cfg := &serverConfig{
		readTimeout:       30 * time.Second,
		readHeaderTimeout: 10 * time.Second,
		writeTimeout:      30 * time.Second,
		idleTimeout:       60 * time.Second,
		maxHeaderBytes:    http.DefaultMaxHeaderBytes,
		keepAlivesEnabled: true,
		gracefulTimeout:   0,
	}

	// apply provided options
	for _, opt := range opts {
		opt(cfg)
	}

	server := &http.Server{
		Addr:              addr,
		Handler:           app,
		ReadTimeout:       cfg.readTimeout,
		ReadHeaderTimeout: cfg.readHeaderTimeout,
		WriteTimeout:      cfg.writeTimeout,
		IdleTimeout:       cfg.idleTimeout,
		MaxHeaderBytes:    cfg.maxHeaderBytes,
	}
	server.SetKeepAlivesEnabled(cfg.keepAlivesEnabled)

	app.emitListen(ListenInfo{Addr: addr, TLS: false})

	if cfg.gracefulTimeout > 0 {
		return app.runWithGracefulShutdown(server, cfg.gracefulTimeout)
	}

	app.logInfo("server_started", slog.String("addr", addr), slog.Bool("tls", false))
	err := server.ListenAndServe()
	if err != nil {
		app.logError("server_listen_error", err, slog.String("addr", addr), slog.Bool("tls", false))
		app.emitError(err)
	}
	return err
}

// RunTLS starts the HTTPS server
func (app *App) RunTLS(addr, certFile, keyFile string, opts ...RunOption) error {
	cfg := &serverConfig{
		readTimeout:       30 * time.Second,
		readHeaderTimeout: 10 * time.Second,
		writeTimeout:      30 * time.Second,
		idleTimeout:       60 * time.Second,
		maxHeaderBytes:    http.DefaultMaxHeaderBytes,
		keepAlivesEnabled: true,
	}

	// apply provided options
	for _, opt := range opts {
		opt(cfg)
	}

	server := &http.Server{
		Addr:              addr,
		Handler:           app,
		ReadTimeout:       cfg.readTimeout,
		ReadHeaderTimeout: cfg.readHeaderTimeout,
		WriteTimeout:      cfg.writeTimeout,
		IdleTimeout:       cfg.idleTimeout,
		MaxHeaderBytes:    cfg.maxHeaderBytes,
	}
	server.SetKeepAlivesEnabled(cfg.keepAlivesEnabled)

	app.emitListen(ListenInfo{Addr: addr, TLS: true})
	app.logInfo("server_started", slog.String("addr", addr), slog.Bool("tls", true))
	err := server.ListenAndServeTLS(certFile, keyFile)
	if err != nil {
		app.logError("server_listen_error", err, slog.String("addr", addr), slog.Bool("tls", true))
		app.emitError(err)
	}
	return err
}

func (app *App) runWithGracefulShutdown(server *http.Server, timeout time.Duration) error {
	errChan := make(chan error, 1)

	// run non blocking server
	go func(addr string) {
		app.logInfo("server_started", slog.String("addr", server.Addr), slog.Bool("graceful", true), slog.Bool("tls", false))
		// send error reports
		errChan <- server.ListenAndServe()
	}(server.Addr)

	// wait for shutdown or os interrupts
	quitChan := make(chan os.Signal, 1)
	signal.Notify(quitChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	// server error
	case err := <-errChan:
		if err != nil {
			app.logError("server_listen_error", err, slog.String("addr", server.Addr), slog.Bool("tls", false))
			app.emitError(err)
		}
		app.emitShutdown(err)
		return err
	// os interrupt
	case <-quitChan:
		app.logInfo("server_shutdown_start", slog.String("addr", server.Addr))

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			app.logError("server_shutdown_error", err, slog.String("addr", server.Addr))
			app.emitError(err)
			app.emitShutdown(err)
			return err
		}

		app.logInfo("server_shutdown_complete", slog.String("addr", server.Addr))
		app.emitShutdown(nil)
		return nil
	}
}

func (app *App) logInfo(msg string, attrs ...slog.Attr) {
	if app.slogger != nil {
		app.slogger.LogAttrs(context.Background(), slog.LevelInfo, msg, attrs...)
		return
	}

	app.logger.Print(msg)
}

func (app *App) logError(msg string, err error, attrs ...slog.Attr) {
	if app.slogger != nil {
		all := make([]slog.Attr, 0, len(attrs)+1)
		all = append(all, attrs...)
		all = append(all, slog.Any("error", err))
		app.slogger.LogAttrs(context.Background(), slog.LevelError, msg, all...)
		return
	}

	if err != nil {
		app.logger.Printf("%s: %v", msg, err)
		return
	}

	app.logger.Print(msg)
}
