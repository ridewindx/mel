package render

import "net/http"

type Data struct {
	ContentType string
	Data        []byte
}

var _ Render = Data{}

func (r Data) Render(w http.ResponseWriter) error {
	if len(r.ContentType) > 0 {
		writeContentType(w, r.ContentType)
	}
	_, err := w.Write(r.Data)
	return err
}
