package mel

import "regexp"
import "testing"

var (
	parsedResult = map[string]nodes{
		"/": {
			{segment: "/", kind: staticNode},
		},
		"/static/css/bootstrap.css": {
			{segment: "/static", kind: staticNode},
			{segment: "/css", kind: staticNode},
			{segment: "/bootstrap.css", kind: staticNode},
		},
		"/:name": {
			{segment: "/", kind: staticNode},
			{segment: ":name", kind: namedNode},
		},
		"/sss:name": {
			{segment: "/sss", kind: staticNode},
			{segment: ":name", kind: namedNode},
		},
		"/(:name)": {
			{segment: "/", kind: staticNode},
			{segment: ":name", kind: namedNode},
		},
		"/(:name)/sss": {
			{segment: "/", kind: staticNode},
			{segment: ":name", kind: namedNode},
			{segment: "/sss", kind: staticNode},
		},
		"/:name-:value": {
			{segment: "/", kind: staticNode},
			{segment: ":name", kind: namedNode},
			{segment: "-", kind: staticNode},
			{segment: ":value", kind: namedNode},
		},
		"/(:name)ssss(:value)": {
			{segment: "/", kind: staticNode},
			{segment: ":name", kind: namedNode},
			{segment: "ssss", kind: staticNode},
			{segment: ":value", kind: namedNode},
		},
		"/(:name[0-9]+)": {
			{segment: "/", kind: staticNode},
			{segment: ":name", kind: regexNode, regexp: regexp.MustCompile("([0-9]+)")},
		},
		"/*name": {
			{segment: "/", kind: staticNode},
			{segment: "*name", kind: anyNode},
		},
		"/*name/ssss": {
			{segment: "/", kind: staticNode},
			{segment: "*name", kind: anyNode},
			{segment: "/ssss", kind: staticNode},
		},
		"/(*name)ssss": {
			{segment: "/", kind: staticNode},
			{segment: "*name", kind: anyNode},
			{segment: "ssss", kind: staticNode},
		},
		"/:name-(:name2[a-z]+)": {
			{segment: "/", kind: staticNode},
			{segment: ":name", kind: namedNode},
			{segment: "-", kind: staticNode},
			{segment: ":name2", kind: regexNode, regexp: regexp.MustCompile("([a-z]+)")},
		},
		"/web/content/(:id3)-(:unique3)/(:filename)": {
			{segment: "/web", kind: staticNode},
			{segment: "/content", kind: staticNode},
			{segment: "/", kind: staticNode},
			{segment: ":id3", kind: namedNode},
			{segment: "-", kind: staticNode},
			{segment: ":unique3", kind: namedNode},
			{segment: "/", kind: staticNode},
			{segment: ":filename", kind: namedNode},
		},
	}
)

func TestParseNode(t *testing.T) {
	for path, expected := range parsedResult {
		result := parsePath(path)
		if len(expected) != len(result) {
			t.Fatalf("%v 's result %v is not equal %v", path, expected, result)
		}
		for i := 0; i < len(expected); i++ {
			if expected[i].segment != result[i].segment ||
				expected[i].kind != result[i].kind {
				t.Fatalf("%v 's %d result %v is not equal %v", path, i, expected[i], result[i])
			}

			if expected[i].kind != regexNode {
				if expected[i].regexp != nil {
					t.Fatalf("%v 's %d result %v is not equal %v", path, i, expected[i], result[i])
				}
			} else {
				if expected[i].regexp == nil {
					t.Fatalf("%v 's %d result %v is not equal %v", path, i, expected[i], result[i])
				}
			}
		}
	}
}

func TestAddRoutes(t *testing.T) {
	router := NewRouter()
	for path := range parsedResult {
		router.addRoute("GETA", path, nil)
	}
	router.printTrees()
}
