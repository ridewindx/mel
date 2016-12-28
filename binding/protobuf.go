package binding

import (
	"io/ioutil"
	"net/http"

	"github.com/golang/protobuf/proto"
)

type protobufBinding struct{}

func (protobufBinding) Name() string {
	return "protobuf"
}

func (protobufBinding) Bind(req *http.Request, obj interface{}) error {
	buf, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return err
	}

	if err = proto.Unmarshal(buf, obj.(proto.Message)); err != nil {
		return err
	}

	// Here it's same to return validate(obj), but util now we can't add `binding:""` to the struct
	// which automatically generate by gen-proto
	return nil
	// return validate(obj)
}
