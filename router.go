package mel

import (
	"bytes"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"net/http"
)

var Methods = []string{
	"GET",
	"POST",
	"HEAD",
	"DELETE",
	"PUT",
	"OPTIONS",
	"TRACE",
	"PATCH",
}

var AllowCustomMethod = true

type RouteKind uint8

const (
	FuncRoute       RouteKind = iota // func ()
	FuncRepReqRoute                  // func (http.ResponseWriter, *http.Request)
	FuncRepRoute                     // func (http.ResponseWriter)
	FuncReqRoute                     // func (*http.Request)
	FuncCtxRoute                     // func (*mel.Context)
)

type Route struct {
	kind   RouteKind
	method reflect.Value
	handlers []Handler
}

/*
func newRoute(kind RouteKind, method reflect.Value) *Route {
	return &Route{
		kind: kind,
		method: method,
	}
}
*/

func (r *Route) execute(ctx *Context) {
	target := func(ctx *Context) {
		var args []reflect.Value
		switch r.kind {
		case FuncRoute:
		case FuncRepReqRoute:
			args = []reflect.Value{reflect.ValueOf(ctx.Writer), reflect.ValueOf(ctx.Request)}
		case FuncRepRoute:
			args = []reflect.Value{reflect.ValueOf(ctx.Writer)}
		case FuncReqRoute:
			args = []reflect.Value{reflect.ValueOf(ctx.Request)}
		case FuncCtxRoute:
			args = []reflect.Value{reflect.ValueOf(ctx)}
		}
		r.method.Call(args)
	}

	ctx.handlers = append(r.handlers, target)
}

/*
type Router interface {
	AddRoute(methods interface{}, path string, handler interface{}, middlewares ...Handler)
	Match(requestPath, method string) (*Route, Params)
}
*/

type nodeKind byte

const (
	staticNode nodeKind = iota
	namedNode
	anyNode
	regexNode
)

type node struct {
	kind     nodeKind
	segment  string // path segment
	regexp   *regexp.Regexp // non-null when kind is regexNode

	children nodes

	path     string // the entire path, only presents in the "leaf" node
	route    *Route
}

func (n *node) equal(o *node) bool {
	if n.kind == o.kind && n.segment == o.segment {
		return true
	}
	return false
}

type nodes []*node

func (e nodes) Len() int {
	return len(e)
}

func (e nodes) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

// Static route is first.
// If two static routes, content longer is first.
// Otherwise, first added, first matched.
func (e nodes) Less(i, j int) bool {
	if e[i].kind == staticNode {
		if e[j].kind == staticNode {
			return len(e[i].segment) > len(e[j].segment)
		}
		return true
	}
	if e[j].kind == staticNode {
		return false
	}
	return i < j
}

type Router struct {
	routesGroup

	trees map[string]*node
}

func NewRouter() *Router {
	trees := make(map[string]*node)
	for _, m := range Methods {
		trees[m] = &node{}
	}

	r := &Router{
        routesGroup: routesGroup{
			basePath: "/",
		},
		trees: trees,
	}

	r.routesGroup.router = r

	return r
}

var specialBytes = []byte(`.\+*?|[]{}^$`)

func isSpecial(c byte) bool {
	return bytes.IndexByte(specialBytes, c) > -1
}

func isAlpha(c byte) bool {
	return 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || c == '_'
}

func isDigit(c byte) bool {
	return '0' <= c && c <= '9'
}

func isAlphaNum(c byte) bool {
	return isAlpha(c) || isDigit(c)
}

func parsePath(path string) []*node {
	length := len(path)
	var nodes = make([]*node, 0)
	var i, start, bracket int
	for ; i < length; i++ {
		if path[i] == '/' {
			if bracket == 0 && i > start {
				nodes = append(nodes, &node{
					kind:    staticNode,
					segment: path[start:i],
				})
				start = i
			}
		} else if path[i] == '(' {
			bracket = 1
		} else if path[i] == ':' {
			nodes = append(nodes, &node{
				kind:    staticNode,
				segment: path[start : i-bracket],
			})

			start = i

			var re string
			if bracket == 1 {
				var reStart = -1
				for ; i < length && path[i] != ')'; i++ {
					if reStart == -1 && isSpecial(path[i]) {
						reStart = i
					}
				}
				if path[i] != ')' {
					panic("Route path lack of ')'")
				}
				if reStart > -1 {
					re = path[reStart:i]
				}
			} else {
				for i = i + 1; i < length && isAlphaNum(path[i]); i++ {
				}
			}

			if len(re) > 0 {
				nodes = append(nodes, &node{
					kind:    regexNode,
					segment: path[start : i-len(re)],
					regexp:  regexp.MustCompile("(" + re + ")"),
				})
			} else {
				nodes = append(nodes, &node{
					kind:    namedNode,
					segment: path[start:i],
				})
			}

			i = i + bracket
			start = i
			bracket = 0

			if i == length {
				return nodes
			}
		} else if path[i] == '*' {
			nodes = append(nodes, &node{
				kind:    staticNode,
				segment: path[start : i-bracket],
			})

			start = i

			if bracket == 1 {
				for ; i < length && path[i] != ')'; i++ {
				}
				if path[i] != ')' {
					panic("Route path lack of ')'")
				}
			} else {
				for i = i + 1; i < length && isAlphaNum(path[i]); i++ {
				}
			}

			nodes = append(nodes, &node{
				kind:    anyNode,
				segment: path[start:i],
			})

			i = i + bracket
			start = i
			bracket = 0

			if i == length {
				return nodes
			}
		} else {
			bracket = 0
		}
	}

	nodes = append(nodes, &node{
		kind:    staticNode,
		segment: path[start:i],
	})

	return nodes
}

// validate parsed nodes, any non-static route should have static route successor
func validateNodes(nodes []*node) bool {
	if len(nodes) == 0 {
		return false
	}
	var last = nodes[0]
	for _, node := range nodes[1:] {
		if last.kind != staticNode && node.kind != staticNode {
			return false
		}
		last = node
	}
	return true
}

func (r *Router) addRoute(method, path string, route *Route) {
	segments := parsePath(path)
	num := len(segments)

	leafNode := segments[num-1]
	leafNode.route = route
	leafNode.path = path

	if !validateNodes(segments) {
		panic("Any non-static route should have static route successor: " + path)
	}

	p, ok := r.trees[method]
	if !ok {
		if !AllowCustomMethod {
			panic("Not allow custom method: " + method)
		}
		p = &node{}
		r.trees[method] = p
	}

	i := 0

	outer:
	for ; i < num; i++ {
		for _, node := range p.children {
			if node.equal(segments[i]) {
				if i == num-1 {
					// override leaf node
					node.route = route
					node.path = path
				}
				p = node
				continue outer
			}
		}

		p.children = append(p.children, segments[i])
		sort.Sort(p.children) // sort by priority
		p = segments[i]
	}
}

func printNodes(i int, nodes []*node) {
	for _, n := range nodes {
		for j := 0; j < i; j++ {
			fmt.Print("  ")
		}
		if i > 1 {
			fmt.Print("┗", "  ")
		}

		fmt.Print(n.segment)
		if len(n.path) != 0 {
			fmt.Printf("  path[ %s ]", n.path)
		}
		if n.route != nil {
			fmt.Printf("  func[ %p ]", n.route.method.Interface())
		}
		fmt.Println()
		printNodes(i+1, n.children)
	}
}

func (r *Router) PrintTrees() {
	for method, n := range r.trees {
		if len(n.children) > 0 {
			fmt.Println(method)
			printNodes(1, n.children)
			fmt.Println()
		}
	}
}

func (r *Router) matchNode(n *node, path string, params Params) (*node, Params) {
	if n.kind == staticNode {
		if strings.HasPrefix(path, n.segment) {
			if len(path) == len(n.segment) {
				return n, params
			}

			for _, c := range n.children {
				newN, newParams := r.matchNode(c, path[len(n.segment):], params)
				if newN != nil {
					return newN, newParams
				}
			}
		}
	} else if n.kind == anyNode {
		for _, c := range n.children {
			idx := strings.LastIndex(path, c.segment)
			if idx > -1 {
				params = append(params, Param{n.segment, path[:idx]})
				return r.matchNode(c, path[idx:], params)
			}
		}

		return n, append(params, Param{n.segment, path})
	} else if n.kind == namedNode {
		for _, c := range n.children {
			idx := strings.Index(path, c.segment)
			if idx > -1 {
				params = append(params, Param{n.segment, path[:idx]})
				return r.matchNode(c, path[idx:], params)
			}
		}

		idx := strings.IndexByte(path, '/')
		if idx == -1 {
			params = append(params, Param{n.segment, path})
			return n, params
		}
	} else if n.kind == regexNode {
		idx := strings.IndexByte(path, '/')
		if idx > -1 {
			if n.regexp.MatchString(path[:idx]) {
				for _, c := range n.children {
					newN, newParams := r.matchNode(c, path[idx:], params)
					if newN != nil {
						return newN, append(Params{Param{n.segment, path[:idx]}}, newParams...)
					}
				}
			}

			return nil, params
		}

		for _, c := range n.children {
			idx := strings.Index(path, c.segment)
			if idx > -1 && n.regexp.MatchString(path[:idx]) {
				params = append(params, Param{n.segment, path[:idx]})
				return r.matchNode(c, path[idx:], params)
			}
		}

		if n.regexp.MatchString(path) {
			params = append(params, Param{n.segment, path})
			return n, params
		}
	}

	return nil, params
}

func (r *Router) Match(method, path string) (*Route, Params) {
	cn, ok := r.trees[method]
	if !ok {
		return nil, nil
	}

	params := make(Params, 0, strings.Count(path, "/"))
	for _, n := range cn.children {
		newN, newParams := r.matchNode(n, path, params)
		if newN != nil {
			return newN.route, newParams
		}
	}

	return nil, nil
}

func (r *Router) addFunc(methods []string, path string, function interface{}, handlers []Handler) {
	v := reflect.ValueOf(function)
    t := v.Type()

	var kind RouteKind

	if t.NumIn() == 0 {
		kind = FuncRoute
	} else if t.NumIn() == 1 {
		if t.In(0) == reflect.TypeOf(&Context{}) {
			kind = FuncCtxRoute
		} else if t.In(0) == reflect.TypeOf(&http.Request{}) {
			kind = FuncReqRoute
		} else if t.In(0).Kind() == reflect.Interface &&
			t.In(0).Name() == "ResponseWriter" &&
			t.In(0).PkgPath() == "net/http" {
			kind = FuncRepRoute
		} else {
			panic(fmt.Sprintln("Invalid function type", methods, path, function))
		}
	} else if t.NumIn() == 2 &&
		t.In(0).Kind() == reflect.Interface &&
		t.In(0).Name() == "ResponseWriter" &&
		t.In(0).PkgPath() == "net/http" &&
		t.In(1) == reflect.TypeOf(&http.Request{}) {
		kind = FuncRepReqRoute
	} else {
		panic(fmt.Sprintln("Invalid function type", methods, path, function))
	}

	route := &Route{
		kind: kind,
		method: v,
		handlers: handlers,
	}
	for _, m := range methods {
		r.addRoute(m, path, route)
	}
}

func (r *Router) addStruct(methods map[string]string, path string, structPtr interface{}, handlers []Handler) {
	v := reflect.ValueOf(structPtr)
	t := v.Type()

	for verb, name := range methods {
		method, ok := t.MethodByName(name)
		if !ok {
			method, ok = t.MethodByName("Any")
		}

		if !ok {
			continue
		}

		var kind RouteKind

		mt := method.Type
		if mt.NumIn() == 1 {
			kind = FuncRoute
		} else if mt.NumIn() == 2 {
			if mt.In(1) == reflect.TypeOf(&Context{}) {
				kind = FuncCtxRoute
			} else if t.In(1) == reflect.TypeOf(&http.Request{}) {
				kind = FuncReqRoute
			} else if t.In(1).Kind() == reflect.Interface &&
				t.In(1).Name() == "ResponseWriter" &&
				t.In(1).PkgPath() == "net/http" {
				kind = FuncRepRoute
			} else {
				panic(fmt.Sprintln("Invalid function type", methods, path, mt))
			}
		} else if t.NumIn() == 3 &&
			t.In(1).Kind() == reflect.Interface &&
			t.In(1).Name() == "ResponseWriter" &&
			t.In(1).PkgPath() == "net/http" &&
			t.In(2) == reflect.TypeOf(&http.Request{}) {
			kind = FuncRepReqRoute
		} else {
			panic(fmt.Sprintln("Invalid function type", methods, path, mt))
		}

		f := func(in []reflect.Value) []reflect.Value {
			in = append([]reflect.Value{v}, in...)
			return method.Func.Call(in)
		}
		r.addRoute(verb, path, &Route{
			kind: kind,
			method: reflect.ValueOf(f),
			handlers: handlers,
		})
	}
}

func removeTrailingSlash(path string) string {
	path = strings.TrimRight(path, "/")
	if path == "" {
		path = "/"
	}
	return path
}

func (r *Router) Register(methods interface{}, path string, target interface{}, handlers ...Handler) {
	assert(path[0] == '/', "Path must begin with '/'")
	// path = removeTrailingSlash(path)

	var ms []string
	switch methods.(type) {
	case string:
		ms = []string{methods.(string)}
	case []string:
		ms = methods.([]string)
	default:
		panic("Invalid HTTP methods")
	}

	v := reflect.ValueOf(target)

	if v.Kind() == reflect.Func {
		r.addFunc(ms, path, target, handlers)
	} else if v.Kind() == reflect.Ptr && v.Elem().Kind() == reflect.Struct {
		var mm = make(map[string]string)
		for _, m := range ms {
			mm[m] = strings.Title(strings.ToLower(m))
		}
		r.addStruct(mm, path, target, handlers)
	} else {
		panic("Invalid route handler")
	}
}
