package render

import (
	"encoding/xml"
	"net/http"
)

type XML struct {
	Data interface{}
}

var Render = XML{}

const xmlContentType = "application/xml; charset=utf-8"

func (r XML) Render(w http.ResponseWriter) error {
	writeContentType(w, xmlContentType)
	return xml.NewEncoder(w).Encode(r.Data)
}
