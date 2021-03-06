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
	"time"
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
	assert.Nil(t, c1.Mel)

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

	assert.Nil(t, c2.Mel)
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
	c.Writer.Reset(w)
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
	c.JSON(201, Map{"foo": "bar"})

	assert.Equal(t, w.Code, 201)
	assert.Equal(t, w.Body.String(), "{\"foo\":\"bar\"}\n")
	assert.Equal(t, w.HeaderMap.Get("Content-Type"), "application/json; charset=utf-8")
}

func TestContextRenderAPIJSON(t *testing.T) {
	c, w := CreateTestContext()
	c.Header("Content-Type", "application/vnd.api+json")
	c.JSON(201, Map{"foo": "bar"})

	assert.Equal(t, w.Code, 201)
	assert.Equal(t, w.Body.String(), "{\"foo\":\"bar\"}\n")
	assert.Equal(t, w.HeaderMap.Get("Content-Type"), "application/vnd.api+json")
}

func TestContextRenderIndentedJSON(t *testing.T) {
	c, w := CreateTestContext()
	c.JSON(201, Map{"foo": "bar", "bar": "foo", "nested": Map{"foo": "bar"}}, true)

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
	c.Mel = r

	c.HTML(201, "t", Map{"name": "alexandernyquist"})

	assert.Equal(t, w.Code, 201)
	assert.Equal(t, w.Body.String(), "Hello alexandernyquist")
	assert.Equal(t, w.HeaderMap.Get("Content-Type"), "text/html; charset=utf-8")
}

func TestContextRenderXML(t *testing.T) {
	c, w := CreateTestContext()
	c.XML(201, render.XMLMap{"foo": "bar"})

	assert.Equal(t, w.Code, 201)
	assert.Equal(t, w.Body.String(), "<map><foo>bar</foo></map>")
	assert.Equal(t, w.HeaderMap.Get("Content-Type"), "application/xml; charset=utf-8")
}

func TestContextRenderString(t *testing.T) {
	c, w := CreateTestContext()
	c.Text(201, "test %s %d", "string", 2)

	assert.Equal(t, w.Code, 201)
	assert.Equal(t, w.Body.String(), "test string 2")
	assert.Equal(t, w.HeaderMap.Get("Content-Type"), "text/plain; charset=utf-8")
}

func TestContextRenderHTMLString(t *testing.T) {
	c, w := CreateTestContext()
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Text(201, "<html>%s %d</html>", "string", 3)

	assert.Equal(t, w.Code, 201)
	assert.Equal(t, w.Body.String(), "<html>string 3</html>")
	assert.Equal(t, w.HeaderMap.Get("Content-Type"), "text/html; charset=utf-8")
}

func TestContextRenderData(t *testing.T) {
	c, w := CreateTestContext()
	c.Data(201, "text/csv", []byte(`foo,bar`))

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

func TestContextRenderFile(t *testing.T) {
	c, w := CreateTestContext()
	c.Request, _ = http.NewRequest("GET", "/", nil)
	c.File("./mel.go")

	assert.Equal(t, w.Code, 200)
	assert.Contains(t, w.Body.String(), "func New() *Mel {")
	assert.Equal(t, w.HeaderMap.Get("Content-Type"), "text/plain; charset=utf-8")
}

func TestContextRenderYAML(t *testing.T) {
	c, w := CreateTestContext()
	c.YAML(201, Map{"foo": "bar"})

	assert.Equal(t, w.Code, 201)
	assert.Equal(t, w.Body.String(), "foo: bar\n")
	assert.Equal(t, w.HeaderMap.Get("Content-Type"), "application/x-yaml; charset=utf-8")
}

func TestContextHeaders(t *testing.T) {
	c, _ := CreateTestContext()
	c.Header("Content-Type", "text/plain")
	c.Header("X-Custom", "value")

	assert.Equal(t, c.Writer.Header().Get("Content-Type"), "text/plain")
	assert.Equal(t, c.Writer.Header().Get("X-Custom"), "value")

	c.Header("Content-Type", "text/html")
	c.Header("X-Custom", "")

	assert.Equal(t, c.Writer.Header().Get("Content-Type"), "text/html")
	_, exist := c.Writer.Header()["X-Custom"]
	assert.False(t, exist)
}

func TestContextRenderRedirectWithRelativePath(t *testing.T) {
	c, w := CreateTestContext()
	c.Request, _ = http.NewRequest("POST", "http://example.com", nil)
	assert.Panics(t, func() { c.Redirect(299, "/new_path") })
	assert.Panics(t, func() { c.Redirect(309, "/new_path") })

	c.Redirect(301, "/path")
	assert.Equal(t, w.Code, 301)
	assert.Equal(t, w.Header().Get("Location"), "/path")
}

func TestContextRenderRedirectWithAbsolutePath(t *testing.T) {
	c, w := CreateTestContext()
	c.Request, _ = http.NewRequest("POST", "http://example.com", nil)

	c.Redirect(302, "http://google.com")
	assert.Equal(t, w.Code, 302)
	assert.Equal(t, w.Header().Get("Location"), "http://google.com")
}

func TestContextRenderRedirectWith201(t *testing.T) {
	c, w := CreateTestContext()
	c.Request, _ = http.NewRequest("POST", "http://example.com", nil)

	c.Redirect(201, "/resource")
	assert.Equal(t, w.Code, 201)
	assert.Equal(t, w.Header().Get("Location"), "/resource")
}

func TestContextRenderRedirectAll(t *testing.T) {
	c, _ := CreateTestContext()
	c.Request, _ = http.NewRequest("POST", "http://example.com", nil)
	assert.Panics(t, func() { c.Redirect(200, "/resource") })
	assert.Panics(t, func() { c.Redirect(202, "/resource") })
	assert.Panics(t, func() { c.Redirect(299, "/resource") })
	assert.Panics(t, func() { c.Redirect(309, "/resource") })
	assert.NotPanics(t, func() { c.Redirect(300, "/resource") })
	assert.NotPanics(t, func() { c.Redirect(308, "/resource") })
}

func TestContextIsAborted(t *testing.T) {
	c, _ := CreateTestContext()
	assert.False(t, c.IsAborted())

	c.Abort()
	assert.True(t, c.IsAborted())

	c.Next()
	assert.True(t, c.IsAborted())

	c.index++
	assert.True(t, c.IsAborted())
}

func TestContextAbortWithStatus(t *testing.T) {
	c, w := CreateTestContext()
	c.index = 4
	c.AbortWithStatus(401)

	assert.Equal(t, c.index, abortIndex)
	assert.Equal(t, c.Writer.Status(), 401)
	assert.Equal(t, w.Code, 401)
	assert.True(t, c.IsAborted())
}

func TestContextError(t *testing.T) {
	c, _ := CreateTestContext()
	assert.Empty(t, c.Errors)

	c.Error(errors.New("first error"))
	assert.Len(t, c.Errors, 1)
	assert.Equal(t, c.Errors.String(), "Error #01: first error\n")

	c.Error(&Error{
		Err:  errors.New("second error"),
		Meta: "some data 2",
		Type: ErrorTypePublic,
	})
	assert.Len(t, c.Errors, 2)

	assert.Equal(t, c.Errors[0].Err, errors.New("first error"))
	assert.Nil(t, c.Errors[0].Meta)
	assert.Equal(t, c.Errors[0].Type, ErrorTypePrivate)

	assert.Equal(t, c.Errors[1].Err, errors.New("second error"))
	assert.Equal(t, c.Errors[1].Meta, "some data 2")
	assert.Equal(t, c.Errors[1].Type, ErrorTypePublic)

	assert.Equal(t, c.Errors.Last(), c.Errors[1])
}

func TestContextTypedError(t *testing.T) {
	c, _ := CreateTestContext()
	c.Error(errors.New("externo 0")).Type = ErrorTypePublic
	c.Error(errors.New("interno 0")).Type = ErrorTypePrivate

	for _, err := range c.Errors.ByType(ErrorTypePublic) {
		assert.Equal(t, err.Type, ErrorTypePublic)
	}
	for _, err := range c.Errors.ByType(ErrorTypePrivate) {
		assert.Equal(t, err.Type, ErrorTypePrivate)
	}
	assert.Equal(t, c.Errors.Errors(), []string{"externo 0", "interno 0"})
}

func TestContextAbortWithError(t *testing.T) {
	c, w := CreateTestContext()
	c.AbortWithError(401, errors.New("bad input")).Meta = "some input"

	assert.Equal(t, w.Code, 401)
	assert.Equal(t, c.index, abortIndex)
	assert.True(t, c.IsAborted())
}

func TestContextClientIP(t *testing.T) {
	r := New()
	c, _ := CreateTestContext()
	c.Mel = r
	c.Request, _ = http.NewRequest("POST", "/", nil)

	c.Request.Header.Set("X-Real-IP", " 10.10.10.10  ")
	c.Request.Header.Set("X-Forwarded-For", "  20.20.20.20, 30.30.30.30")
	c.Request.RemoteAddr = "  40.40.40.40:42123 "

	assert.Equal(t, c.ClientIP(), "10.10.10.10")

	c.Request.Header.Del("X-Real-IP")
	assert.Equal(t, c.ClientIP(), "20.20.20.20")

	c.Request.Header.Set("X-Forwarded-For", "30.30.30.30  ")
	assert.Equal(t, c.ClientIP(), "30.30.30.30")

	c.Request.Header.Del("X-Forwarded-For")
	assert.Equal(t, c.ClientIP(), "40.40.40.40")
}

func TestContextContentType(t *testing.T) {
	c, _ := CreateTestContext()
	c.Request, _ = http.NewRequest("POST", "/", nil)
	c.Request.Header.Set("Content-Type", "application/json; charset=utf-8")

	assert.Equal(t, c.ContentType(), "application/json")
}

func TestContextAutoBindJSON(t *testing.T) {
	c, _ := CreateTestContext()
	c.Request, _ = http.NewRequest("POST", "/", bytes.NewBufferString("{\"foo\":\"bar\", \"bar\":\"foo\"}"))
	c.Request.Header.Add("Content-Type", binding.MIMEJSON)

	var obj struct {
		Foo string `json:"foo"`
		Bar string `json:"bar"`
	}
	assert.NoError(t, c.Bind(&obj))
	assert.Equal(t, obj.Bar, "foo")
	assert.Equal(t, obj.Foo, "bar")
	assert.Empty(t, c.Errors)
}

func TestContextBindWithJSON(t *testing.T) {
	c, w := CreateTestContext()
	c.Request, _ = http.NewRequest("POST", "/", bytes.NewBufferString("{\"foo\":\"bar\", \"bar\":\"foo\"}"))
	c.Request.Header.Add("Content-Type", binding.MIMEXML)

	var obj struct {
		Foo string `json:"foo"`
		Bar string `json:"bar"`
	}
	assert.NoError(t, c.BindJSON(&obj))
	assert.Equal(t, obj.Bar, "foo")
	assert.Equal(t, obj.Foo, "bar")
	assert.Equal(t, w.Body.Len(), 0)
}

func TestContextBadAutoBind(t *testing.T) {
	c, w := CreateTestContext()
	c.Request, _ = http.NewRequest("POST", "http://example.com", bytes.NewBufferString("\"foo\":\"bar\", \"bar\":\"foo\"}"))
	c.Request.Header.Add("Content-Type", binding.MIMEJSON)
	var obj struct {
		Foo string `json:"foo"`
		Bar string `json:"bar"`
	}

	assert.False(t, c.IsAborted())
	assert.Error(t, c.Bind(&obj))

	assert.Empty(t, obj.Bar)
	assert.Empty(t, obj.Foo)
	assert.Equal(t, w.Code, 400)
	assert.True(t, c.IsAborted())
}

func TestContextGolangContext(t *testing.T) {
	c, _ := CreateTestContext()
	c.Request, _ = http.NewRequest("POST", "/", bytes.NewBufferString("{\"foo\":\"bar\", \"bar\":\"foo\"}"))
	assert.NoError(t, c.Err())
	assert.Nil(t, c.Done())
	ti, ok := c.Deadline()
	assert.Equal(t, ti, time.Time{})
	assert.False(t, ok)
	assert.Equal(t, c.Value(0), c.Request)
	assert.Nil(t, c.Value("foo"))

	c.Set("foo", "bar")
	assert.Equal(t, c.Value("foo"), "bar")
	assert.Nil(t, c.Value(1))
}
