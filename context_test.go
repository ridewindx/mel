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

func TestContextHandlers(t *testing.T) {
	c, _ := CreateTestContext()
	assert.Nil(t, c.handlers)

	c.handlers = []Handler{}
	assert.NotNil(t, c.handlers)
}

func TestContextSetGet(t *testing.T) {
	c, _ := CreateTestContext()
	c.Set("foo", "bar")

	value, exists := c.Get("foo")
	assert.Equal(t, value, "bar")
	assert.True(t, exists)

	value, exists = c.Get("foo2")
	assert.Nil(t, value)
	assert.False(t, exists)

	assert.Equal(t, c.MustGet("foo"), "bar")
	assert.Panics(t, func() { c.MustGet("no_exist") })
}

func TestContextSetGetValues(t *testing.T) {
	c, _ := CreateTestContext()
	c.Set("string", "this is a string")
	c.Set("int32", int32(-42))
	c.Set("int64", int64(42424242424242))
	c.Set("uint64", uint64(42))
	c.Set("float32", float32(4.2))
	c.Set("float64", 4.2)
	var a interface{} = 1
	c.Set("intInterface", a)

	assert.Exactly(t, c.MustGet("string").(string), "this is a string")
	assert.Exactly(t, c.MustGet("int32").(int32), int32(-42))
	assert.Exactly(t, c.MustGet("int64").(int64), int64(42424242424242))
	assert.Exactly(t, c.MustGet("uint64").(uint64), uint64(42))
	assert.Exactly(t, c.MustGet("float32").(float32), float32(4.2))
	assert.Exactly(t, c.MustGet("float64").(float64), 4.2)
	assert.Exactly(t, c.MustGet("intInterface").(int), 1)
}

func TestContextQuery(t *testing.T) {
	c, _ := CreateTestContext()
	c.Request, _ = http.NewRequest("GET", "http://example.com/?foo=bar&page=10&id=", nil)

	value, ok := c.GetQuery("foo")
	assert.True(t, ok)
	assert.Equal(t, value, "bar")
	assert.Equal(t, c.Query("foo", "none"), "bar")
	assert.Equal(t, c.Query("foo"), "bar")

	value, ok = c.GetQuery("page")
	assert.True(t, ok)
	assert.Equal(t, value, "10")
	assert.Equal(t, c.Query("page", "0"), "10")
	assert.Equal(t, c.Query("page"), "10")

	value, ok = c.GetQuery("id")
	assert.True(t, ok)
	assert.Empty(t, value)
	assert.Equal(t, c.Query("id", "nada"), "")
	assert.Empty(t, c.Query("id"))

	value, ok = c.GetQuery("NoKey")
	assert.False(t, ok)
	assert.Empty(t, value)
	assert.Equal(t, c.Query("NoKey", "nada"), "nada")
	assert.Empty(t, c.Query("NoKey"))

	// postform should not mess
	value, ok = c.GetPostForm("page")
	assert.False(t, ok)
	assert.Empty(t, value)
	assert.Empty(t, c.PostForm("foo"))
}
