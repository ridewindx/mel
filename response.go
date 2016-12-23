package mel

import (
    "net/http"
    "net"
    "bufio"
)

// ResponseWriter is a wrapper around http.ResponseWriter that
// provides extra information about the response.
type ResponseWriter interface {
    http.ResponseWriter
    http.Flusher
    http.Hijacker

    // Status returns the status code,
    // or 0 if the response has not been written.
    Status() int

    // Written returns whether or not the response has been written.
    Written() bool

    // Size returns the size of the response body.
    Size() int
}

type responseWriter struct {
    http.ResponseWriter
    status int
    size int
}

func (rw *responseWriter) WriteHeader(status int) {
    rw.status = status
    rw.ResponseWriter.WriteHeader(status)
}

func (rw *responseWriter) Write(bytes []byte) (int, error) {
    if !rw.Written() {
        rw.WriteHeader(http.StatusOK)
    }
    size, err := rw.ResponseWriter.Write(bytes)
    rw.size += size
    return size, err
}

func (rw *responseWriter) Status() int {
    return rw.status
}

func (rw *responseWriter) Size() int {
    return rw.size
}

func (rw *responseWriter) Written() bool {
    return rw.status != 0
}

func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
    return rw.ResponseWriter.(http.Hijacker).Hijack()
}

func (rw *responseWriter) CloseNotify() <-chan bool {
    return rw.ResponseWriter.(http.CloseNotifier).CloseNotify()
}

func (rw *responseWriter) Flush() {
    flusher, ok := rw.ResponseWriter.(http.Flusher)
    if ok {
        flusher.Flush()
    }
}
