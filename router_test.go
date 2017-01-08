package mel

import (
	"testing"
	"regexp"
	"log"
)

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
	for path, expect := range parsedResult {
		result := parsePath(path)
		if len(expect) != len(result) {
			t.Fatalf("%v 's result %v is not equal %v", path, expect, result)
		}
		for i := 0; i < len(expect); i++ {
			if expect[i].segment != result[i].segment ||
				expect[i].kind != result[i].kind {
				t.Fatalf("%v 's %d result %v is not equal %v", path, i, expect[i], result[i])
			}

			if expect[i].kind != regexNode {
				if expect[i].regexp != nil {
					t.Fatalf("%v 's %d result %v is not equal %v", path, i, expect[i], result[i])
				}
			} else {
				if expect[i].regexp == nil {
					t.Fatalf("%v 's %d result %v is not equal %v", path, i, expect[i], result[i])
				}
			}
		}
	}
}

func TestAddRoutes(t *testing.T) {
	router := NewRouter()
	for path := range parsedResult {
		router.addRoute("GET", path, nil)
	}
	// router.PrintTrees()
}

type MatchResult struct {
	path   string
	match  bool
	params Params
}

var (
	matchResults = map[string][]MatchResult{
		"/": {
			{"/", true, Params{}},
			{"/s", false, Params{}},
			{"/123", false, Params{}},
		},
		"/ss/tt": {
			{"/ss/tt", true, Params{}},
			{"/s", false, Params{}},
			{"/ss", false, Params{}},
		},
		"/:name": {
			{"/s", true, Params{{":name", "s"}}},
			{"/", false, Params{}},
			{"/123/s", false, Params{}},
		},
		"/:name1/:name2/:name3": {
			{"/1/2/3", true, Params{{":name1", "1"}, {":name2", "2"}, {":name3", "3"}}},
			{"/1/2", false, Params{}},
			{"/1/2/3/", false, Params{}},
		},
		"/*name": {
			{"/s", true, Params{{"*name", "s"}}},
			{"/123/s", true, Params{{"*name", "123/s"}}},
			{"/", false, Params{}},
		},
		"/(*name)ssss": {
			{"/sssss", true, Params{{"*name", "s"}}},
			{"/ssss", true, Params{{"*name", ""}}},
			{"/123/ssss", true, Params{{"*name", "123/"}}},
			{"/", false, Params{}},
			{"/ss", false, Params{}},
		},
		"/111(*name)ssss": {
			{"/111sssss", true, Params{{"*name", "s"}}},
			{"/111/123/ssss", true, Params{{"*name", "/123/"}}},
			{"/", false, Params{}},
			{"/ss", false, Params{}},
		},
		"/(:name[0-9]+)": {
			{"/123", true, Params{{":name", "123"}}},
			{"/sss", false, Params{}},
		},
		"/ss(:name[0-9]+)": {
			{"/ss123", true, Params{{":name", "123"}}},
			{"/sss", false, Params{}},
		},
		"/ss(:name[0-9]+)tt": {
			{"/ss123tt", true, Params{{":name", "123"}}},
			{"/ss123ttt", false, Params{}},
			{"/sss", false, Params{}},
		},
		"/:name1-(:name2[0-9]+)": {
			{"/ss-123", true, Params{{":name1", "ss"}, {":name2", "123"}}},
			{"/sss", false, Params{}},
		},
		"/(:name1)00(:name2[0-9]+)": {
			{"/ss00123", true, Params{{":name1", "ss"}, {":name2", "123"}}},
			{"/sss", false, Params{}},
		},
		"/(:name1)!(:name2[0-9]+)!(:name3.*)": {
			{"/ss!123!456", true, Params{{":name1", "ss"}, {":name2", "123"}, {":name3", "456"}}},
			{"/sss", false, Params{}},
		},
		"/web/content/(:id3)-(:unique3)/(:filename)": {
			{"/web/content/36-0420888/website.assets_frontend.0.css", true, Params{{":id3", "36"}, {":unique3", "0420888"}, {":filename", "website.assets_frontend.0.css"}}},
		},
	}
)

func TestRoutingSingle(t *testing.T) {
	for path, expects := range matchResults {
		router := NewRouter()
		router.Register("GET", path, func () {})
		// router.PrintTrees()

		for _, expect := range expects {
			handler, params := router.Match("GET", expect.path)
			if expect.match {
				if handler == nil {
					t.Fatal(path, expect, "handler should not be nil")
				}
				for i, v := range params {
					if expect.params[i].Key != v.Key {
						t.Fatal(path, expect, "param name", v, "is not equal to", expect.params[i])
					}
					if expect.params[i].Value != v.Value {
						t.Fatal(path, expect, "param value", v, "is not equal to", expect.params[i])
					}
				}
			} else {
				if handler != nil {
					t.Fatal(path, expect, "handler should be nil")
				}
			}
		}
	}
}

type MultipleMatchResult struct {
	paths []string
	results []MatchResult
}

var (
	multipleMatchResult = []MultipleMatchResult{
		{
			[]string{"/"},
			[]MatchResult{
				{"/", true, Params{}},
				{"/s", false, Params{}},
				{"/123", false, Params{}},
			},
		},

		{
			[]string{"/admin", "/:name"},
			[]MatchResult{
				{"/", false, Params{}},
				{"/admin", true, Params{}},
				{"/s", true, Params{{":name", "s"}}},
				{"/123", true, Params{{":name", "123"}}},
			},
		},

		{
			[]string{"/:name", "/admin"},
			[]MatchResult{
				{"/", false, Params{}},
				{"/admin", true, Params{}},
				{"/s", true, Params{{":name", "s"}}},
				{"/123", true, Params{{":name", "123"}}},
			},
		},

		{
			[]string{"/admin", "/*name"},
			[]MatchResult{
				{"/", false, Params{}},
				{"/admin", true, Params{}},
				{"/s", true, Params{{"*name", "s"}}},
				{"/123", true, Params{{"*name", "123"}}},
			},
		},

		{
			[]string{"/*name", "/admin"},
			[]MatchResult{
				{"/", false, Params{}},
				{"/admin", true, Params{}},
				{"/s", true, Params{{"*name", "s"}}},
				{"/123", true, Params{{"*name", "123"}}},
			},
		},

		{
			[]string{"/*name", "/:name"},
			[]MatchResult{
				{"/", false, Params{}},
				{"/s", true, Params{{"*name", "s"}}},
				{"/123", true, Params{{"*name", "123"}}},
			},
		},

		{
			[]string{"/:name", "/*name"},
			[]MatchResult{
				{"/", false, Params{}},
				{"/s", true, Params{{":name", "s"}}},
				{"/123", true, Params{{":name", "123"}}},
				{"/123/1", true, Params{{"*name", "123/1"}}},
			},
		},

		{
			[]string{"/*name", "/*name/123"},
			[]MatchResult{
				{"/", false, Params{}},
				{"/123", true, Params{{"*name", "123"}}},
				{"/s", true, Params{{"*name", "s"}}},
				{"/abc/123", true, Params{{"*name", "abc"}}},
				{"/name1/name2/123", true, Params{{"*name", "name1/name2"}}},
			},
		},

		{
			[]string{"/admin/ui", "/*name", "/:name"},
			[]MatchResult{
				{"/", false, Params{}},
				{"/admin/ui", true, Params{}},
				{"/s", true, Params{{"*name", "s"}}},
				{"/123", true, Params{{"*name", "123"}}},
			},
		},

		{
			[]string{"/(:id[0-9]+)", "/(:id[0-9]+)/edit", "/(:id[0-9]+)/del"},
			[]MatchResult{
				{"/1", true, Params{{":id", "1"}}},
				{"/admin/ui", false, Params{}},
				{"/2/edit", true, Params{{":id", "2"}}},
				{"/3/del", true, Params{{":id", "3"}}},
			},
		},

		{
			[]string{"/admin/ui", "/:name1/:name2"},
			[]MatchResult{
				{"/", false, Params{}},
				{"/admin/ui", true, Params{}},
				{"/s", false, Params{}},
				{"/admin/ui2", true, Params{{":name1", "admin"}, {":name2", "ui2"}}},
				{"/123/s", true, Params{{":name1", "123"}, {":name2", "s"}}},
			},
		},

		{
			[]string{"/(:name1)/(:name1)abc/",
				"/(:name1)/(:name1)abc(:name2)abc/",
				"/(:name1)/abc(:name1)abc(:name2)abc/",
				"/(:name1)/(:name1)/"},
			[]MatchResult{
				{"/abc/abc123abc123abc", true, Params{
					{":name1", "abc"},
					{":name1", "123"},
					{":name2", "123"},
				},
				},
			},
		},
	}
)

func TestRoutingMultiple(t *testing.T) {
	log.SetFlags(log.Lshortfile)
	for _, item := range multipleMatchResult {
		router := NewRouter()
		for _, path := range item.paths {
			router.Register("GET", path, func() {})
		}
		router.PrintTrees()

		for _, expect := range item.results {
			handler, params := router.Match("GET", expect.path)
			if expect.match {
				if handler == nil {
					t.Fatal(item.paths, expect, "handler should not be nil")
				}

				if len(expect.params) != len(params) {
					t.Fatal(item.paths, expect, "params", params, "is not equal to", expect.params)
				}
				for i, v := range params {
					if expect.params[i].Key != v.Key {
						t.Fatal(item.paths, expect, "params name", v, "is not equal to", expect.params[i])
					}
					if expect.params[i].Value != v.Value {
						t.Fatal(item.paths, expect, "params value", v, "is not equal to", expect.params[i])
					}
				}
			} else {
				if handler != nil {
					t.Fatal(item.paths, expect, "handler should be nil")
				}
			}
		}
	}
}

