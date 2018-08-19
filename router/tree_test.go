// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org/router source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package router

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"aahframework.org/ahttp"
	"github.com/stretchr/testify/assert"
)

func TestTreeBasicUseCase(t *testing.T) {
	testcases := []struct {
		route        string
		result       *Route
		resultParams ahttp.PathParams
	}{
		{route: "/hi", result: &Route{Path: "/hi"}},
		{route: "/contact", result: &Route{Path: "/contact"}},
		{route: "/co", result: &Route{Path: "/co"}},
		{route: "/c", result: &Route{Path: "/c"}},
		{route: "/a", result: &Route{Path: "/a"}},
		{route: "/ab", result: &Route{Path: "/ab"}},
		{route: "/doc/", result: &Route{Path: "/doc/"}},
		{route: "/doc/go_faq.html", result: &Route{Path: "/doc/go_faq.html"}},
		{route: "/doc/go1.html", result: &Route{Path: "/doc/go1.html"}},
		{route: "/α", result: &Route{Path: "/α"}},
		{route: "/β", result: &Route{Path: "/β"}},
	}

	tt := newTree()

	for _, tc := range testcases {
		err := tt.add(tc.route, tc.result)
		assert.Nil(t, err, "unexpected")
	}
	tt.root.inferwnode()

	for _, tc := range testcases {
		v, p, _ := tt.lookup(tc.route)
		assert.Nil(t, p)
		assert.NotNil(t, v)
		assert.Equal(t, tc.result, v)
	}

	var buf bytes.Buffer
	tt.root.printTree(&buf, 0)
}

func TestTreeRouteParameters(t *testing.T) {
	routes := []string{
		"/cmd/welcome",
		"/cmd/welcome/:sub",
		"/cmd/:tool/:sub",
		"/cmd/:tool/",
		"/src/*filepath",
		"/assets/*filepath",
		"/search/",
		"/search/:query",
		"/user_:name/about",
		"/user_:name",
		"/files/:dir",
		"/files/:dir/*filepath",
		"/doc/",
		"/doc/go_faq.html",
		"/doc/go1.html",
		"/info/:user/public",
		"/info/:user/project/:project",
		"/",
	}

	tt := newTree()

	for _, route := range routes {
		err := tt.add(route, &Route{Path: route})
		assert.Nil(t, err, "unexpected")
	}

	tt.root.inferwnode()
	// tt.root.printTree(os.Stdout, 0)

	testcases := []struct {
		route        string
		result       *Route
		resultParams ahttp.URLParams
	}{
		{route: "/doc/go1.html", result: &Route{Path: "/doc/go1.html"}},
		{route: "/cmd/aahcli"},
		{route: "/search/someth!ng+in+ünìcodé/"},
		{
			route:        "/info/:JeevaM/project/*aahframework",
			result:       &Route{Path: "/info/:user/project/:project"},
			resultParams: ahttp.URLParams{{Key: "user", Value: ":JeevaM"}, {Key: "project", Value: "*aahframework"}},
		},
		{
			route:        "/cmd/supertool/aah",
			result:       &Route{Path: "/cmd/:tool/:sub"},
			resultParams: ahttp.URLParams{{Key: "tool", Value: "supertool"}, {Key: "sub", Value: "aah"}},
		},
		{
			route:        "/files/jeeva/path/to/jeeva/file",
			result:       &Route{Path: "/files/:dir/*filepath"},
			resultParams: ahttp.URLParams{{Key: "dir", Value: "jeeva"}, {Key: "filepath", Value: "path/to/jeeva/file"}},
		},
		{
			route:        "/user_welcome2/about",
			result:       &Route{Path: "/user_:name/about"},
			resultParams: ahttp.URLParams{{Key: "name", Value: "welcome2"}},
		},
		{
			route:        "/cmd/aahcli/",
			result:       &Route{Path: "/cmd/:tool/"},
			resultParams: ahttp.URLParams{{Key: "tool", Value: "aahcli"}},
		},
		{
			route:        "/search/someth!ng+in+ünìcodé",
			result:       &Route{Path: "/search/:query"},
			resultParams: ahttp.URLParams{{Key: "query", Value: "someth!ng+in+ünìcodé"}},
		},
		{
			route:        "/files/js/inc/framework.js",
			result:       &Route{Path: "/files/:dir/*filepath"},
			resultParams: ahttp.URLParams{{Key: "dir", Value: "js"}, {Key: "filepath", Value: "inc/framework.js"}},
		},
		{
			route:        "/info/gordon/public",
			result:       &Route{Path: "/info/:user/public"},
			resultParams: ahttp.URLParams{{Key: "user", Value: "gordon"}},
		},
		{
			route:        "/info/gordon/project/go",
			result:       &Route{Path: "/info/:user/project/:project"},
			resultParams: ahttp.URLParams{{Key: "user", Value: "gordon"}, {Key: "project", Value: "go"}},
		},
		{
			route:        "/src/path/to/source/file.go",
			result:       &Route{Path: "/src/*filepath"},
			resultParams: ahttp.URLParams{{Key: "filepath", Value: "path/to/source/file.go"}},
		},
	}

	for _, tc := range testcases {
		v, p, _ := tt.lookup(tc.route)
		assert.Equal(t, tc.result, v)
		assert.Equal(t, tc.resultParams, p)
	}

	var buf bytes.Buffer
	tt.root.printTree(&buf, 0)

	// Enable trailing slash
	tt.tralingSlash = true
	trailingRoutes := []string{
		"/cmd/aahcli",
		"/search",
		"/doc",
		"/cmd/welcome/trail2/",
		"/files/js/",
	}
	for _, route := range trailingRoutes {
		v, p, rts := tt.lookup(route)
		assert.Nil(t, v)
		assert.Nil(t, p)
		assert.True(t, rts)
	}

}

func TestTreePyramidHierarchyURL(t *testing.T) {
	tt := &tree{root: new(node)}

	routes := []string{
		"/country/:country_id/city/:city_id/district/:district_id/edit",
		"/country",
		"/country/create",
		"/country/create/",
		"/country/:country_id",
		"/country/:country_id/edit",
		"/country/:country_id/city",
		"/country/:country_id/city/create",
		"/country/:country_id/city/:city_id/",
		"/country/:country_id/city/:city_id",
		"/country/:country_id/city/:city_id/edit",
		"/country/:country_id/city/:city_id/district",
		"/country/:country_id/city/:city_id/district/create",
		"/country/:country_id/city/:city_id/district/:district_id",
	}

	for _, route := range routes {
		err := tt.add(route, &Route{Path: route})
		assert.Nil(t, err, "unexpected")
	}
	tt.root.inferwnode()

	testcases := []struct {
		route        string
		result       *Route
		resultParams ahttp.URLParams
	}{
		{route: "/country", result: &Route{Path: "/country"}},
		{route: "/country/create", result: &Route{Path: "/country/create"}},
		{
			route:        "/country/USA",
			result:       &Route{Path: "/country/:country_id"},
			resultParams: ahttp.URLParams{{Key: "country_id", Value: "USA"}},
		},
		{
			route:        "/country/India",
			result:       &Route{Path: "/country/:country_id"},
			resultParams: ahttp.URLParams{{Key: "country_id", Value: "India"}},
		},
		{
			route:        "/country/India/edit",
			result:       &Route{Path: "/country/:country_id/edit"},
			resultParams: ahttp.URLParams{{Key: "country_id", Value: "India"}},
		},
		{
			route:        "/country/India/city/create",
			result:       &Route{Path: "/country/:country_id/city/create"},
			resultParams: ahttp.URLParams{{Key: "country_id", Value: "India"}},
		},
		{
			route:        "/country/USA/city/3645363/district",
			result:       &Route{Path: "/country/:country_id/city/:city_id/district"},
			resultParams: ahttp.URLParams{{Key: "country_id", Value: "USA"}, {Key: "city_id", Value: "3645363"}},
		},
		{
			route:        "/country/India/city/97634643/district",
			result:       &Route{Path: "/country/:country_id/city/:city_id/district"},
			resultParams: ahttp.URLParams{{Key: "country_id", Value: "India"}, {Key: "city_id", Value: "97634643"}},
		},
		{
			route:        "/country/USA/city/3645363/district/029832/edit",
			result:       &Route{Path: "/country/:country_id/city/:city_id/district/:district_id/edit"},
			resultParams: ahttp.URLParams{{Key: "country_id", Value: "USA"}, {Key: "city_id", Value: "3645363"}, {Key: "district_id", Value: "029832"}},
		},
	}

	for _, tc := range testcases {
		v, p, _ := tt.lookup(tc.route)
		assert.Equal(t, tc.result, v)
		assert.Equal(t, tc.resultParams, p)
	}

	// Enable traling slash
	tt.tralingSlash = true
	v, p, _ := tt.lookup("/country/create/")
	assert.Equal(t, &Route{Path: "/country/create/"}, v)
	assert.Nil(t, p)

	v, p, _ = tt.lookup("/country/USA/city/3645363/")
	assert.Equal(t, &Route{Path: "/country/:country_id/city/:city_id/"}, v)
	assert.Equal(t, ahttp.URLParams{{Key: "country_id", Value: "USA"}, {Key: "city_id", Value: "3645363"}}, p)

	var buf bytes.Buffer
	tt.root.printTree(&buf, 0)
}

func TestTreeRoutesAddErrors(t *testing.T) {
	testcases := []struct {
		route string
		err   error
	}{
		{route: "/cmd/:tool/:sub"},
		{route: "/assets/*filepath"},
		{route: "/:id"},
		{route: "/cmd/:tool/:sub", err: errNodeExists},
		{
			route: "/cmd/*tool",
			err:   errors.New("aah/router: parameter based edge already exists[/cmd/:tool...] new[/cmd/*tool...]"),
		},
		{
			route: "/cmd/:tool2/:sub",
			err:   errors.New("aah/router: parameter based edge already exists[/cmd/:tool...] new[/cmd/:tool2...]"),
		},
		{
			route: "/cmd/:tool2",
			err:   errors.New("aah/router: parameter based edge already exists[/cmd/:tool...] new[/cmd/:tool2...]"),
		},
		{
			route: "/cmd/:tool2/",
			err:   errors.New("aah/router: parameter based edge already exists[/cmd/:tool...] new[/cmd/:tool2...]"),
		},
		{
			route: "/cmd/:tool/:sub2",
			err:   errors.New("aah/router: parameter based edge already exists[/cmd/:/:sub...] new[/cmd/:/:sub2...]"),
		},

		{
			route: "/assets/*filepath2",
			err:   errors.New("aah/router: parameter based edge already exists[/assets/*filepath...] new[/assets/*filepath2...]"),
		},
		{
			route: "/path/*file/sub",
			err:   errors.New("incorrect use of wildcard URL param [/path/*file/sub]. It should come as last param [/path/*file]"),
		},
		{
			route: "/*filepath",
			err:   errors.New("aah/router: parameter based edge already exists[/:id...] new[/*filepath...]"),
		},
	}

	tt := newTree()
	for _, tc := range testcases {
		err := tt.add(tc.route, &Route{Path: tc.route})
		assert.Equal(t, tc.err, err)
	}

	tt = newTree()
	err := tt.add("/*filepath", &Route{Path: "/*filepath"})
	assert.Equal(t, nil, err)
	err = tt.add("/:id", &Route{Path: "/:id"})
	assert.Equal(t, errors.New("aah/router: parameter based edge already exists[/*filepath...] new[/:id...]"), err)
}

func TestTreeWildcardRoutes(t *testing.T) {
	routes := []string{
		"/static/*filepath",
		"/",
		"/hotels",
		"/hotels/:id",
		"/hotels/:id/booking",
		"/favicon.ico",
		"/settings",
		"/logout",
		"/register",
		"/static",
	}

	tt := newTree()
	tt.tralingSlash = true
	for _, route := range routes {
		err := tt.add(route, &Route{Path: route})
		assert.Nil(t, err, "unexpected")
	}

	tt.root.inferwnode()

	searchRoutes := []string{
		"/",
		"/hotels/12345/booking",
		"/favicon.ico",
		"/static/img/aahframework.png",
		"/static",
		"/static/",
	}

	for _, r := range searchRoutes {
		v, p, rts := tt.lookup(r)
		if rts {
			assert.Nil(t, v)
			assert.Nil(t, p)
		} else {
			assert.NotNil(t, v)
		}
	}
}

func TestTreeRouteNotFound(t *testing.T) {
	routes := []string{
		"/country/:country_id/city/:city_id/district/:district_id/edit",
		"/country",
		"/country/create",
		"/country/create/",
		"/country/:country_id",
		"/country/:country_id/edit",
		"/country/:country_id/city",
		"/country/:country_id/city/create",
		"/country/:country_id/city/:city_id/",
		"/country/:country_id/city/:city_id",
		"/country/:country_id/city/:city_id/edit",
		"/country/:country_id/city/:city_id/district",
		"/country/:country_id/city/:city_id/district/create",
		"/country/:country_id/city/:city_id/district/:district_id",
	}

	tt := &tree{root: new(node)}
	for _, route := range routes {
		err := tt.add(route, &Route{Path: route})
		assert.Nil(t, err, "unexpected")
	}

	tt.root.inferwnode()

	searchRoutes := []string{
		"/countries",
		"/country/creates",
		"/country/India/city/welcome/sds",
	}

	for _, r := range searchRoutes {
		v, p, _ := tt.lookup(r)
		assert.Nil(t, v)
		assert.Nil(t, p)
	}
}

func TestTreeVariousRouteTypes(t *testing.T) {
	routes := []string{
		"/cmd/:tool/:sub",
		"/cmd/vet",
		"/src/",
		"/src1/",
		"/src1/*filepath",
		"/src2*filepath",
		"/search/:query",
		"/search/invalid",
		"/user_:name",
		"/user_x",
		"/id:id",
		"/id/:id",
	}

	tt := &tree{root: new(node)}
	for _, route := range routes {
		err := tt.add(route, &Route{Path: route})
		assert.Nil(t, err, "unexpected")
	}

	tt.root.inferwnode()

	testcases := []struct {
		route        string
		result       *Route
		resultParams ahttp.URLParams
	}{
		{route: "/cmd/vet", result: &Route{Path: "/cmd/vet"}},
		{route: "/search/invalid", result: &Route{Path: "/search/invalid"}},
		{route: "/user_x", result: &Route{Path: "/user_x"}},
		{
			route:        "/id2837463463",
			result:       &Route{Path: "/id:id"},
			resultParams: ahttp.URLParams{{Key: "id", Value: "2837463463"}},
		},
		{
			route:        "/src2/welcometojungle",
			result:       &Route{Path: "/src2*filepath"},
			resultParams: ahttp.URLParams{{Key: "filepath", Value: "/welcometojungle"}},
		},
	}

	for _, tc := range testcases {
		v, p, _ := tt.lookup(tc.route)
		assert.Equal(t, tc.result, v)
		assert.Equal(t, tc.resultParams, p)
	}
}

func TestTreeEmptyParameterName(t *testing.T) {
	routes := []string{
		"/user:",
		"/user:/",
		"/cmd/:/",
		"/src/*",
	}
	tt := newTree()
	for _, route := range routes {
		err := tt.add(route, nil)
		assert.Equal(t, fmt.Errorf("aah/router: parameter name required: '%s'", route), err)
	}
}

func TestTreeDoubleParameters(t *testing.T) {
	routes := []string{
		"/:foo:bar",
		"/:foo:bar/",
		"/:foo*bar",
	}

	tt := newTree()

	for _, route := range routes {
		err := tt.add(route, nil)
		assert.Equal(t, fmt.Errorf("aah/router: only one paramter allowed in the path segment: '%s'", route), err)
	}
}

func TestTreeCasesentiveCheck(t *testing.T) {
	testcases := []struct {
		route  string
		search string
	}{
		{"/HI", "/hi"},
		{"/HI/", "/hi/"},
		{"/B", "/b"},
		{"/B/", "/b/"},
		{"/abc", "/ABC"},
		{"/abc/", "/ABC/"},
		{"/aBc", "/ABC"},
		{"/aBc/", "/ABC/"},
		{"/abC", "/ABC"},
		{"/abC/", "/ABC/"},
		{"/SEARCH/QUERY", "/search/QUERY"},
		{"/SEARCH/QUERY/", "/search/QUERY/"},
		{"/CMD/TOOL/", "/cmd/TOOL/"},
		{"/CMD/TOOL", "/cmd/TOOL"},
		{"/SRC/FILE/PATH", "/src/FILE/PATH"},
		{"/x/Y", "/x/y"},
		{"/x/Y/", "/x/y/"},
		{"/X/y", "/x/y"},
		{"/X/y/", "/x/y/"},
		{"/X/Y", "/x/y"},
		{"/X/Y/", "/x/y/"},
		{"/Y/", "/y/"},
		{"/Y", "/y"},
		{"/Y/z", "/y/z"},
		{"/Y/z/", "/y/z/"},
		{"/Y/Z", "/y/z"},
		{"/Y/Z/", "/y/z/"},
		{"/y/Z", "/y/z"},
		{"/y/Z/", "/y/z/"},
		{"/Aa", "/aa"},
		{"/Aa/", "/aa/"},
		{"/AA", "/aa"},
		{"/AA/", "/aa/"},
		{"/aA", "/aa"},
		{"/aA/", "/aa/"},
		{"/A/", "/a/"},
		{"/A", "/a"},
		{"/DOC", "/doc"},
		{"/DOC/", "/doc/"},
		{"/NO", "/no"},
		{"/DOC/GO", "/doc/go"},
		{"/π", "/Π"},
		{"/π/", "/Π/"},
	}

	for _, tc := range testcases {
		tt := newTree()
		err := tt.add(tc.route, &Route{Path: tc.route})
		assert.Nil(t, err, "unexpected")

		tt.root.inferwnode()

		v, p, _ := tt.lookup(tc.search)
		assert.NotNil(t, v)
		assert.Nil(t, p)
		assert.Equal(t, tc.route, v.Path)
	}
}

func TestCountParams(t *testing.T) {
	if countParams("/path/:param1/static/*catch-all") != 2 {
		t.Fail()
	}
	if countParams(strings.Repeat("/:param", 256)) != 255 {
		t.Fail()
	}
}

func newTree() *tree {
	t := &tree{
		root: new(node),
	}
	return t
}
