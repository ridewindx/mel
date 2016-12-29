package render

import (
	"html/template"
	"net/http"
)

type (
	HTMLRender interface {
		Instance(string, interface{}) Render
	}

	HTMLProduction struct {
		Template *template.Template
	}

	HTMLDebug struct {
		Files []string
		Glob  string
	}

	HTML struct {
		Template *template.Template
		Name     string
		Data     interface{}
	}
)

var _ HTMLRender = HTMLDebug{}
var _ HTMLRender = HTMLProduction{}
var _ Render = HTML{}

const htmlContentType = "text/html; charset=utf-8"

func (r HTMLProduction) Instance(name string, data interface{}) Render {
	return HTML{
		Template: r.Template,
		Name:     name,
		Data:     data,
	}
}

func (r HTMLDebug) Instance(name string, data interface{}) Render {
	return HTML{
		Template: r.loadTemplate(),
		Name:     name,
		Data:     data,
	}
}

func (r HTMLDebug) loadTemplate() *template.Template {
	if len(r.Files) > 0 {
		return template.Must(template.ParseFiles(r.Files...))
	}
	if len(r.Glob) > 0 {
		return template.Must(template.ParseGlob(r.Glob))
	}
	panic("the HTML debug render was created without files or glob pattern")
}

func (r HTML) Render(w http.ResponseWriter) error {
	writeContentType(w, htmlContentType)
	if len(r.Name) == 0 {
		return r.Template.Execute(w, r.Data)
	}
	return r.Template.ExecuteTemplate(w, r.Name, r.Data)
}