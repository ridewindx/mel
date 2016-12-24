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

func newRoute(kind RouteKind, method reflect.Value) *Route {
	return &Route{
		kind: kind,
		method: method,
	}
}

type Router interface {
	AddRoute(methods interface{}, path string, handler interface{}, middlewares ...Handler)
	Match(requestPath, method string) (*Route, Params)
}

type nodeKind byte

const (
	staticNode nodeKind = iota
	namedNode
	anyNode
	regexNode
)

type node struct {
	kind    nodeKind
	route   *Route
	regexp  *regexp.Regexp
	content string
	edges   edges
	path    string
}

func (n *node) equal(o *node) bool {
	if n.kind == o.kind && n.content == o.content {
		return true
	}
	return false
}

type edges []*node

func (e edges) Len() int {
	return len(e)
}

func (e edges) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

// Static route is first.
// If two static routes, content longer is first.
// Otherwise, first added, first matched.
func (e edges) Less(i, j int) bool {
	if e[i].kind == staticNode {
		if e[j].kind == staticNode {
			return len(e[i].content) > len(e[j].content)
		}
		return true
	}
	if e[j].kind == staticNode {
		return false
	}
	return i < j
}

type router struct {
    routerGroup

	trees map[string]*node
}

func NewRouter() (r *router) {
	r = &router{
        routerGroup{
			basePath: "/",
		},
		trees: make(map[string]*node),
	}

	r.routerGroup.router = r

	for _, m := range Methods {
		r.trees[m] = &node{}
	}

	return
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
					content: path[start:i],
				})
				start = i
			}
		} else if path[i] == '(' {
			bracket = 1
		} else if path[i] == ':' {
			nodes = append(nodes, &node{
				kind:    staticNode,
				content: path[start : i-bracket],
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
					content: path[start : i-len(re)],
					regexp:  regexp.MustCompile("(" + re + ")"),
				})
			} else {
				nodes = append(nodes, &node{
					kind:    namedNode,
					content: path[start:i],
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
				content: path[start : i-bracket],
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
				content: path[start:i],
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
		content: path[start:i],
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

func (r *router) addRoute(method, path string, route *Route) {
	nodes := parsePath(path)
	num := len(nodes)

	leafNode := nodes[num-1]
	leafNode.route = route
	leafNode.path = path

	if !validateNodes(nodes) {
		panic("Any non-static route should have static route successor: " + path)
	}

	p := r.trees[method]

	outer:
	for i := 0; i < num; i++ {
		for _, node := range p.edges {
			if node.equal(nodes[i]) {
				if i == num-1 {
					node.route = route
				}
				p = node
				continue outer
			}
		}

		p.edges = append(p.edges, nodes[i])
		sort.Sort(p.edges)
		p = nodes[i]
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

		fmt.Print(n.content)
		if n.route != nil {
			fmt.Print("  ", n.route.method.Type())
			fmt.Printf("  %p", n.route.method.Interface())
		}
		fmt.Println()
		printNodes(i+1, n.edges)
	}
}

func (r *router) printTrees() {
	for _, method := range Methods {
		if len(r.trees[method].edges) > 0 {
			fmt.Println(method)
			printNodes(1, r.trees[method].edges)
			fmt.Println()
		}
	}
}

func (r *router) matchNode(n *node, path string, params Params) (*node, Params) {
	if n.kind == staticNode {
		if strings.HasPrefix(path, n.content) {
			if len(path) == len(n.content) {
				return n, params
			}

			for _, c := range n.edges {
				newN, newParams := r.matchNode(c, path[len(n.content):], params)
				if newN != nil {
					return newN, newParams
				}
			}
		}
	} else if n.kind == anyNode {
		for _, c := range n.edges {
			idx := strings.LastIndex(path, c.content)
			if idx > -1 {
				params = append(params, param{n.content, path[:idx]})
				return r.matchNode(c, path[idx:], params)
			}
		}

		return n, append(params, param{n.content, path})
	} else if n.kind == namedNode {
		for _, c := range n.edges {
			idx := strings.Index(path, c.content)
			if idx > -1 {
				params = append(params, param{n.content, path[:idx]})
				return r.matchNode(c, path[idx:], params)
			}
		}

		idx := strings.IndexByte(path, '/')
		if idx == -1 {
			params = append(params, param{n.content, path})
			return n, params
		}
	} else if n.kind == regexNode {
		idx := strings.IndexByte(path, '/')
		if idx > -1 {
			if n.regexp.MatchString(path[:idx]) {
				for _, c := range n.edges {
					newN, newParams := r.matchNode(c, path[idx:], params)
					if newN != nil {
						return newN, append(Params{param{n.content, path[:idx]}}, newParams...)
					}
				}
			}

			return nil, params
		}

		for _, c := range n.edges {
			idx := strings.Index(path, c.content)
			if idx > -1 && n.regexp.MatchString(path[:idx]) {
				params = append(params, param{n.content, path[:idx]})
				return r.matchNode(c, path[idx:], params)
			}
		}

		if n.regexp.MatchString(path) {
			params = append(params, param{n.content, path})
			return n, params
		}
	}

	return nil, params
}

func (r *router) Match(method, path string) (*Route, Params) {
	cn, ok := r.trees[method]
	if !ok {
		return nil, nil
	}

	params := make(Params, 0, strings.Count(path, "/"))
	for _, n := range cn.edges {
		newN, newParams := r.matchNode(n, path, params)
		if newN != nil {
			return newN.route, newParams
		}
	}

	return nil, nil
}

func (r *router) addFunc(methods []string, path string, function interface{}, handlers []Handler) {
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
		method: function,
		handlers: handlers,
	}
	for _, m := range methods {
		r.addRoute(m, path, route)
	}
}

func (r *router) addStruct(methods map[string]string, path string, structPtr interface{}, handlers []Handler) {
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

		var f reflect.Value = func(in []reflect.Value) []reflect.Value {
			in = append([]reflect.Value{structPtr}, in...)
			return method.Func.Call(in)
		}
		r.addRoute(verb, path, &Route{
			kind: kind,
			method: f,
			handlers: handlers,
		})
	}
}

func (r *router) AddRoute(methods interface{}, path string, object interface{}, handlers ...Handler) {
	assert(path[0] == '/', "Path must begin with '/'")
	assert(len(handlers) > 0, "There must be at least one handler")

	var ms []string
	switch methods.(type) {
	case string:
		ms = []string{methods.(string)}
	case []string:
		ms = methods.([]string)
	default:
		panic("Invalid HTTP methods")
	}

	v := reflect.ValueOf(object)

	if v.Kind() == reflect.Func {
		r.addFunc(ms, path, object, handlers)
	} else if v.Kind() == reflect.Ptr && v.Elem().Kind() == reflect.Struct {
		var mm = make(map[string]string)
		for _, m := range ms {
			mm[m] = strings.Title(strings.ToLower(m))
		}
		r.addStruct(mm, path, object, handlers)
	} else {
		panic("Invalid route handler")
	}
}
