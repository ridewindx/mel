package mel

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"reflect"
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

