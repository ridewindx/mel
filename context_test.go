package mel

import (
	"testing"
	"github.com/stretchr/testify/assert"

	"strings"
	"errors"
	"net/http"
	"net/http/httptest"
	"bytes"
	"mime/multipart"
	"github.com/ridewindx/mel/binding"
	"fmt"
	"html/template"
	"github.com/ridewindx/mel/render"
	"github.com/manucorporat/sse"
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

func TestContextQueryAndPostForm(t *testing.T) {
	c, _ := CreateTestContext()
	body := bytes.NewBufferString("foo=bar&page=11&both=&foo=second")
	c.Request, _ = http.NewRequest("POST", "/?both=GET&id=main&id=omit&array[]=first&array[]=second", body)
	c.Request.Header.Add("Content-Type", binding.MIMEPOSTForm)

	assert.Equal(t, c.PostForm("foo", "none"), "bar")
	assert.Equal(t, c.PostForm("foo"), "bar")
	assert.Empty(t, c.Query("foo"))

	value, ok := c.GetPostForm("page")
	assert.True(t, ok)
	assert.Equal(t, value, "11")
	assert.Equal(t, c.PostForm("page", "0"), "11")
	assert.Equal(t, c.PostForm("page"), "11")
	assert.Equal(t, c.Query("page"), "")

	value, ok = c.GetPostForm("both")
	assert.True(t, ok)
	assert.Empty(t, value)
	assert.Empty(t, c.PostForm("both"))
	assert.Equal(t, c.PostForm("both", "nothing"), "")
	assert.Equal(t, c.Query("both"), "GET")

	value, ok = c.GetQuery("id")
	assert.True(t, ok)
	assert.Equal(t, value, "main")
	assert.Equal(t, c.PostForm("id", "000"), "000")
	assert.Equal(t, c.Query("id"), "main")
	assert.Empty(t, c.PostForm("id"))

	value, ok = c.GetQuery("NoKey")
	assert.False(t, ok)
	assert.Empty(t, value)
	value, ok = c.GetPostForm("NoKey")
	assert.False(t, ok)
	assert.Empty(t, value)
	assert.Equal(t, c.PostForm("NoKey", "nada"), "nada")
	assert.Equal(t, c.Query("NoKey", "nothing"), "nothing")
	assert.Empty(t, c.PostForm("NoKey"))
	assert.Empty(t, c.Query("NoKey"))

	var obj struct {
		Foo   string   `form:"foo"`
		ID    string   `form:"id"`
		Page  int      `form:"page"`
		Both  string   `form:"both"`
		Array []string `form:"array[]"`
	}
	assert.NoError(t, c.Bind(&obj))
	assert.Equal(t, obj.Foo, "bar")
	assert.Equal(t, obj.ID, "main")
	assert.Equal(t, obj.Page, 11)
	assert.Equal(t, obj.Both, "")
	assert.Equal(t, obj.Array, []string{"first", "second"})

	values, ok := c.GetQuerys("array[]")
	assert.True(t, ok)
	assert.Equal(t, "first", values[0])
	assert.Equal(t, "second", values[1])

	values = c.Querys("array[]")
	assert.Equal(t, "first", values[0])
	assert.Equal(t, "second", values[1])

	values = c.Querys("nokey")
	assert.Equal(t, 0, len(values))

	values = c.Querys("both")
	assert.Equal(t, 1, len(values))
	assert.Equal(t, "GET", values[0])
}

func TestContextPostFormMultipart(t *testing.T) {
	c, _ := CreateTestContext()
	c.Request = createMultipartRequest()

	var obj struct {
		Foo      string   `form:"foo"`
		Bar      string   `form:"bar"`
		BarAsInt int      `form:"bar"`
		Array    []string `form:"array"`
		ID       string   `form:"id"`
	}
	assert.NoError(t, c.Bind(&obj))
	assert.Equal(t, obj.Foo, "bar")
	assert.Equal(t, obj.Bar, "10")
	assert.Equal(t, obj.BarAsInt, 10)
	assert.Equal(t, obj.Array, []string{"first", "second"})
	assert.Equal(t, obj.ID, "")

	value, ok := c.GetQuery("foo")
	assert.False(t, ok)
	assert.Empty(t, value)
	assert.Empty(t, c.Query("bar"))
	assert.Equal(t, c.Query("id", "nothing"), "nothing")

	value, ok = c.GetPostForm("foo")
	assert.True(t, ok)
	assert.Equal(t, value, "bar")
	assert.Equal(t, c.PostForm("foo"), "bar")

	value, ok = c.GetPostForm("array")
	assert.True(t, ok)
	assert.Equal(t, value, "first")
	assert.Equal(t, c.PostForm("array"), "first")

	assert.Equal(t, c.PostForm("bar", "nothing"), "10")

	value, ok = c.GetPostForm("id")
	assert.True(t, ok)
	assert.Empty(t, value)
	assert.Empty(t, c.PostForm("id"))
	assert.Empty(t, c.PostForm("id", "nothing"))

	value, ok = c.GetPostForm("nokey")
	assert.False(t, ok)
	assert.Empty(t, value)
	assert.Equal(t, c.PostForm("nokey", "nothing"), "nothing")

	values, ok := c.GetPostForms("array")
	assert.True(t, ok)
	assert.Equal(t, "first", values[0])
	assert.Equal(t, "second", values[1])

	values = c.PostForms("array")
	assert.Equal(t, "first", values[0])
	assert.Equal(t, "second", values[1])

	values = c.PostForms("nokey")
	assert.Equal(t, 0, len(values))

	values = c.PostForms("foo")
	assert.Equal(t, 1, len(values))
	assert.Equal(t, "bar", values[0])
}

func TestContextSetCookie(t *testing.T) {
	c, _ := CreateTestContext()
	c.SetCookie(&Cookie{
		Name: "user",
		Value: "gin",
		Path: "/",
		Domain: "localhost",
		MaxAge: 1,
		Secure: true,
		HttpOnly: true,
	})
	assert.Equal(t, c.Writer.Header().Get("Set-Cookie"),
		         "user=gin; Path=/; Domain=localhost; Max-Age=1; HttpOnly; Secure")
}

func TestContextGetCookie(t *testing.T) {
	c, _ := CreateTestContext()
	c.Request, _ = http.NewRequest("GET", "/get", nil)
	c.Request.Header.Set("Cookie", "user=gin")
	cookie, _ := c.Cookie("user")
	assert.Equal(t, cookie, "gin")
}

func TestContextRenderJSON(t *testing.T) {
	c, w := CreateTestContext()
	c.Status(201)
	c.JSON(Map{"foo": "bar"})

	assert.Equal(t, w.Code, 201)
	assert.Equal(t, w.Body.String(), "{\"foo\":\"bar\"}\n")
	assert.Equal(t, w.HeaderMap.Get("Content-Type"), "application/json; charset=utf-8")
}

func TestContextRenderAPIJSON(t *testing.T) {
	c, w := CreateTestContext()
	c.Header("Content-Type", "application/vnd.api+json")
	c.Status(201)
	c.JSON(Map{"foo": "bar"})

	assert.Equal(t, w.Code, 201)
	assert.Equal(t, w.Body.String(), "{\"foo\":\"bar\"}\n")
	assert.Equal(t, w.HeaderMap.Get("Content-Type"), "application/vnd.api+json")
}

func TestContextRenderIndentedJSON(t *testing.T) {
	c, w := CreateTestContext()
	c.Status(201)
	c.JSON(Map{"foo": "bar", "bar": "foo", "nested": Map{"foo": "bar"}}, true)

	assert.Equal(t, w.Code, 201)
	assert.Equal(t, w.Body.String(), `{
    "bar": "foo",
    "foo": "bar",
    "nested": {
        "foo": "bar"
    }
}`)
	assert.Equal(t, w.HeaderMap.Get("Content-Type"), "application/json; charset=utf-8")
}

func TestContextRenderHTML(t *testing.T) {
	c, w := CreateTestContext()
	templ := template.Must(template.New("t").Parse(`Hello {{.name}}`))
	r := New()
	r.SetTemplate(templ)
	c.mel = r

	c.Status(201)
	c.HTML("t", Map{"name": "alexandernyquist"})

	assert.Equal(t, w.Code, 201)
	assert.Equal(t, w.Body.String(), "Hello alexandernyquist")
	assert.Equal(t, w.HeaderMap.Get("Content-Type"), "text/html; charset=utf-8")
}

func TestContextRenderXML(t *testing.T) {
	c, w := CreateTestContext()
	c.Status(201)
	c.XML(render.XMLMap{"foo": "bar"})

	assert.Equal(t, w.Code, 201)
	assert.Equal(t, w.Body.String(), "<map><foo>bar</foo></map>")
	assert.Equal(t, w.HeaderMap.Get("Content-Type"), "application/xml; charset=utf-8")
}

func TestContextRenderString(t *testing.T) {
	c, w := CreateTestContext()
	c.Status(201)
	c.Text("test %s %d", "string", 2)

	assert.Equal(t, w.Code, 201)
	assert.Equal(t, w.Body.String(), "test string 2")
	assert.Equal(t, w.HeaderMap.Get("Content-Type"), "text/plain; charset=utf-8")
}

func TestContextRenderHTMLString(t *testing.T) {
	c, w := CreateTestContext()
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(201)
	c.Text("<html>%s %d</html>", "string", 3)

	assert.Equal(t, w.Code, 201)
	assert.Equal(t, w.Body.String(), "<html>string 3</html>")
	assert.Equal(t, w.HeaderMap.Get("Content-Type"), "text/html; charset=utf-8")
}

func TestContextRenderData(t *testing.T) {
	c, w := CreateTestContext()
	c.Status(201)
	c.Data("text/csv", []byte(`foo,bar`))

	assert.Equal(t, w.Code, 201)
	assert.Equal(t, w.Body.String(), "foo,bar")
	assert.Equal(t, w.HeaderMap.Get("Content-Type"), "text/csv")
}

func TestContextRenderSSE(t *testing.T) {
	c, w := CreateTestContext()
	c.SSE("float", 1.5)
	e := sse.Event{
		Id:   "123",
		Data: "text",
	}
	e.Render(c.Writer)
	c.SSE("chat", Map{
		"foo": "bar",
		"bar": "foo",
	})

	assert.Equal(t, strings.Replace(w.Body.String(), " ", "", -1),
					strings.Replace(`event:float
data:1.5

id:123
data:text

event:chat
data:{"bar":"foo","foo":"bar"}

`, " ", "", -1))
}

