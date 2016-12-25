package mel

import (
    "net/http"
    "math"
)

const abortIndex int8 = math.MaxInt8 / 2

type Context struct {
    Request *http.Request
    Writer ResponseWriter

    Params Params
    handlers []Handler
    index int8

    mel *Mel
    Keys map[string]interface{}

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

func (c *Context) File(filePath string) {
    http.ServeFile(c.Writer, c.Request, filePath)
}
