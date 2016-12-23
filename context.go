package mel

import (
    "net/http"
    "math"
)

type Handler interface {
    Handle(*Context)
}

type Context struct {
    req *http.Request
    ResponseWriter
}

const abortIndex int8 = math.MaxInt8 / 2
