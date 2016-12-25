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

    Params Params
    handlers []Handler
    index int8

    mel *Mel
    Keys map[string]interface{}

}

const abortIndex int8 = math.MaxInt8 / 2

func (c *Context) File(filePath string) {
    http.ServeFile(c.Writer, c.Request, filePath)
}
