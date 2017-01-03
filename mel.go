package mel

import (
	"html/template"
	"net"
	"net/http"
	"os"
	"sync"
)

type Handler func(*Context)

type Mel struct {
	*router
	handlers []Handler
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

	mel.handleHTTPRequest(c)

	mel.pool.Put(c)
}

func (mel *Mel) handleHTTPRequest(ctx *Context) {
	httpMethod := ctx.Request.Method
	path := ctx.Request.URL.Path

	// Find root of the tree for the given HTTP method
	t := mel.trees
	for i, tl := 0, len(t); i < tl; i++ {
		if t[i].method == httpMethod {
			root := t[i].root
			// Find route in tree
			handlers, params, tsr := root.getValue(path, ctx.Params)
			if handlers != nil {
				ctx.handlers = handlers
				ctx.Params = params
				ctx.Next()
				ctx.writermem.WriteHeaderNow()
				return

			} else if httpMethod != "CONNECT" && path != "/" {
				if tsr && engine.RedirectTrailingSlash {
					redirectTrailingSlash(ctx)
					return
				}
				if engine.RedirectFixedPath && redirectFixedPath(ctx, root, engine.RedirectFixedPath) {
					return
				}
			}
			break
		}
	}

	// TODO: unit test
	if engine.HandleMethodNotAllowed {
		for _, tree := range engine.trees {
			if tree.method != httpMethod {
				if handlers, _, _ := tree.root.getValue(path, nil); handlers != nil {
					ctx.handlers = engine.allNoMethod
					serveError(ctx, 405, default405Body)
					return
				}
			}
		}
	}
	ctx.handlers = engine.allNoRoute
	serveError(ctx, 404, default404Body)
}

