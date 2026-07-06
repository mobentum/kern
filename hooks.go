package kern

// RouteInfo contains route metadata for lifecycle hooks.
type RouteInfo struct {
	Method string
	Path   string
	Name   string
}

// ListenInfo contains listen metadata for lifecycle hooks.
type ListenInfo struct {
	Addr string
	TLS  bool
}

// RouteHook is triggered when a route is registered.
type RouteHook func(RouteInfo)

// ListenHook is triggered before a server starts listening.
type ListenHook func(ListenInfo)

// ShutdownHook is triggered when graceful shutdown exits.
type ShutdownHook func(error)

// ErrorHook is triggered when app-level server errors are observed.
type ErrorHook func(error)

// OnRoute registers a callback for route registration events.
func (app *App) OnRoute(hook RouteHook) {
	if hook != nil {
		app.onRouteHooks = append(app.onRouteHooks, hook)
	}
}

// OnListen registers a callback for listen events.
func (app *App) OnListen(hook ListenHook) {
	if hook != nil {
		app.onListenHooks = append(app.onListenHooks, hook)
	}
}

// OnShutdown registers a callback for graceful shutdown events.
func (app *App) OnShutdown(hook ShutdownHook) {
	if hook != nil {
		app.onShutdownHooks = append(app.onShutdownHooks, hook)
	}
}

// OnError registers a callback for app-level server errors.
func (app *App) OnError(hook ErrorHook) {
	if hook != nil {
		app.onErrorHooks = append(app.onErrorHooks, hook)
	}
}

func (app *App) emitRoute(info RouteInfo) {
	for _, hook := range app.onRouteHooks {
		hook(info)
	}
}

func (app *App) emitListen(info ListenInfo) {
	for _, hook := range app.onListenHooks {
		hook(info)
	}
}

func (app *App) emitShutdown(err error) {
	for _, hook := range app.onShutdownHooks {
		hook(err)
	}
}

func (app *App) emitError(err error) {
	for _, hook := range app.onErrorHooks {
		hook(err)
	}
}
