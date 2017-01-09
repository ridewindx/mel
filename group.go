package mel

import (
	"regexp"
	"strings"
	"net/http"
	"path"
)

// routesGroup is used internally to configure router,
// a routesGroup is associated with a prefix and an array of handlers (middleware)
type RoutesGroup struct {
	BasePath string
	Handlers []Handler
	router   *Router
}

// Group creates a new router group.
// You should add all the routes that have common middlwares or the same path prefix.
func (group *RoutesGroup) Group(relativePath string, handlers ...Handler) *RoutesGroup {
	return &RoutesGroup{
		BasePath: joinPaths(group.BasePath, relativePath),
		Handlers: group.combineHandlers(handlers),
		router: group.router,
	}
}

// Use adds middleware to the group.
func (group *RoutesGroup) Use(middlewares ...Handler) {
	group.Handlers = append(group.Handlers, middlewares...)
}

func (group *RoutesGroup) handle(httpMethod, relativePath string, target interface{}, handlers []Handler) {
	absolutePath := joinPaths(group.BasePath, relativePath)
    handlers = group.combineHandlers(handlers)
	group.router.Register(httpMethod, absolutePath, target, handlers...)
}

// Handle registers a new request handle and middleware with the given path and method.
// The last handler should be the real handler, the other ones should be middleware
// that can and should be shared among different routes.
//
// This function is intended for bulk loading and to allow the usage of less
// frequently used, non-standardized or custom methods (e.g. for internal
// communication with a proxy).
func (group *RoutesGroup) Handle(httpMethod, relativePath string, target interface{}, handlers ...Handler) {
    if matches, err := regexp.MatchString("^[A-Z]+$", httpMethod); !matches || err != nil {
		panic("HTTP method " + httpMethod + " is invalid")
	}
	group.handle(httpMethod, relativePath, target, handlers)
}

func (group *RoutesGroup) Get(relativePath string, target interface{}, handlers ...Handler) {
	group.handle("GET", relativePath, target, handlers)
}

func (group *RoutesGroup) Post(relativePath string, target interface{}, handlers ...Handler) {
	group.handle("POST", relativePath, target, handlers)
}

func (group *RoutesGroup) Head(relativePath string, target interface{}, handlers ...Handler) {
	group.handle("HEAD", relativePath, target, handlers)
}

func (group *RoutesGroup) Delete(relativePath string, target interface{}, handlers ...Handler) {
	group.handle("DELETE", relativePath, target, handlers)
}

func (group *RoutesGroup) Put(relativePath string, target interface{}, handlers ...Handler) {
	group.handle("PUT", relativePath, target, handlers)
}

func (group *RoutesGroup) Options(relativePath string, target interface{}, handlers ...Handler) {
	group.handle("OPTIONS", relativePath, target, handlers)
}

func (group *RoutesGroup) Trace(relativePath string, target interface{}, handlers ...Handler) {
	group.handle("TRACE", relativePath, target, handlers)
}

func (group *RoutesGroup) Patch(relativePath string, target interface{}, handlers ...Handler) {
	group.handle("PATCH", relativePath, target, handlers)
}

// Any registers a route that matches all the HTTP methods.
func (group *RoutesGroup) Any(relativePath string, target interface{}, handlers ...Handler) {
	group.Get(relativePath, target, handlers...)
    group.Post(relativePath, target, handlers...)
	group.Head(relativePath, target, handlers...)
	group.Delete(relativePath, target, handlers...)
	group.Put(relativePath, target, handlers...)
	group.Options(relativePath, target, handlers...)
	group.Trace(relativePath, target, handlers...)
	group.Patch(relativePath, target, handlers...)
}

// StaticFile registers a single route in order to server a single file of the local filesystem.
// router.StaticFile("favicon.ico", "./resources/favicon.ico")
func (group *RoutesGroup) StaticFile(relativePath, filePath string) {
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
func (group *RoutesGroup) StaticDir(relativePath, root string) {
	if strings.Contains(relativePath, ":") || strings.Contains(relativePath, "*") {
		panic("URL parameters can not be used when serving a static folder")
	}

	fs := Dir(root, false)

	absolutePath := joinPaths(group.BasePath, relativePath)
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

func (group *RoutesGroup) combineHandlers(handlers []Handler) []Handler {
	finalSize := len(group.Handlers) + len(handlers)
	if finalSize >= int(abortIndex) {
		panic("too many handlers")
	}
	mergedHandlers := make([]Handler, finalSize)
	copy(mergedHandlers, group.Handlers)
	copy(mergedHandlers[len(group.Handlers):], handlers)
	return mergedHandlers
}
