package render

import (
	"net/http"

	"gopkg.in/yaml.v2"
)

const yamlContentType = "application/x-yaml; charset=utf-8"

func WriteYAML(w http.ResponseWriter, obj interface{}) error {
	writeContentType(w, yamlContentType)

	bytes, err := yaml.Marshal(obj)
	if err != nil {
		return err
	}

	_, err = w.Write(bytes)
	return err
}

func (r *Renderer) YAML(obj interface{}) error {
	return WriteYAML(r.writer, obj)
}
