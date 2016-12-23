package mel

import "net/http"

type Handler interface {
    Handle(*Context)
}

type Context struct {
    req *http.Request
    ResponseWriter
}
