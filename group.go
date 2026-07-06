package kern

import "net/http"

// Group is a route group with common named prefix
type Group struct {
	prefix      string
	middlewares []MiddlewareFunc

	app *App
}

// Use registers middlewares to the group
func (g *Group) Use(middlewares ...MiddlewareFunc) {
	g.middlewares = append(g.middlewares, middlewares...)
}

// Group creates a sub-group. Global middlewares come first in the chain
func (g *Group) Group(prefix string, middlewares ...MiddlewareFunc) *Group {
	// copy all applicable middlewares
	return &Group{
		prefix:      g.prefix + prefix,
		middlewares: append(append([]MiddlewareFunc{}, g.middlewares...), middlewares...),
		app:         g.app,
	}
}

func (g *Group) handle(method, path string, handler HandlerFunc) {
	g.handleNamedWithConstraintsAndMiddleware(method, path, "", nil, handler, nil)
}

func (g *Group) handleNamed(method, path, name string, handler HandlerFunc) {
	g.handleNamedWithConstraintsAndMiddleware(method, path, name, nil, handler, nil)
}

func (g *Group) handleNamedWithConstraints(method, path, name string, constraints PathConstraints, handler HandlerFunc) {
	g.handleNamedWithConstraintsAndMiddleware(method, path, name, constraints, handler, nil)
}

func (g *Group) handleNamedWithConstraintsAndMiddleware(
	method,
	path,
	name string,
	constraints PathConstraints,
	handler HandlerFunc,
	routeMiddlewares []MiddlewareFunc,
) {
	// compose full path with group prefix
	path = g.prefix + path
	constraints = clonePathConstraints(constraints)

	if len(g.middlewares) == 0 {
		g.app.handleNamedWithConstraintsAndMiddleware(method, path, name, constraints, handler, routeMiddlewares)
		return
	}

	routeInfo := RouteInfo{Method: method, Path: path, Name: name}
	g.app.registerRoute(routeInfo)
	g.app.emitRoute(routeInfo)

	// Compose middleware using http.Handler so any valid middleware return type works.
	var wrappedHandler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !validatePathConstraints(r, constraints) {
			http.NotFound(w, r)
			return
		}

		if g.app.bodyLimit > 0 {
			r.Body = http.MaxBytesReader(w, r.Body, g.app.bodyLimit)
		}

		// retrieve context from pool
		c := g.app.pool.Get().(*Context)
		c.reset(w, r)

		handler(c)
		g.app.pool.Put(c)
	})

	allMiddlewares := make([]MiddlewareFunc, 0, len(g.middlewares)+len(routeMiddlewares))
	allMiddlewares = append(allMiddlewares, g.middlewares...)
	allMiddlewares = append(allMiddlewares, routeMiddlewares...)

	for idx := len(allMiddlewares) - 1; idx >= 0; idx-- {
		wrappedHandler = allMiddlewares[idx](wrappedHandler)
	}

	// handle request with middlewares
	pattern := method + " " + path
	g.app.router.Handle(pattern, wrappedHandler)
}

// GET registers a GET route for the group
func (g *Group) GET(path string, handler HandlerFunc) {
	g.handle(http.MethodGet, path, handler)
}

// GETNamed registers a named GET route for the group.
func (g *Group) GETNamed(name, path string, handler HandlerFunc) {
	g.handleNamed(http.MethodGet, path, name, handler)
}

// POST registers a POST route for the group
func (g *Group) POST(path string, handler HandlerFunc) {
	g.handle(http.MethodPost, path, handler)
}

// POSTNamed registers a named POST route for the group.
func (g *Group) POSTNamed(name, path string, handler HandlerFunc) {
	g.handleNamed(http.MethodPost, path, name, handler)
}

// PATCH registers a PATCH route for the group
func (g *Group) PATCH(path string, handler HandlerFunc) {
	g.handle(http.MethodPatch, path, handler)
}

// PATCHNamed registers a named PATCH route for the group.
func (g *Group) PATCHNamed(name, path string, handler HandlerFunc) {
	g.handleNamed(http.MethodPatch, path, name, handler)
}

// PUT registers a PUT route for the group
func (g *Group) PUT(path string, handler HandlerFunc) {
	g.handle(http.MethodPut, path, handler)
}

// PUTNamed registers a named PUT route for the group.
func (g *Group) PUTNamed(name, path string, handler HandlerFunc) {
	g.handleNamed(http.MethodPut, path, name, handler)
}

// DELETE registers a DELETE route for the group
func (g *Group) DELETE(path string, handler HandlerFunc) {
	g.handle(http.MethodDelete, path, handler)
}

// DELETENamed registers a named DELETE route for the group.
func (g *Group) DELETENamed(name, path string, handler HandlerFunc) {
	g.handleNamed(http.MethodDelete, path, name, handler)
}

// HEAD registers a HEAD route for the group
func (g *Group) HEAD(path string, handler HandlerFunc) {
	g.handle(http.MethodHead, path, handler)
}

// HEADNamed registers a named HEAD route for the group.
func (g *Group) HEADNamed(name, path string, handler HandlerFunc) {
	g.handleNamed(http.MethodHead, path, name, handler)
}

// OPTIONS registers a OPTIONS route for the group
func (g *Group) OPTIONS(path string, handler HandlerFunc) {
	g.handle(http.MethodOptions, path, handler)
}

// OPTIONSNamed registers a named OPTIONS route for the group.
func (g *Group) OPTIONSNamed(name, path string, handler HandlerFunc) {
	g.handleNamed(http.MethodOptions, path, name, handler)
}

// RouteWithConstraints registers a constrained route for the group.
func (g *Group) RouteWithConstraints(method, path string, constraints PathConstraints, handler HandlerFunc) {
	g.handleNamedWithConstraints(method, path, "", constraints, handler)
}

// RouteNamedWithConstraints registers a named constrained route for the group.
func (g *Group) RouteNamedWithConstraints(name, method, path string, constraints PathConstraints, handler HandlerFunc) {
	g.handleNamedWithConstraints(method, path, name, constraints, handler)
}

// RouteWithMiddleware registers a route with route-specific middleware.
func (g *Group) RouteWithMiddleware(method, path string, handler HandlerFunc, middlewares ...MiddlewareFunc) {
	g.handleNamedWithConstraintsAndMiddleware(method, path, "", nil, handler, middlewares)
}

// RouteNamedWithMiddleware registers a named route with route-specific middleware.
func (g *Group) RouteNamedWithMiddleware(name, method, path string, handler HandlerFunc, middlewares ...MiddlewareFunc) {
	g.handleNamedWithConstraintsAndMiddleware(method, path, name, nil, handler, middlewares)
}
