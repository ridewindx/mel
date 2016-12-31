package render

import (
	"net/http"
)

type Renderer struct {
	writer   http.ResponseWriter
}

func New(w http.ResponseWriter) *Renderer {
	return &Renderer{
		writer: w,
	}
}

func writeContentType(w http.ResponseWriter, values ...string) {
	header := w.Header()
	if len(header["Content-Type"]) == 0 {
		header["Content-Type"] = values
	}
}
