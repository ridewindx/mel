package mel

import (
	"regexp"
	"strings"
)

type RouterGroup interface {
    Group(string, ...Handler) *RouterGroup

	Use(...Handler)

	Handle(string, interface{}, ...Handler)
	Get(string, interface{}, ...Handler)
	Post(string, interface{}, ...Handler)
	Head(string, interface{}, ...Handler)
	Delete(string, interface{}, ...Handler)
	Put(string, interface{}, ...Handler)
	Options(string, interface{}, ...Handler)
	Trace(string, interface{}, ...Handler)
	Patch(string, interface{}, ...Handler)
	Any(string, interface{}, ...Handler)
}

// routerGroup is used internally to configure router,
// a routerGroup is associated with a prefix and an array of handlers (middleware)
type routerGroup struct {
	basePath string
	handlers []Handler
	router *router
}

var _ RouterGroup = &routerGroup{}

// Group creates a new router group.
// You should add all the routes that have common middlwares or the same path prefix.
func (group *routerGroup) Group(relativePath string, handlers ...Handler) *RouterGroup {
	return &routerGroup{
		basePath: joinPaths(group.basePath, relativePath),
		handlers: group.combineHandlers(handlers),
		router: group.router,
	}
}

// Use adds middleware to the group.
func (group *routerGroup) Use(middleware ...Handler) {
	group.handlers = append(group.handlers, middleware...)
}

func (group *routerGroup) handle(httpMethod, relativePath string, target interface{}, handlers []Handler) {
	absolutePath := joinPaths(group.basePath, relativePath)
    handlers = group.combineHandlers(handlers)
	group.router.AddRoute(httpMethod, absolutePath, target, handlers...)
}

// Handle registers a new request handle and middleware with the given path and method.
// The last handler should be the real handler, the other ones should be middleware
// that can and should be shared among different routes.
//
// This function is intended for bulk loading and to allow the usage of less
// frequently used, non-standardized or custom methods (e.g. for internal
// communication with a proxy).
func (group *routerGroup) Handle(httpMethod, relativePath string, target interface{}, handlers ...Handler) {
    if matches, err := regexp.MatchString("^[A-Z]+$", httpMethod); !matches || err != nil {
		panic("HTTP method " + httpMethod + " is invalid")
	}
	group.handle(httpMethod, relativePath, target, handlers)
}

func (group *routerGroup) Get(relativePath string, target interface{}, handlers ...Handler) {
	group.handle("GET", relativePath, target, handlers)
}

func (group *routerGroup) Post(relativePath string, target interface{}, handlers ...Handler) {
	group.handle("POST", relativePath, target, handlers)
}

func (group *routerGroup) Head(relativePath string, target interface{}, handlers ...Handler) {
	group.handle("Head", relativePath, target, handlers)
}

func (group *routerGroup) Delete(relativePath string, target interface{}, handlers ...Handler) {
	group.handle("DELETE", relativePath, target, handlers)
}

func (group *routerGroup) Put(relativePath string, target interface{}, handlers ...Handler) {
	group.handle("PUT", relativePath, target, handlers)
}

func (group *routerGroup) Options(relativePath string, target interface{}, handlers ...Handler) {
	group.handle("OPTIONS", relativePath, target, handlers)
}

func (group *routerGroup) Trace(relativePath string, target interface{}, handlers ...Handler) {
	group.handle("TRACE", relativePath, target, handlers)
}

func (group *routerGroup) Patch(relativePath string, target interface{}, handlers ...Handler) {
	group.handle("PATCH", relativePath, target, handlers)
}

// Any registers a route that matches all the HTTP methods.
func (group *routerGroup) Any(relativePath string, target interface{}, handlers ...Handler) {
	group.Get(relativePath, target, handlers)
    group.Post(relativePath, target, handlers)
	group.Head(relativePath, target, handlers)
	group.Delete(relativePath, target, handlers)
	group.Put(relativePath, target, handlers)
	group.Options(relativePath, target, handlers)
	group.Trace(relativePath, target, handlers)
	group.Patch(relativePath, target, handlers)
}

// StaticFile registers a single route in order to server a single file of the local filesystem.
// router.StaticFile("favicon.ico", "./resources/favicon.ico")
func (group *routerGroup) StaticFile(relativePath, filePath string) {
	if strings.Contains(relativePath, ":") || strings.Contains(relativePath, "*") {
		panic("URL parameters can not be used when serving a static file")
	}

	handler := func(c *Context) {
		c.File(filePath)
	}

	group.Get(relativePath, handler)
	group.Head(relativePath, handler)
}

func (group *RouterGroup) StaticDir(relativePath, root string) {
	if strings.Contains(relativePath, ":") || strings.Contains(relativePath, "*") {
		panic("URL parameters can not be used when serving a static folder")
	}

	fs := Dir(root, false)

	absolutePath := joinPaths(group.)
}

func (group *routerGroup) combineHandlers(handlers []Handler) []Handler {
	finalSize := len(group.handlers) + len(handlers)
	if finalSize >= int(abortIndex) {
		panic("too many handlers")
	}
	mergedHandlers := make([]Handler, finalSize)
	copy(mergedHandlers, group.handlers)
	copy(mergedHandlers[len(group.handlers):], handlers)
	return mergedHandlers
}
