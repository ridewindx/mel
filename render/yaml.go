package render

import (
	"net/http"

	"gopkg.in/yaml.v2"
)

type YAML struct {
	Data interface{}
}

var yamlContentType = []string{"application/x-yaml; charset=utf-8"}

func (r YAML) Render(w http.ResponseWriter) error {
	writeContentType(w, yamlContentType)

	bytes, err := yaml.Marshal(r.Data)
	if err != nil {
		return err
	}

	w.Write(bytes)
	return nil
}
