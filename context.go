package mel

import (
    "net/http"
    "math"
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
// Let's say you have an authorization middleware that validates that the current request is authorized. If the
// authorization fails (ex: the password does not match), call Abort to ensure the remaining handlers
// for this request are not called.
func (c *Context) Abort() {
    c.index = abortIndex
}

// AbortWithStatus calls `Abort()` and writes the headers with the specified status code.
// For example, a failed attempt to authentificate a request could use: context.AbortWithStatus(401).
func (c *Context) AbortWithStatus(code int) {
    c.Status(code)
    c.Writer.WriteHeaderNow()
    c.Abort()
}

// AbortWithError calls `AbortWithStatus()` and `Error()` internally. This method stops the chain, writes the status code and
// pushes the specified error to `c.Errors`.
// See Context.Error() for more details.
func (c *Context) AbortWithError(code int, err error) *Error {
    c.AbortWithStatus(code)
    return c.Error(err)
}


func (c *Context) File(filePath string) {
    http.ServeFile(c.Writer, c.Request, filePath)
}
