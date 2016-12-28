package render

import (
	"encoding/json"
	"net/http"
)

type (
	JSON struct {
		Data interface{}
	}

	IndentedJSON struct {
		Data interface{}
	}
)

var jsonContentType = []string{"application/json; charset=utf-8"}

func (r JSON) Render(w http.ResponseWriter) error {
	return WriteJSON(w, r.Data)
}

func (r IndentedJSON) Render(w http.ResponseWriter) error {
	writeContentType(w, jsonContentType)
	jsonBytes, err := json.MarshalIndent(r.Data, "", "    ")
	if err != nil {
		return err
	}
	w.Write(jsonBytes)
	return nil
}

func WriteJSON(w http.ResponseWriter, obj interface{}) error {
	writeContentType(w, jsonContentType)
	return json.NewEncoder(w).Encode(obj)
}
