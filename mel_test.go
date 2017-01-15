package mel

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"reflect"
	"time"
	"net"
	"net/http"
	"net/http/httptest"
	"io/ioutil"
	"os"
	"fmt"
	"bufio"
	"runtime"
)

func TestCreateApp(t *testing.T) {
	r := New()
	assert.Equal(t, "/", r.BasePath)
	assert.Equal(t, r.Router, r.router)
	assert.Empty(t, r.Handlers)
}

func TestAddRoute(t *testing.T) {
	router := NewRouter()
	router.Register("GET", "/", func(*Context) {})

	assert.Len(t, router.trees, len(Methods))
	assert.NotNil(t, router.trees["GET"].children)
	assert.Nil(t, router.trees["POST"].children)
	assert.Len(t, router.trees["GET"].children, 1)
	assert.Len(t, router.trees["POST"].children, 0)

	router.Register("POST", "/", func(*Context) {})

	assert.NotNil(t, router.trees["GET"].children)
	assert.NotNil(t, router.trees["POST"].children)
	assert.Len(t, router.trees["GET"].children, 1)
	assert.Len(t, router.trees["POST"].children, 1)

	router.Register("POST", "/post", func(*Context) {})
	assert.Len(t, router.trees["GET"].children, 1)
	assert.Len(t, router.trees["POST"].children, 2)
}

func TestAddRouteFails(t *testing.T) {
	router := New()
	assert.Panics(t, func() { router.Register("", "/", func(*Context) {}) })
	assert.Panics(t, func() { router.Register("GET", "a", func(*Context) {}) })
	assert.Panics(t, func() { router.Register("GET", "/", []Handler{}) })

	router.Register("POST", "/post", func(*Context) {})
	assert.NotPanics(t, func() { router.Register("POST", "/post", func(*Context) {}) }) // TODO: reasonable?
}

func compareFunc(t *testing.T, a, b interface{}) {
	sf1 := reflect.ValueOf(a)
	sf2 := reflect.ValueOf(b)
	if sf1.Pointer() != sf2.Pointer() {
		t.Error("different functions")
	}
}

func TestNoRouteWithoutGlobalHandlers(t *testing.T) {
	var middleware0  = func(*Context) {}
	var middleware1  = func(*Context) {}

	router := New()

	router.NoRoute(middleware0)
	assert.Nil(t, router.Handlers)
	assert.Len(t, router.noRoute, 1)
	assert.Len(t, router.allNoRoute, 1)
	compareFunc(t, router.noRoute[0], middleware0)
	compareFunc(t, router.allNoRoute[0], middleware0)

	router.NoRoute(middleware1, middleware0)
	assert.Len(t, router.noRoute, 2)
	assert.Len(t, router.allNoRoute, 2)
	compareFunc(t, router.noRoute[0], middleware1)
	compareFunc(t, router.allNoRoute[0], middleware1)
	compareFunc(t, router.noRoute[1], middleware0)
	compareFunc(t, router.allNoRoute[1], middleware0)
}

func TestNoRouteWithGlobalHandlers(t *testing.T) {
	var middleware0 = func(*Context) {}
	var middleware1 = func(*Context) {}
	var middleware2 = func(*Context) {}

	router := New()
	router.Use(middleware2)

	router.NoRoute(middleware0)
	assert.Len(t, router.allNoRoute, 2)
	assert.Len(t, router.Handlers, 1)
	assert.Len(t, router.noRoute, 1)

	compareFunc(t, router.Handlers[0], middleware2)
	compareFunc(t, router.noRoute[0], middleware0)
	compareFunc(t, router.allNoRoute[0], middleware2)
	compareFunc(t, router.allNoRoute[1], middleware0)

	router.Use(middleware1)
	assert.Len(t, router.allNoRoute, 3)
	assert.Len(t, router.Handlers, 2)
	assert.Len(t, router.noRoute, 1)

	compareFunc(t, router.Handlers[0], middleware2)
	compareFunc(t, router.Handlers[1], middleware1)
	compareFunc(t, router.noRoute[0], middleware0)
	compareFunc(t, router.allNoRoute[0], middleware2)
	compareFunc(t, router.allNoRoute[1], middleware1)
	compareFunc(t, router.allNoRoute[2], middleware0)
}

func TestNoMethodWithoutGlobalHandlers(t *testing.T) {
	var middleware0 = func(*Context) {}
	var middleware1 = func(*Context) {}

	router := New()

	router.NoMethod(middleware0)
	assert.Empty(t, router.Handlers)
	assert.Len(t, router.noMethod, 1)
	assert.Len(t, router.allNoMethod, 1)
	compareFunc(t, router.noMethod[0], middleware0)
	compareFunc(t, router.allNoMethod[0], middleware0)

	router.NoMethod(middleware1, middleware0)
	assert.Len(t, router.noMethod, 2)
	assert.Len(t, router.allNoMethod, 2)
	compareFunc(t, router.noMethod[0], middleware1)
	compareFunc(t, router.allNoMethod[0], middleware1)
	compareFunc(t, router.noMethod[1], middleware0)
	compareFunc(t, router.allNoMethod[1], middleware0)
}

func TestNoMethodWithGlobalHandlers(t *testing.T) {
	var middleware0 = func(*Context) {}
	var middleware1 = func(*Context) {}
	var middleware2 = func(*Context) {}

	router := New()
	router.Use(middleware2)

	router.NoMethod(middleware0)
	assert.Len(t, router.allNoMethod, 2)
	assert.Len(t, router.Handlers, 1)
	assert.Len(t, router.noMethod, 1)

	compareFunc(t, router.Handlers[0], middleware2)
	compareFunc(t, router.noMethod[0], middleware0)
	compareFunc(t, router.allNoMethod[0], middleware2)
	compareFunc(t, router.allNoMethod[1], middleware0)

	router.Use(middleware1)
	assert.Len(t, router.allNoMethod, 3)
	assert.Len(t, router.Handlers, 2)
	assert.Len(t, router.noMethod, 1)

	compareFunc(t, router.Handlers[0], middleware2)
	compareFunc(t, router.Handlers[1], middleware1)
	compareFunc(t, router.noMethod[0], middleware0)
	compareFunc(t, router.allNoMethod[0], middleware2)
	compareFunc(t, router.allNoMethod[1], middleware1)
	compareFunc(t, router.allNoMethod[2], middleware0)
}

func testRequest(t *testing.T, url string) {
	resp, err := http.Get(url)
	defer resp.Body.Close()
	assert.NoError(t, err)

	body, ioerr := ioutil.ReadAll(resp.Body)
	assert.NoError(t, ioerr)
	assert.Equal(t, "it worked", string(body), "resp body should match")
	assert.Equal(t, "200 OK", resp.Status, "should get a 200")
}

func TestRunEmpty(t *testing.T) {
	os.Setenv("PORT", "")
	router := New()
	go func() {
		router.Get("/example", func(c *Context) {
			c.Text(http.StatusOK, "it worked")
		})
		assert.NoError(t, router.Run())
	}()
	// have to wait for the goroutine to start and run the server
	// otherwise the main thread will complete
	time.Sleep(5 * time.Millisecond)

	assert.Error(t, router.Run(":8080"))
	testRequest(t, "http://localhost:8080/example")
}

func TestRunEmptyWithEnv(t *testing.T) {
	os.Setenv("PORT", "3123")
	router := New()
	go func() {
		router.Get("/example", func(c *Context) { c.Text(http.StatusOK, "it worked") })
		assert.NoError(t, router.Run())
	}()
	// have to wait for the goroutine to start and run the server
	// otherwise the main thread will complete
	time.Sleep(5 * time.Millisecond)

	assert.Error(t, router.Run(":3123"))
	testRequest(t, "http://localhost:3123/example")
}

func TestRunTooMuchParams(t *testing.T) {
	router := New()
	assert.Panics(t, func() {
		router.Run("2", "2")
	})
}

func TestRunWithPort(t *testing.T) {
	router := New()
	go func() {
		router.Get("/example", func(c *Context) { c.Text(http.StatusOK, "it worked") })
		assert.NoError(t, router.Run(":5150"))
	}()
	// have to wait for the goroutine to start and run the server
	// otherwise the main thread will complete
	time.Sleep(5 * time.Millisecond)

	assert.Error(t, router.Run(":5150"))
	testRequest(t, "http://localhost:5150/example")
}

func TestUnixSocket(t *testing.T) {
	if runtime.GOOS != "unix" {
		return
	}

	router := New()

	go func() {
		router.Get("/example", func(c *Context) { c.Text(http.StatusOK, "it worked") })
		assert.NoError(t, router.RunUnix("/tmp/unix_unit_test"))
	}()
	// have to wait for the goroutine to start and run the server
	// otherwise the main thread will complete
	time.Sleep(5 * time.Millisecond)

	c, err := net.Dial("unix", "/tmp/unix_unit_test")
	assert.NoError(t, err)

	fmt.Fprintf(c, "GET /example HTTP/1.0\r\n\r\n")
	scanner := bufio.NewScanner(c)
	var response string
	for scanner.Scan() {
		response += scanner.Text()
	}
	assert.Contains(t, response, "HTTP/1.0 200", "should get a 200")
	assert.Contains(t, response, "it worked", "resp body should match")
}

func TestBadUnixSocket(t *testing.T) {
	if runtime.GOOS != "unix" {
		return
	}

	router := New()
	assert.Error(t, router.RunUnix("#/tmp/unix_unit_test"))
}

func TestWithHttptestWithAutoSelectedPort(t *testing.T) {
	router := New()
	router.Get("/example", func(c *Context) { c.Text(http.StatusOK, "it worked") })

	ts := httptest.NewServer(router)
	defer ts.Close()

	testRequest(t, ts.URL+"/example")
}

func TestWithHttptestWithSpecifiedPort(t *testing.T) {
	router := New()
	router.Get("/example", func(c *Context) { c.Text(http.StatusOK, "it worked") })

	l, _ := net.Listen("tcp", ":8033")
	ts := httptest.Server{
		Listener: l,
		Config:   &http.Server{Handler: router},
	}
	ts.Start()
	defer ts.Close()

	testRequest(t, "http://localhost:8033/example")
}
