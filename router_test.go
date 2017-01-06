package mel

import "regexp"
import "testing"

var (
	parsedResult = map[string]nodes{
		"/": nodes{
			{content: "/", kind: staticNode},
		},
		"/static/css/bootstrap.css": nodes{
			{content: "/static", kind: staticNode},
			{content: "/css", kind: staticNode},
			{content: "/bootstrap.css", kind: staticNode},
		},
		"/:name": nodes{
			{content: "/", kind: staticNode},
			{content: ":name", kind: namedNode},
		},
		"/sss:name": nodes{
			{content: "/sss", kind: staticNode},
			{content: ":name", kind: namedNode},
		},
		"/(:name)": nodes{
			{content: "/", kind: staticNode},
			{content: ":name", kind: namedNode},
		},
		"/(:name)/sss": nodes{
			{content: "/", kind: staticNode},
			{content: ":name", kind: namedNode},
			{content: "/sss", kind: staticNode},
		},
		"/:name-:value": nodes{
			{content: "/", kind: staticNode},
			{content: ":name", kind: namedNode},
			{content: "-", kind: staticNode},
			{content: ":value", kind: namedNode},
		},
		"/(:name)ssss(:value)": nodes{
			{content: "/", kind: staticNode},
			{content: ":name", kind: namedNode},
			{content: "ssss", kind: staticNode},
			{content: ":value", kind: namedNode},
		},
		"/(:name[0-9]+)": nodes{
			{content: "/", kind: staticNode},
			{content: ":name", kind: regexNode, regexp: regexp.MustCompile("([0-9]+)")},
		},
		"/*name": nodes{
			{content: "/", kind: staticNode},
			{content: "*name", kind: anyNode},
		},
		"/*name/ssss": nodes{
			{content: "/", kind: staticNode},
			{content: "*name", kind: anyNode},
			{content: "/ssss", kind: staticNode},
		},
		"/(*name)ssss": nodes{
			{content: "/", kind: staticNode},
			{content: "*name", kind: anyNode},
			{content: "ssss", kind: staticNode},
		},
		"/:name-(:name2[a-z]+)": nodes{
			{content: "/", kind: staticNode},
			{content: ":name", kind: namedNode},
			{content: "-", kind: staticNode},
			{content: ":name2", kind: regexNode, regexp: regexp.MustCompile("([a-z]+)")},
		},
		"/web/content/(:id3)-(:unique3)/(:filename)": nodes{
			{content: "/web", kind: staticNode},
			{content: "/content", kind: staticNode},
			{content: "/", kind: staticNode},
			{content: ":id3", kind: namedNode},
			{content: "-", kind: staticNode},
			{content: ":unique3", kind: namedNode},
			{content: "/", kind: staticNode},
			{content: ":filename", kind: namedNode},
		},
	}
)

func TestParseNode(t *testing.T) {
	for p, r := range parsedResult {
		res := parsePath(p)
		if len(r) != len(res) {
			t.Fatalf("%v 's result %v is not equal %v", p, r, res)
		}
		for i := 0; i < len(r); i++ {
			if r[i].content != res[i].content ||
				r[i].kind != res[i].kind {
				t.Fatalf("%v 's %d result %v is not equal %v", p, i, r[i], res[i])
			}

			if r[i].kind != regexNode {
				if r[i].regexp != nil {
					t.Fatalf("%v 's %d result %v is not equal %v", p, i, r[i], res[i])
				}
			} else {
				if r[i].regexp == nil {
					t.Fatalf("%v 's %d result %v is not equal %v", p, i, r[i], res[i])
				}
			}
		}
	}
}
