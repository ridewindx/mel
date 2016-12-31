package render

import (
	"fmt"
	"net/http"
)

const plainContentType = "text/plain; charset=utf-8"

func WriteText(w http.ResponseWriter, format string, data ...interface{}) error {
	writeContentType(w, plainContentType)

	_, err := fmt.Fprintf(w, format, data...)
	return err
}

func (r *Renderer) Text(format string, data ...interface{}) error {
	return WriteText(r.writer, format, data...)
}
