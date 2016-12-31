package mel

import (
	"html/template"
	"github.com/ridewindx/mel/render"
)

type Handler func(*Context)

type Mel struct {
	Router
	handlers []Handler

	ForwardedByClientIP    bool

	Template *template.Template
}

func (mel *Mel) SetTemplate(template *template.Template) {
	mel.Template = template
}

func (mel *Mel) LoadTemplateGlob(pattern string) {
	mel.SetTemplate(template.Must(template.ParseGlob(pattern)))
}

func (mel *Mel) LoadTemplates(files ...string) {
	mel.SetTemplate(template.Must(template.ParseFiles(files...)))
}
