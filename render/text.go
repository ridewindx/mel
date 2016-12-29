package render

import (
	"fmt"
	"net/http"
)

type String struct {
	Format string
	Data   []interface{}
}

var _ Render = String{}

const plainContentType = "text/plain; charset=utf-8"

func (r String) Render(w http.ResponseWriter) error {
	return WriteString(w, r.Format, r.Data)
}

func WriteString(w http.ResponseWriter, format string, data []interface{}) error {
	writeContentType(w, plainContentType)

	_, err := fmt.Fprintf(w, format, data...)
	return err
}
