package mel

import (
    "net/http"
    "math"
)

type Handler interface {
    Handle(*Context)
}

type Context struct {
    Request *http.Request
    Writer ResponseWriter
}

const abortIndex int8 = math.MaxInt8 / 2

func (c *Context) File(filePath string) {
    http.ServeFile(c.Writer, c.Request, filePath)
}
