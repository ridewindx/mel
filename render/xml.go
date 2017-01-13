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

type XMLMap map[string]interface{}

// MarshalXML allows type H to be used with xml.Marshal
func (m XMLMap) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name = xml.Name{
		Space: "",
		Local: "map",
	}
	if err := e.EncodeToken(start); err != nil {
		return err
	}
	for key, value := range m {
		elem := xml.StartElement{
			Name: xml.Name{Space: "", Local: key},
			Attr: []xml.Attr{},
		}
		if err := e.EncodeElement(value, elem); err != nil {
			return err
		}
	}
	if err := e.EncodeToken(xml.EndElement{Name: start.Name}); err != nil {
		return err
	}
	return nil
}
