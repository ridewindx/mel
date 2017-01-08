package mel

import (
	"html/template"
	"net"
	"net/http"
	"os"
	"sync"
	"github.com/gin-gonic/gin/binding"
)

var default404Body = []byte("404 page not found")
var default405Body = []byte("405 method not allowed")

type Handler func(*Context)

type Mel struct {
	*Router
	pool     sync.Pool

	allNoRoute  []Handler
	allNoMethod []Handler
	noRoute     []Handler
	noMethod    []Handler

	RedirectTrailingSlash bool
	RedirectFixedPath bool
	HandleMethodNotAllowed bool
	ForwardedByClientIP    bool

	Template *template.Template
}

func New() *Mel {
	debugPrintWARNINGNew()
	mel := &Mel{
		Router: NewRouter(),
		RedirectTrailingSlash:  true,
		RedirectFixedPath:      false,
		HandleMethodNotAllowed: false,
		ForwardedByClientIP:    true,
	}

	mel.pool.New = func() interface{} {
		return &Context{mel: mel}
	}

	return mel
}

func (mel *Mel) SetTemplate(template *template.Template) {
	mel.Template = template
}

func (mel *Mel) LoadTemplateGlob(pattern string) {
	mel.SetTemplate(template.Must(template.ParseGlob(pattern)))
}

func (mel *Mel) LoadTemplates(files ...string) {
	mel.SetTemplate(template.Must(template.ParseFiles(files...)))
}

// NoRoute sets handlers for requests that match no route.
// It return a 404 code by default.
func (mel *Mel) NoRoute(handlers ...Handler) {
	mel.noRoute = handlers
	mel.rebuildNoRouteHandlers()
}

// NoMethod sets handlers for requests that match a route for another HTTP method.
// It return a 405 code by default.
func (mel *Mel) NoMethod(handlers ...Handler) {
	mel.noMethod = handlers
	mel.rebuildNoMethodHandlers()
}

// Use attachs global middlewares to the app. The middlewares attached though Use() will be
// included in the handlers chain for every single request. Even 404, 405, static files...
// For example, this is the right place for a logger or error management middleware.
func (mel *Mel) Use(middleware ...Handler) {
	mel.Router.Use(middleware...)
	mel.rebuildNoRouteHandlers()
	mel.rebuildNoMethodHandlers()
}

func (mel *Mel) rebuildNoRouteHandlers() {
	mel.allNoRoute = mel.combineHandlers(mel.noRoute)
}

func (mel *Mel) rebuildNoMethodHandlers() {
	mel.allNoMethod = mel.combineHandlers(mel.noMethod)
}

// Run attaches `mel` to a http.Server and starts listening and serving HTTP requests.
// Note: this method will block the calling goroutine indefinitely unless an error happens.
func (mel *Mel) Run(addrs ...string) (err error) {
	addr := resolveAddress(addrs)

	debugPrint("Listening and serving HTTP on %s\n", addr)
	err = http.ListenAndServe(addr, mel)
	debugPrintError(err)

	return
}

// RunTLS attaches `mel` to a http.Server and starts listening and serving HTTPS (secure) requests.
// Note: this method will block the calling goroutine indefinitely unless an error happens.
func (mel *Mel) RunTLS(addr string, certFile string, keyFile string) (err error) {
	debugPrint("Listening and serving HTTPS on %s\n", addr)
	err = http.ListenAndServeTLS(addr, certFile, keyFile, mel)
	debugPrintError(err)

	return
}

// RunUnix attaches `mel` to a http.Server and starts listening and serving HTTP requests
// through the specified unix socket (ie. a file).
// Note: this method will block the calling goroutine indefinitely unless an error happens.
func (mel *Mel) RunUnix(file string) (err error) {
	debugPrint("Listening and serving HTTP on unix:/%s", file)
	defer func() { debugPrintError(err) }()

	os.Remove(file)
	listener, err := net.Listen("unix", file)
	if err != nil {
		return
	}
	defer listener.Close()
	err = http.Serve(listener, mel)

	return
}

// ServerHTTP implements the http.Handler interface.
func (mel *Mel) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	c := mel.pool.Get().(*Context)
	c.init(w, req)

	mel.handle(c)

	mel.pool.Put(c)
}

func (mel *Mel) handle(ctx *Context) {
	httpMethod := ctx.Request.Method
	path := ctx.Request.URL.Path

	route, params, tsr := mel.Router.Match(httpMethod, path)
	if route != nil {
		route.execute(ctx)
		ctx.Params = params
		ctx.Next()
		return
	} else if httpMethod != "CONNECT" && path != "/" {
		if tsr && mel.RedirectTrailingSlash {
			redirectTrailingSlash(ctx)
			return
		}
		if mel.RedirectFixedPath && redirectFixedPath(ctx, mel.Router, mel.RedirectTrailingSlash) {
			return
		}
	}

	if mel.HandleMethodNotAllowed {
		for method := range mel.Router.trees {
			if method != httpMethod {
				route, _, _ := mel.Router.Match(method, path)
				if route != nil {
					ctx.handlers = mel.allNoMethod
					serveError(ctx, 405, default405Body)
					return
				}
			}
		}
	}

	ctx.handlers = mel.allNoRoute
	serveError(ctx, 404, default404Body)
}

func serveError(c *Context, code int, defaultMessage []byte) {
	c.Next()

	if !c.Writer.Written() {
		c.Writer.Header().Set("Content-Type", binding.MIMEPlain)
		c.Writer.WriteHeader(code)
		c.Writer.Write(defaultMessage)
	}
}

func redirectTrailingSlash(c *Context) {
	req := c.Request
	path := req.URL.Path
	code := 301 // Permanent redirect, request with GET method
	if req.Method != "GET" {
		code = 307
	}

	assert(len(path) > 1 && path[len(path)-1] == '/', "Path has no trailing slash")
	req.URL.Path = path[:len(path)-1]
	debugPrint("redirecting request %d: %s --> %s", code, path, req.URL.String())
	http.Redirect(c.Writer, req, req.URL.String(), code)
}

func redirectFixedPath(ctx *Context, router *Router, trailingSlash bool) bool {
	req := ctx.Request
	httpMethod := req.Method
	path := req.URL.Path

	fixedPath := cleanPath(path)
	route, _, tsr := router.Match(httpMethod, fixedPath)
	if route == nil {
		if !(tsr && trailingSlash) {
			return false
		}

		fixedPath = fixedPath[:len(fixedPath) - 1]
	}

	code := 301 // Permanent redirect, request with GET method
	if req.Method != "GET" {
		code = 307
	}
	req.URL.Path = string(fixedPath)
	debugPrint("redirecting request %d: %s --> %s", code, path, req.URL.String())
	http.Redirect(ctx.Writer, req, req.URL.String(), code)
	return true
}
