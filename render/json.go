package render

import (
	"encoding/json"
	"net/http"
)

const jsonContentType = "application/json; charset=utf-8"

func WriteJSON(w http.ResponseWriter, obj interface{}) error {
	writeContentType(w, jsonContentType)
	return json.NewEncoder(w).Encode(obj)
}

func WriteIndentedJSON(w http.ResponseWriter, obj interface{}) error {
	writeContentType(w, jsonContentType)
	jsonBytes, err := json.MarshalIndent(obj, "", "    ")
	if err != nil {
		return err
	}
	_, err = w.Write(jsonBytes)
	return err
}

func (r *Renderer) JSON(obj interface{}, indented ...bool) error {
	if len(indented) > 0 && indented[0] {
		return WriteIndentedJSON(r.writer, obj)
	} else {
		return WriteJSON(r.writer, obj)
	}
}
