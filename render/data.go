package render

import "net/http"

func WriteData(w http.ResponseWriter, contentType string, data []byte) error {
	if len(contentType) > 0 {
		writeContentType(w, contentType)
	}
	_, err := w.Write(data)
	return err
}

func (r *Renderer) Data(contentType string, data []byte) error {
	return WriteData(r.writer, contentType, data)
}
