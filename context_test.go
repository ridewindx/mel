package mel

import (
	"testing"
	"github.com/stretchr/testify/assert"

	"errors"
	"net/http"
	"net/http/httptest"
	"bytes"
	"mime/multipart"
	"github.com/ridewindx/mel/binding"
	"fmt"
)

func createMultipartRequest() *http.Request {
	boundary := "--testboundary"
	body := new(bytes.Buffer)
	mw := multipart.NewWriter(body)
	defer mw.Close()

	must(mw.SetBoundary(boundary))
	must(mw.WriteField("foo", "bar"))
	must(mw.WriteField("bar", "10"))
	must(mw.WriteField("bar", "foo2"))
	must(mw.WriteField("array", "first"))
	must(mw.WriteField("array", "second"))
	must(mw.WriteField("id", ""))
	req, err := http.NewRequest("POST", "/", body)
	must(err)
	req.Header.Set("Content-Type", binding.MIMEMultipartPOSTForm+"; boundary="+boundary)
	return req
}

func must(err error) {
	if err != nil {
		panic(err.Error())
	}
}

func TestContextReset(t *testing.T) {
	pool := newPool(nil)
	c1 := pool.Get()
	assert.Nil(t, c1.mel)

	c1.index = 2
	c1.Writer = &responseWriter{ResponseWriter: httptest.NewRecorder()}
	c1.Params = Params{Param{}}
	c1.Error(errors.New("test"))
	c1.Set("foo", "bar")
	pool.Put(c1)

	c2 := pool.Get()
	if (c1 != c2) {
		fmt.Println("Context from the pool is not the original one")
		return
	}

	assert.Nil(t, c2.mel)
	assert.Nil(t, c2.Request)
	assert.Len(t, c2.Params, 0)
	assert.Len(t, c2.handlers, 0)
	assert.EqualValues(t, c2.index, preStartIndex)
	assert.False(t, c2.IsAborted())
	assert.Nil(t, c2.Keys)
	assert.Len(t, c2.Errors, 0)
	assert.Empty(t, c2.Errors.Errors())
	assert.Empty(t, c2.Errors.ByType(ErrorTypeAny))
}

func CreateTestContext() (c *Context, w *httptest.ResponseRecorder) {
	w = httptest.NewRecorder()
	c = newContext()
	c.Writer.ResponseWriter = w
	return
}

