package mel

import (
    "net/http"
    "math"
    "strings"
    "net"
)

const abortIndex int8 = math.MaxInt8 / 2

type Context struct {
    Request *http.Request
    Writer responseWriter

    Params Params
    handlers []Handler
    index int8

    mel *Mel
    Keys map[string]interface{}
    Errors errors
}

func NewContext() *Context {
    return &Context{
        index: -1,
    }
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

// ClientIP implements a best effort algorithm to return the real client IP.
// It parses X-Real-IP and X-Forwarded-For in order to work properly with
// reverse-proxies such as nginx or haproxy.
func (c *Context) ClientIP() string {
    if c.mel.ForwardedByClientIP {
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

func (c *Context) requestHeader(key string) string {
    if values, _ := c.Request.Header[key]; len(values) > 0 {
        return values[0]
    }
    return ""
}

func (c *Context) Status(status int) {
    c.Writer.status = status
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

func (c *Context) File(filePath string) {
    http.ServeFile(c.Writer, c.Request, filePath)
}
