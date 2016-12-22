package mel

import (
	"bytes"
	"fmt"
	"reflect"
	"regexp"
	"sort"
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
	StructRoute                      // func (struct) <Method>()
	StructPtrRoute                   // func (*struct) <Method>()
)

type Route struct {
	kind   RouteKind
	method interface{}
}

func (r *Route) isStruct() bool {
	return r.kind == StructRoute || r.kind == StructPtrRoute
}

type Router interface {
	Route(methods interface{}, path string, handler interface{}, middlewares ...Handler)
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
	trees map[string]*node
}

func NewRouter() (r *router) {
	r = &router{
		trees: make(map[string]*node),
	}

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

func (r *router) Match(method, path string) (*Route, Params) {
	cn, ok := r.trees[method]
	if !ok {
		return nil, nil
	}


}

func (r *router) addStruct(methods map[string]string, url string, object interface{}) {
	v := reflect.ValueOf(object)
	t := v.Type().Elem()

	for verb, method := range methods {
		if m, ok :=
	}
}

func (r *router) Route()
