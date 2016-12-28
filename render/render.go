package render

import "net/http"

type Render interface {
	Render(http.ResponseWriter) error
}

var (
	_ Render     = JSON{}
	_ Render     = IndentedJSON{}
	_ Render     = XML{}
	_ Render     = String{}
	_ Render     = Redirect{}
	_ Render     = Data{}
	_ Render     = HTML{}
	_ HTMLRender = HTMLDebug{}
	_ HTMLRender = HTMLProduction{}
	_ Render     = YAML{}
)

func writeContentType(w http.ResponseWriter, values []string) {
	header := w.Header()
	if len(header["Content-Type"]) == 0 {
		header["Content-Type"] = values
	}
}
