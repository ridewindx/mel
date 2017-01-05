package mel

import (
	"regexp"
	"strings"
	"net/http"
	"path"
)

type RoutesGroup interface {
    Group(string, ...Handler) *RoutesGroup

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

// routesGroup is used internally to configure router,
// a routesGroup is associated with a prefix and an array of handlers (middleware)
type routesGroup struct {
	basePath string
	handlers []Handler
	router *router
}

var _ RoutesGroup = &routesGroup{}

// Group creates a new router group.
// You should add all the routes that have common middlwares or the same path prefix.
func (group *routesGroup) Group(relativePath string, handlers ...Handler) *RoutesGroup {
	return &routesGroup{
		basePath: joinPaths(group.basePath, relativePath),
		handlers: group.combineHandlers(handlers),
		router: group.router,
	}
}

// Use adds middleware to the group.
func (group *routesGroup) Use(middlewares ...Handler) {
	group.handlers = append(group.handlers, middlewares...)
}

func (group *routesGroup) handle(httpMethod, relativePath string, handlers []interface{}) {
	if len(handlers) == 0 {
		panic("Routing target not found")
	}

	target := handlers[-1]
	middlewares := make([]Handler, 0, len(handlers)-1)
	for _, h := range handlers[:len(handlers)-1] {
		middlewares = append(middlewares, h.(Handler))
	}

	absolutePath := joinPaths(group.basePath, relativePath)
    middlewares = group.combineHandlers(middlewares)
	group.router.AddRoute(httpMethod, absolutePath, target, middlewares...)
}

// Handle registers a new request handle and middleware with the given path and method.
// The last handler should be the real handler, the other ones should be middleware
// that can and should be shared among different routes.
//
// This function is intended for bulk loading and to allow the usage of less
// frequently used, non-standardized or custom methods (e.g. for internal
// communication with a proxy).
func (group *routesGroup) Handle(httpMethod, relativePath string, handlers ...interface{}) {
    if matches, err := regexp.MatchString("^[A-Z]+$", httpMethod); !matches || err != nil {
		panic("HTTP method " + httpMethod + " is invalid")
	}
	group.handle(httpMethod, relativePath, handlers)
}

func (group *routesGroup) Get(relativePath string, handlers ...interface{}) {
	group.handle("GET", relativePath, handlers)
}

func (group *routesGroup) Post(relativePath string, handlers ...interface{}) {
	group.handle("POST", relativePath, handlers)
}

func (group *routesGroup) Head(relativePath string, handlers ...interface{}) {
	group.handle("Head", relativePath, handlers)
}

func (group *routesGroup) Delete(relativePath string, handlers ...interface{}) {
	group.handle("DELETE", relativePath, handlers)
}

func (group *routesGroup) Put(relativePath string, handlers ...interface{}) {
	group.handle("PUT", relativePath, handlers)
}

func (group *routesGroup) Options(relativePath string, handlers ...interface{}) {
	group.handle("OPTIONS", relativePath, handlers)
}

func (group *routesGroup) Trace(relativePath string, handlers ...interface{}) {
	group.handle("TRACE", relativePath, handlers)
}

func (group *routesGroup) Patch(relativePath string, handlers ...interface{}) {
	group.handle("PATCH", relativePath, handlers)
}

// Any registers a route that matches all the HTTP methods.
func (group *routesGroup) Any(relativePath string, handlers ...interface{}) {
	group.Get(relativePath, handlers)
    group.Post(relativePath, handlers)
	group.Head(relativePath, handlers)
	group.Delete(relativePath, handlers)
	group.Put(relativePath, handlers)
	group.Options(relativePath, handlers)
	group.Trace(relativePath, handlers)
	group.Patch(relativePath, handlers)
}

// StaticFile registers a single route in order to server a single file of the local filesystem.
// router.StaticFile("favicon.ico", "./resources/favicon.ico")
func (group *routesGroup) StaticFile(relativePath, filePath string) {
	if strings.Contains(relativePath, ":") || strings.Contains(relativePath, "*") {
		panic("URL parameters can not be used when serving a static file")
	}

	handler := func(c *Context) {
		c.File(filePath)
	}

	group.Get(relativePath, handler)
	group.Head(relativePath, handler)
}

// StaticDir serves files from the given file system root.
// router.StaticDir("/static", "/var/www")
func (group *routesGroup) StaticDir(relativePath, root string) {
	if strings.Contains(relativePath, ":") || strings.Contains(relativePath, "*") {
		panic("URL parameters can not be used when serving a static folder")
	}

	fs := Dir(root, false)

	absolutePath := joinPaths(group.basePath, relativePath)
	fileHandler := http.StripPrefix(absolutePath, http.FileServer(fs))
	_, nolisting := fs.(*onlyfilesFS)
	handler := func(c *Context) {
		if nolisting {
			c.Writer.WriteHeader(404)
		}
		fileHandler.ServeHTTP(c.Writer, c.Request)
	}

	pathPattern := path.Join(relativePath, "/*filepath")

	group.Get(pathPattern, handler)
	group.Head(pathPattern, handler)
}

func (group *routesGroup) combineHandlers(handlers []Handler) []Handler {
	finalSize := len(group.handlers) + len(handlers)
	if finalSize >= int(abortIndex) {
		panic("too many handlers")
	}
	mergedHandlers := make([]Handler, finalSize)
	copy(mergedHandlers, group.handlers)
	copy(mergedHandlers[len(group.handlers):], handlers)
	return mergedHandlers
}
