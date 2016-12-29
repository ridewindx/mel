package render

import "net/http"

type Render interface {
	Render(http.ResponseWriter) error
}

func writeContentType(w http.ResponseWriter, values ...string) {
	header := w.Header()
	if len(header["Content-Type"]) == 0 {
		header["Content-Type"] = values
	}
}
