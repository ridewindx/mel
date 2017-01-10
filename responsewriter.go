package mel

import (
    "net/http"
    "net"
    "bufio"
    "io"
)

// ResponseWriter is a wrapper around http.ResponseWriter that
// provides extra information about the response.
type ResponseWriter interface {
    http.ResponseWriter
    http.Flusher
    http.Hijacker
    http.CloseNotifier

    // Status returns the http response status code,
    // or 0 if the response has not been written.
    Status() int

    // Written returns whether or not the response has been written.
    Written() bool

    // WriteString writes the string into the response body.
    WriteString(string) (int, error)

    // Size returns the number of bytes already written into the response http body.
    Size() int
}

type responseWriter struct {
    http.ResponseWriter
    status int
    size int
    written bool
}

var _ ResponseWriter = &responseWriter{}

func (w *responseWriter) reset() {
    w.ResponseWriter = nil
    w.status = 0
    w.size = 0
    w.written = false
}

func (w *responseWriter) WriteHeader(status int) {
	if w.Written() {
        // TODO: debugPrint("[WARNING] Headers were already written. Wanted to override status code %d with %d", w.status, status)
		return
    }

    if status == 0 {
        w.status = http.StatusOK // default http status
    } else {
        w.status = status
    }

    w.ResponseWriter.WriteHeader(w.status)
	w.written = true
}

func (w *responseWriter) Write(bytes []byte) (int, error) {
    if !w.Written() {
        w.WriteHeader(w.status)
    }
    size, err := w.ResponseWriter.Write(bytes)
    w.size += size
    return size, err
}

func (w *responseWriter) WriteString(s string) (int, error) {
    if !w.Written() {
        w.WriteHeader(http.StatusOK)
    }

	size, err := io.WriteString(w.ResponseWriter, s)
    w.size += size
    return size, err
}

func (w *responseWriter) Status() int {
    return w.status
}

func (w *responseWriter) Size() int {
    return w.size
}

func (w *responseWriter) Written() bool {
    return w.written
}

// Hijack implements the http.Hijacker interface
func (w *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
    return w.ResponseWriter.(http.Hijacker).Hijack()
}

// CloseNotify implements the http.CloseNotify interface
func (w *responseWriter) CloseNotify() <-chan bool {
    return w.ResponseWriter.(http.CloseNotifier).CloseNotify()
}

// Flush implements the http.Flush interface
func (w *responseWriter) Flush() {
    flusher, ok := w.ResponseWriter.(http.Flusher)
    if ok {
        flusher.Flush()
    }
}
