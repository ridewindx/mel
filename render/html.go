package render

import (
	"html/template"
	"net/http"
)

const htmlContentType = "text/html; charset=utf-8"

func WriteHTML(w http.ResponseWriter, template *template.Template, name string, obj interface{}) error {
	writeContentType(w, htmlContentType)
	if len(name) == 0 {
		return template.Execute(w, obj)
	}
	return template.ExecuteTemplate(w, name, obj)
}

func (r *Renderer) HTML(template *template.Template, name string, obj interface{}) error {
	return WriteHTML(r.writer, template, name, obj)
}
