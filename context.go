package mel

import (
    "net/http"
    "math"
    "strings"
    "net"
    "net/url"
    "github.com/ridewindx/mel/render"
    "github.com/ridewindx/mel/binding"
    "fmt"
    "github.com/manucorporat/sse"
    "io"
    "time"
	"sync"
)

const preStartIndex int8 = -1
const abortIndex int8 = math.MaxInt8 / 2

type pool struct {
    sync.Pool
}

func (p *pool) Get() *Context {
    c := p.Pool.Get().(*Context)
    return c
}

func (p *pool) Put(c *Context) {
    c.Request = nil
    c.Writer.Reset(nil)
	c.Params = nil
    c.handlers = nil
    c.index = preStartIndex
    c.Keys = nil
    c.Errors = nil

    p.Pool.Put(c)
}

func newPool(mel *Mel) *pool {
    var p pool
    p.Pool.New = func() interface{} {
        c := newContext()
        c.Mel = mel
        return c
    }
    return &p
}

type Context struct {
    Request  *http.Request
    Writer   ResponseWriter

    Params
    handlers []Handler
    index    int8

    Keys     map[string]interface{}
    Errors

    *Mel
}

func newContext() *Context {
    return &Context{
        Writer: &responseWriter{},
        index: preStartIndex,
    }
}

func (c *Context) reset(w http.ResponseWriter, req *http.Request) {
    c.Writer.Reset(w)
    c.Request = req
}

// Next executes the pending handlers in the chain inside the calling handler.
// It should be used only inside middleware.
func (c *Context) Next() {
    c.index++
    s := int8(len(c.handlers))
    for ; c.index < s; c.index++ {
        c.handlers[c.index](c)
    }
}

// IsAborted returns true if the current context was aborted.
func (c *Context) IsAborted() bool {
    return c.index >= abortIndex
}

// Abort prevents pending handlers from being called.
// Note that this will not stop the current handler.
// Let's say you have an authorization middleware that
// validates that the current request is authorized.
// If the authorization fails, call Abort to ensure
// the remaining handlers for this request are not called.
func (c *Context) Abort() {
    c.index = abortIndex
}

// AbortWithStatus calls `Abort()` and writes the headers with the specified status code.
// For example, a failed attempt to authentificate a request could use: context.AbortWithStatus(401).
func (c *Context) AbortWithStatus(status int) {
    c.Writer.WriteHeader(status)
    c.Abort()
}

// AbortWithError calls `AbortWithStatus()` and `Error()` internally. This method stops the chain, writes the status code and
// pushes the specified error to `c.Errors`.
// See Context.Error() for more details.
func (c *Context) AbortWithError(code int, err error) *Error {
    c.AbortWithStatus(code)
    return c.Error(err)
}

// Error adds an error to the context.
// It's a good idea to call Error for each error that occurred during the resolution of a request.
// A middleware can be used to collect all the errors
// and push them to a database together, print a log, or append it in the HTTP response.
func (c *Context) Error(err error) *Error {
    var e *Error
    switch err.(type) {
    case *Error:
        e = err.(*Error)
    default:
        e = &Error{
            Err: err,
            Type: ErrorTypePrivate,
        }
    }
    c.Errors = append(c.Errors, e)
    return e
}

// Set stores a new key/value pair exclusivelly for this context.
// It also lazy initializes c.Keys if it was not used previously.
func (c *Context) Set(key string, value interface{}) {
    if c.Keys == nil {
        c.Keys = make(map[string]interface{})
    }
    c.Keys[key] = value
}

// Get returns the value for the given key, i.e., (value, true).
// If the value does not exists it returns (nil, false).
func (c *Context) Get(key string) (value interface{}, exists bool) {
    if c.Keys != nil {
        value, exists = c.Keys[key]
    }
    return
}

// MustGet returns the value for the given key if it exists, otherwise it panics.
func (c *Context) MustGet(key string) interface{} {
    if value, exists := c.Get(key); exists {
        return value
    }
    panic("Key \"" + key + "\" does not exist")
}

// Query returns the keyed url query value if it exists.
// If the key does not exist, it returns the defaultValue string,
// or an empty string (no defaultValue specified).
//     GET /?name=Manu&lastname=
//     c.Query("name") == "Manu"
//     c.Query("lastname") == ""
//     c.Query("id") == ""
// 	   c.Query("name", "unknown") == "Manu"
// 	   c.Query("lastname", "none") == ""
// 	   c.Query("id", "none") == "none"
func (c *Context) Query(key string, defaultValue ...string) string {
    if value, ok := c.GetQuery(key); ok {
        return value
    }
    if (len(defaultValue) > 0) {
        return defaultValue[0]
    }
    return ""
}

// GetQuery returns the keyed url query value.
// If it exists, it returns `(value, true)` (even when the value is an empty string),
// othewise it returns `("", false)`.
//     GET /?name=Manu&lastname=
//     ("Manu", true) == c.GetQuery("name")
// 	   ("", false) == c.GetQuery("id")
// 	   ("", true) == c.GetQuery("lastname")
func (c *Context) GetQuery(key string) (string, bool) {
    if values, ok := c.Request.URL.Query()[key]; ok && len(values) > 0 {
        return values[0], true
    }
    return "", false
}

// Querys returns a slice of strings for a given query key.
// The length of the slice depends on the number of params with the given key.
func (c *Context) Querys(key string) []string {
    values, _ := c.GetQuerys(key)
    return values
}

// GetQuerys returns a slice of strings for a given query key, plus
// a boolean value whether at least one value exists for the given key.
func (c *Context) GetQuerys(key string) ([]string, bool) {
    if values, ok := c.Request.URL.Query()[key]; ok && len(values) > 0 {
        return values, true
    }
    return []string{}, false
}

func (c *Context) PostForm(key string, defaultValue ...string) string {
    if value, ok := c.GetPostForm(key); ok {
        return value
    }
    if (len(defaultValue) > 0) {
        return defaultValue[0]
    }
    return ""
}

func (c *Context) GetPostForm(key string) (string, bool) {
    if values, ok := c.GetPostForms(key); ok {
        return values[0], ok
    }
    return "", false
}

// PostForms returns a slice of strings for a given form key.
func (c *Context) PostForms(key string) []string {
    values, _ := c.GetPostForms(key)
    return values
}

// GetPostForms returns a slice of strings for a given form key, plus
// a boolean value whether at least one value exists for the given key.
func (c *Context) GetPostForms(key string) ([]string, bool) {
    req := c.Request
    req.ParseMultipartForm(32 << 20) // 32 MB

    if values := req.PostForm[key]; len(values) > 0 {
        return values, true
    }

    if req.MultipartForm != nil && req.MultipartForm.File != nil {
        if values := req.MultipartForm.Value[key]; len(values) > 0 {
            return values, true
        }
    }

    return []string{}, false
}

// Bind checks the Content-Type to select a binding engine automatically,
// Depending the "Content-Type" header different bindings are used:
// 		"application/json" --> JSON binding
// 		"application/xml"  --> XML binding
// otherwise --> returns an error
// It parses the request's body as JSON if Content-Type == "application/json" using JSON or XML as a JSON input.
// It decodes the json payload into the struct specified as a pointer.
// Like ParseBody() but this method also writes a 400 error if the json is not valid.
func (c *Context) Bind(obj interface{}) error {
    b := binding.Default(c.Request.Method, c.ContentType())
    return c.BindWith(obj, b)
}

// BindJSON is a shortcut for c.BindWith(obj, binding.JSON)
func (c *Context) BindJSON(obj interface{}) error {
    return c.BindWith(obj, binding.JSON)
}

// BindWith binds the passed struct pointer using the specified binding engine.
// See the binding package.
func (c *Context) BindWith(obj interface{}, b binding.Binding) error {
    if err := b.Bind(c.Request, obj); err != nil {
        c.AbortWithError(400, err).Type = ErrorTypeBind
        return err
    }
    return nil
}

// ClientIP implements a best effort algorithm to return the real client IP.
// It parses X-Real-IP and X-Forwarded-For in order to work properly with
// reverse-proxies such as nginx or haproxy.
func (c *Context) ClientIP() string {
    if c.Mel.ForwardedByClientIP {
        clientIP := strings.TrimSpace(c.requestHeader("X-Real-Ip"))
        if len(clientIP) > 0 {
            return clientIP
        }
        clientIP = c.requestHeader("X-Forwarded-For")
        if index := strings.IndexByte(clientIP, ','); index >= 0 {
            clientIP = clientIP[0:index]
        }
        clientIP = strings.TrimSpace(clientIP)
        if len(clientIP) > 0 {
            return clientIP
        }
    }
    if ip, _, err := net.SplitHostPort(strings.TrimSpace(c.Request.RemoteAddr)); err == nil {
        return ip
    }
    return ""
}

// ContentType returns the Content-Type header of the request.
func (c *Context) ContentType() string {
    return filterFlags(c.requestHeader("Content-Type"))
}

func (c *Context) requestHeader(key string) string {
    if values, _ := c.Request.Header[key]; len(values) > 0 {
        return values[0]
    }
    return ""
}

func (c *Context) Status(status int) {
    c.Writer.Status(status)
}

// Header sets the value associated with key in the header.
// If value == "", it removes the key in the header.
func (c *Context) Header(key, value string) {
    if len(value) > 0 {
        c.Writer.Header().Set(key, value)
    } else {
        c.Writer.Header().Del(key)
    }
}

// Cookie represents an HTTP cookie as sent in the Set-Cookie header of an
// HTTP response or the Cookie header of an HTTP request.
// Fields are a subset of http.Cookie fields.
type Cookie struct {
    Name string
    Value string
    Path string
    Domain string
    MaxAge int
    Secure bool
    HttpOnly bool
}

func (c *Context) SetCookie(cookie *Cookie) {
    if len(cookie.Path) == 0 {
        cookie.Path = "/"
    }
    http.SetCookie(c.Writer, &http.Cookie{
        Name: cookie.Name,
        Value: url.QueryEscape(cookie.Value),
        Path: cookie.Path,
        Domain: cookie.Domain,
        MaxAge: cookie.MaxAge,
        Secure: cookie.Secure,
        HttpOnly: cookie.HttpOnly,
    })
}

func (c *Context) Cookie(name string) (string, error) {
    cookie, err := c.Request.Cookie(name)
    if err != nil {
        return "", err
    }
    return url.QueryUnescape(cookie.Value)
}

func (c *Context) renderer() *render.Renderer {
    return render.New(c.Writer)
}

func (c *Context) Data(status int, contentType string, data []byte) error {
    c.Status(status)
    return c.renderer().Data(contentType, data)
}

func (c *Context) Text(status int, format string, data ...interface{}) error {
    c.Status(status)
    return c.renderer().Text(format, data...)
}

func (c *Context) HTML(status int, name string, obj interface{}) error {
    c.Status(status)
    return c.renderer().HTML(c.Mel.Template, name, obj)
}

func (c *Context) JSON(status int, obj interface{}, indented ...bool) error {
    c.Status(status)
    return c.renderer().JSON(obj, indented...)
}

func (c *Context) XML(status int, obj interface{}) error {
    c.Status(status)
    return c.renderer().XML(obj)
}

func (c *Context) YAML(status int, obj interface{}) error {
    c.Status(status)
    return c.renderer().YAML(obj)
}

// Redirect returns a HTTP redirect to the specific location.
func (c *Context) Redirect(code int, location string) {
    if (code < 300 || code > 308) && code != 201 {
        panic(fmt.Sprintf("Cannot redirect with status code %d", code))
    }
    http.Redirect(c.Writer, c.Request, location, code)
}

func (c *Context) File(filePath string) {
    http.ServeFile(c.Writer, c.Request, filePath)
}

// SSE writes a Server-Sent Event into the body stream.
func (c *Context) SSE(name string, message interface{}) error {
	event := sse.Event{
        Event: name,
        Data:  message,
    }
    return event.Render(c.Writer)
}

func (c *Context) Stream(step func(w io.Writer) bool) {
    w := c.Writer
    clientGone := w.CloseNotify()
    for {
        select {
        case <-clientGone:
            return
        default:
            keepOpen := step(w)
            w.Flush()
            if !keepOpen {
                return
            }
        }
    }
}

func (c *Context) Deadline() (deadline time.Time, ok bool) {
    return
}

func (c *Context) Done() <-chan struct{} {
    return nil
}

func (c *Context) Err() error {
    return nil
}

func (c *Context) Value(key interface{}) interface{} {
    if key == 0 {
        return c.Request
    }
    if keyAsString, ok := key.(string); ok {
        val, _ := c.Get(keyAsString)
        return val
    }
    return nil
}
