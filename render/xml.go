package render

import (
	"encoding/xml"
	"net/http"
)

const xmlContentType = "application/xml; charset=utf-8"

func WriteXML(w http.ResponseWriter, obj interface{}) error {
	writeContentType(w, xmlContentType)
	return xml.NewEncoder(w).Encode(obj)
}

func (r *Renderer) XML(obj interface{}) error {
	return WriteXML(r.writer, obj)
}
