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
	*router
	pool     sync.Pool

	RedirectTrailingSlash bool
	RedirectFixedPath bool
	HandleMethodNotAllowed bool
	ForwardedByClientIP    bool

	Template *template.Template
}

func New() *Mel {
	debugPrintWARNINGNew()
	mel := &Mel{
		router: NewRouter(),
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

	route, params := mel.router.Match(httpMethod, path)
	if route != nil {
		route.execute(ctx)
		ctx.Params = params
		ctx.Next()
		return
	} else if httpMethod != "CONNECT" && path != "/" {
		if mel.RedirectTrailingSlash {
			return
		}
		if mel.RedirectFixedPath {
			return
		}
	}

	if mel.HandleMethodNotAllowed {

	}
}

func serveError(c *Context, code int, message []byte) {
	c.Next()

	if !c.Writer.Written() {
		c.Writer.Header().Set("Content-Type", binding.MIMEPlain)
		c.Writer.WriteHeader(code)
		c.Writer.Write(message)
	}
}

func redirectTrailingSlash(c *Context) {
	req := c.Request
	path := req.URL.Path
	code := 301 // Permanent redirect, request with GET method
	if req.Method != "GET" {
		code = 307
	}

	if len(path) > 1 && path[len(path)-1] == '/' {
		req.URL.Path = path[:len(path)-1]
	} else {
		req.URL.Path = path + "/"
	}
	debugPrint("redirecting request %d: %s --> %s", code, path, req.URL.String())
	http.Redirect(c.Writer, req, req.URL.String(), code)
}
