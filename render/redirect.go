package render

import (
	"fmt"
	"net/http"
)

type Redirect struct {
	Code     int
	Request  *http.Request
	Location string
}

var _ Render = Redirect{}

func (r Redirect) Render(w http.ResponseWriter) error {
	if (r.Code < 300 || r.Code > 308) && r.Code != 201 {
		panic(fmt.Sprintf("Cannot redirect with status code %d", r.Code))
	}
	http.Redirect(w, r.Request, r.Location, r.Code)
	return nil
}
