package mel

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
)



func TestRoutesGroupBasic(t *testing.T) {
	router := NewRouter()
	group := router.Group("/hola", func(c *Context) {})
	group.Use(func(c *Context) {})

	assert.Len(t, group.Handlers, 2)
	assert.Equal(t, group.BasePath, "/hola")
	assert.Equal(t, group.router, router)

	group2 := group.Group("manu")
	group2.Use(func(c *Context) {}, func(c *Context) {})

	assert.Len(t, group2.Handlers, 4)
	assert.Equal(t, group2.BasePath, "/hola/manu")
	assert.Equal(t, group2.router, router)
}

func TestRouterGroupBasicHandle(t *testing.T) {
	performRequestInGroup(t, "GET")
	performRequestInGroup(t, "POST")
	performRequestInGroup(t, "PUT")
	performRequestInGroup(t, "PATCH")
	performRequestInGroup(t, "DELETE")
	performRequestInGroup(t, "HEAD")
	performRequestInGroup(t, "OPTIONS")
}

func performRequestInGroup(t *testing.T, method string) {
	app := New()
	v1 := app.Group("v1", func(c *Context) {})
	assert.Equal(t, v1.BasePath, "/v1")

	login := v1.Group("/login/", func(c *Context) {}, func(c *Context) {})
	assert.Equal(t, login.BasePath, "/v1/login/")

	handler := func(c *Context) {
		c.Status(400)
		c.Text("the method was %s and index %d", c.Request.Method, c.index)
	}

	switch method {
	case "GET":
		v1.Get("/test", handler)
		login.Get("/test", handler)
	case "POST":
		v1.Post("/test", handler)
		login.Post("/test", handler)
	case "PUT":
		v1.Put("/test", handler)
		login.Put("/test", handler)
	case "PATCH":
		v1.Patch("/test", handler)
		login.Patch("/test", handler)
	case "DELETE":
		v1.Delete("/test", handler)
		login.Delete("/test", handler)
	case "HEAD":
		v1.Head("/test", handler)
		login.Head("/test", handler)
	case "OPTIONS":
		v1.Options("/test", handler)
		login.Options("/test", handler)
	default:
		panic("unknown method")
	}

	w := performRequest(app, method, "/v1/login/test")
	assert.Equal(t, 400, w.Code)
	assert.Equal(t, w.Body.String(), "the method was " + method + " and index 3")

	w = performRequest(app, method, "/v1/test")
	assert.Equal(t, 400, w.Code)
	assert.Equal(t, w.Body.String(), "the method was " + method + " and index 1")
}

func performRequest(r http.Handler, method, path string) *httptest.ResponseRecorder {
	req, err := http.NewRequest(method, path, nil)
	if err != nil || req == nil {
		panic(err)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

