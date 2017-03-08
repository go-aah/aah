// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/router source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.
//
// Updated for aah framework purpose.
// From upstream updated as of last commit date Feb 06, 2016 git#a7a8c64.

package router

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"aahframework.org/test.v0/assert"
)

type (
	testRequests []struct {
		path       string
		nilHandler bool
		route      string
		pp         PathParams
	}

	testRoute struct {
		path     string
		conflict bool
	}
)

func TestCountParams(t *testing.T) {
	assert.Equal(t, uint8(2), countParams("/path/:param1/static/*catch-all"))

	assert.Equal(t, uint8(255), countParams(strings.Repeat("/:param", 256)))
}

func TestTreeAddAndGet(t *testing.T) {
	tree := &node{}

	routes := [...]string{
		"/hi",
		"/contact",
		"/co",
		"/c",
		"/a",
		"/ab",
		"/doc/",
		"/doc/go_faq.html",
		"/doc/go1.html",
		"/α",
		"/β",
	}
	for _, route := range routes {
		err := tree.add(route, route)
		assert.FailOnError(t, err, "unexpected error")
	}

	//printChildren(tree, "")

	checkRequests(t, tree, testRequests{
		{"/a", false, "/a", nil},
		{"/", true, "", nil},
		{"/hi", false, "/hi", nil},
		{"/contact", false, "/contact", nil},
		{"/co", false, "/co", nil},
		{"/con", true, "", nil},  // key mismatch
		{"/cona", true, "", nil}, // key mismatch
		{"/no", true, "", nil},   // no matching child
		{"/ab", false, "/ab", nil},
		{"/α", false, "/α", nil},
		{"/β", false, "/β", nil},
	})

	checkPriorities(t, tree)
	checkMaxParams(t, tree)
}

func TestTreeWildcard(t *testing.T) {
	tree := &node{}

	routes := [...]string{
		"/",
		"/cmd/:tool/:sub",
		"/cmd/:tool/",
		"/src/*filepath",
		"/search/",
		"/search/:query",
		"/user_:name",
		"/user_:name/about",
		"/files/:dir/*filepath",
		"/doc/",
		"/doc/go_faq.html",
		"/doc/go1.html",
		"/info/:user/public",
		"/info/:user/project/:project",
	}
	for _, route := range routes {
		_ = tree.add(route, route)
	}

	//printChildren(tree, "")

	checkRequests(t, tree, testRequests{
		{"/", false, "/", nil},
		{"/cmd/test/", false, "/cmd/:tool/", PathParams{PathParam{"tool", "test"}}},
		{"/cmd/test", true, "", PathParams{PathParam{"tool", "test"}}},
		{"/cmd/test/3", false, "/cmd/:tool/:sub", PathParams{PathParam{"tool", "test"}, PathParam{"sub", "3"}}},
		{"/src/", false, "/src/*filepath", PathParams{PathParam{"filepath", "/"}}},
		{"/src/some/file.png", false, "/src/*filepath", PathParams{PathParam{"filepath", "/some/file.png"}}},
		{"/search/", false, "/search/", nil},
		{"/search/someth!ng+in+ünìcodé", false, "/search/:query", PathParams{PathParam{"query", "someth!ng+in+ünìcodé"}}},
		{"/search/someth!ng+in+ünìcodé/", true, "", PathParams{PathParam{"query", "someth!ng+in+ünìcodé"}}},
		{"/user_gopher", false, "/user_:name", PathParams{PathParam{"name", "gopher"}}},
		{"/user_gopher/about", false, "/user_:name/about", PathParams{PathParam{"name", "gopher"}}},
		{"/files/js/inc/framework.js", false, "/files/:dir/*filepath", PathParams{PathParam{"dir", "js"}, PathParam{"filepath", "/inc/framework.js"}}},
		{"/info/gordon/public", false, "/info/:user/public", PathParams{PathParam{"user", "gordon"}}},
		{"/info/gordon/project/go", false, "/info/:user/project/:project", PathParams{PathParam{"user", "gordon"}, PathParam{"project", "go"}}},
	})

	checkPriorities(t, tree)
	checkMaxParams(t, tree)
}

func testRoutes(t *testing.T, routes []testRoute) {
	tree := &node{}

	for _, route := range routes {
		err := tree.add(route.path, nil)
		if route.conflict {
			assert.NotNilf(t, err, "no error for conflicting route '%s'", route.path)
		} else {
			assert.Nilf(t, err, "unexpected error for route '%s': %v", route.path, err)
		}
	}

	//printChildren(tree, "")
}

func TestTreeWildcardConflict(t *testing.T) {
	routes := []testRoute{
		{"/cmd/:tool/:sub", false},
		{"/cmd/vet", true},
		{"/src/*filepath", false},
		{"/src/*filepathx", true},
		{"/src/", true},
		{"/src1/", false},
		{"/src1/*filepath", true},
		{"/src2*filepath", true},
		{"/search/:query", false},
		{"/search/invalid", true},
		{"/user_:name", false},
		{"/user_x", true},
		{"/user_:name", false},
		{"/id:id", false},
		{"/id/:id", true},
	}
	testRoutes(t, routes)
}

func TestTreeChildConflict(t *testing.T) {
	routes := []testRoute{
		{"/cmd/vet", false},
		{"/cmd/:tool/:sub", true},
		{"/src/AUTHORS", false},
		{"/src/*filepath", true},
		{"/user_x", false},
		{"/user_:name", true},
		{"/id/:id", false},
		{"/id:id", true},
		{"/:id", true},
		{"/*filepath", true},
	}
	testRoutes(t, routes)
}

func TestTreeDupliatePath(t *testing.T) {
	tree := &node{}

	routes := [...]string{
		"/",
		"/doc/",
		"/src/*filepath",
		"/search/:query",
		"/user_:name",
	}
	for _, route := range routes {
		err := tree.add(route, route)
		assert.FailNowOnErrorf(t, err, "error inserting route '%s': %v", route, err)

		// Add again
		err = tree.add(route, nil)
		assert.NotNilf(t, err, "no error while inserting duplicate route '%s'", route)
	}

	// printChildren(tree, "")

	checkRequests(t, tree, testRequests{
		{"/", false, "/", nil},
		{"/doc/", false, "/doc/", nil},
		{"/src/some/file.png", false, "/src/*filepath", PathParams{PathParam{"filepath", "/some/file.png"}}},
		{"/search/someth!ng+in+ünìcodé", false, "/search/:query", PathParams{PathParam{"query", "someth!ng+in+ünìcodé"}}},
		{"/user_gopher", false, "/user_:name", PathParams{PathParam{"name", "gopher"}}},
	})
}

func TestEmptyWildcardName(t *testing.T) {
	tree := &node{}

	routes := [...]string{
		"/user:",
		"/user:/",
		"/cmd/:/",
		"/src/*",
	}
	for _, route := range routes {
		err := tree.add(route, nil)
		assert.NotNilf(t, err, "no error while inserting route with empty wildcard name '%s'", route)
	}
}

func TestTreeCatchAllConflict(t *testing.T) {
	routes := []testRoute{
		{"/src/*filepath/x", true},
		{"/src2/", false},
		{"/src2/*filepath/x", true},
	}
	testRoutes(t, routes)
}

func TestTreeCatchAllConflictRoot(t *testing.T) {
	routes := []testRoute{
		{"/", false},
		{"/*filepath", true},
	}
	testRoutes(t, routes)
}

func TestTreeDoubleWildcard(t *testing.T) {
	const panicMsg = "only one wildcard per path segment is allowed"

	routes := [...]string{
		"/:foo:bar",
		"/:foo:bar/",
		"/:foo*bar",
	}

	for _, route := range routes {
		tree := &node{}
		err := tree.add(route, nil)
		if !strings.HasPrefix(err.Error(), panicMsg) {
			t.Fatalf(`"Expected error "%s" for route '%s', got "%v"`, panicMsg, route, err)
		}
	}
}

func TestTreeTrailingSlashRedirect(t *testing.T) {
	tree := &node{}

	routes := [...]string{
		"/hi",
		"/b/",
		"/search/:query",
		"/cmd/:tool/",
		"/src/*filepath",
		"/x",
		"/x/y",
		"/y/",
		"/y/z",
		"/0/:id",
		"/0/:id/1",
		"/1/:id/",
		"/1/:id/2",
		"/aa",
		"/a/",
		"/admin",
		"/admin/:category",
		"/admin/:category/:page",
		"/doc",
		"/doc/go_faq.html",
		"/doc/go1.html",
		"/no/a",
		"/no/b",
		"/api/hello/:name",
	}
	for _, route := range routes {
		err := tree.add(route, route)
		assert.FailNowOnErrorf(t, err, "error inserting route '%s': %v", route, err)
	}

	//printChildren(tree, "")

	tsrRoutes := [...]string{
		"/hi/",
		"/b",
		"/search/gopher/",
		"/cmd/vet",
		"/src",
		"/x/",
		"/y",
		"/0/go/",
		"/1/go",
		"/a",
		"/admin/",
		"/admin/config/",
		"/admin/config/permissions/",
		"/doc/",
	}
	for _, route := range tsrRoutes {
		handler, _, tsr, _ := tree.find(route)
		if handler != nil {
			t.Fatalf("non-nil handler for TSR route '%s", route)
		} else if !tsr {
			t.Errorf("expected TSR recommendation for route '%s'", route)
		}
	}

	noTsrRoutes := [...]string{
		"/",
		"/no",
		"/no/",
		"/_",
		"/_/",
		"/api/world/abc",
	}

	for _, route := range noTsrRoutes {
		handler, _, tsr, _ := tree.find(route)
		if handler != nil {
			t.Fatalf("non-nil handler for No-TSR route '%s", route)
		} else if tsr {
			t.Errorf("expected no TSR recommendation for route '%s'", route)
		}
	}
}

func TestTreeRootTrailingSlashRedirect(t *testing.T) {
	tree := &node{}

	err := tree.add("/:test", "/:test")
	assert.FailNowOnError(t, err, "error inserting test route")

	handler, _, tsr, _ := tree.find("/")
	if handler != nil {
		t.Fatalf("non-nil handler")
	} else if tsr {
		t.Errorf("expected no TSR recommendation")
	}
}

func TestTreeFindCaseInsensitivePath(t *testing.T) {
	tree := &node{}

	routes := [...]string{
		"/hi",
		"/b/",
		"/ABC/",
		"/search/:query",
		"/cmd/:tool/",
		"/src/*filepath",
		"/x",
		"/x/y",
		"/y/",
		"/y/z",
		"/0/:id",
		"/0/:id/1",
		"/1/:id/",
		"/1/:id/2",
		"/aa",
		"/a/",
		"/doc",
		"/doc/go_faq.html",
		"/doc/go1.html",
		"/doc/go/away",
		"/no/a",
		"/no/b",
		"/Π",
		"/u/apfêl/",
		"/u/äpfêl/",
		"/u/öpfêl",
		"/v/Äpfêl/",
		"/v/Öpfêl",
		"/w/♬",  // 3 byte
		"/w/♭/", // 3 byte, last byte differs
		"/w/𠜎",  // 4 byte
		"/w/𠜏/", // 4 byte
	}

	for _, route := range routes {
		err := tree.add(route, route)
		assert.FailNowOnErrorf(t, err, "error inserting route '%s': %v", route, err)
	}

	// Check out == in for all registered routes
	// With fixTrailingSlash = true
	for _, route := range routes {
		out, found, err := tree.findCaseInsensitive(route, true)
		assert.Nil(t, err)
		assert.Truef(t, found, "Route '%s' not found!", route)
		assert.Equalf(t, route, string(out), "Wrong result for route '%s': %s", route, string(out))
	}

	// With fixTrailingSlash = false
	for _, route := range routes {
		out, found, err := tree.findCaseInsensitive(route, false)
		assert.Nil(t, err)
		assert.Truef(t, found, "Route '%s' not found!", route)
		assert.Equalf(t, route, string(out), "Wrong result for route '%s': %s", route, string(out))
	}

	tests := []struct {
		in    string
		out   string
		found bool
		slash bool
	}{
		{"/HI", "/hi", true, false},
		{"/HI/", "/hi", true, true},
		{"/B", "/b/", true, true},
		{"/B/", "/b/", true, false},
		{"/abc", "/ABC/", true, true},
		{"/abc/", "/ABC/", true, false},
		{"/aBc", "/ABC/", true, true},
		{"/aBc/", "/ABC/", true, false},
		{"/abC", "/ABC/", true, true},
		{"/abC/", "/ABC/", true, false},
		{"/SEARCH/QUERY", "/search/QUERY", true, false},
		{"/SEARCH/QUERY/", "/search/QUERY", true, true},
		{"/CMD/TOOL/", "/cmd/TOOL/", true, false},
		{"/CMD/TOOL", "/cmd/TOOL/", true, true},
		{"/SRC/FILE/PATH", "/src/FILE/PATH", true, false},
		{"/x/Y", "/x/y", true, false},
		{"/x/Y/", "/x/y", true, true},
		{"/X/y", "/x/y", true, false},
		{"/X/y/", "/x/y", true, true},
		{"/X/Y", "/x/y", true, false},
		{"/X/Y/", "/x/y", true, true},
		{"/Y/", "/y/", true, false},
		{"/Y", "/y/", true, true},
		{"/Y/z", "/y/z", true, false},
		{"/Y/z/", "/y/z", true, true},
		{"/Y/Z", "/y/z", true, false},
		{"/Y/Z/", "/y/z", true, true},
		{"/y/Z", "/y/z", true, false},
		{"/y/Z/", "/y/z", true, true},
		{"/Aa", "/aa", true, false},
		{"/Aa/", "/aa", true, true},
		{"/AA", "/aa", true, false},
		{"/AA/", "/aa", true, true},
		{"/aA", "/aa", true, false},
		{"/aA/", "/aa", true, true},
		{"/A/", "/a/", true, false},
		{"/A", "/a/", true, true},
		{"/DOC", "/doc", true, false},
		{"/DOC/", "/doc", true, true},
		{"/NO", "", false, true},
		{"/DOC/GO", "", false, true},
		{"/π", "/Π", true, false},
		{"/π/", "/Π", true, true},
		{"/u/ÄPFÊL/", "/u/äpfêl/", true, false},
		{"/u/ÄPFÊL", "/u/äpfêl/", true, true},
		{"/u/ÖPFÊL/", "/u/öpfêl", true, true},
		{"/u/ÖPFÊL", "/u/öpfêl", true, false},
		{"/v/äpfêL/", "/v/Äpfêl/", true, false},
		{"/v/äpfêL", "/v/Äpfêl/", true, true},
		{"/v/öpfêL/", "/v/Öpfêl", true, true},
		{"/v/öpfêL", "/v/Öpfêl", true, false},
		{"/w/♬/", "/w/♬", true, true},
		{"/w/♭", "/w/♭/", true, true},
		{"/w/𠜎/", "/w/𠜎", true, true},
		{"/w/𠜏", "/w/𠜏/", true, true},
	}
	// With fixTrailingSlash = true
	for _, test := range tests {
		out, found, err := tree.findCaseInsensitive(test.in, true)
		assert.FailOnError(t, err, "")

		assert.Equalf(t, test.found, found,
			"Wrong result for '%s': got %s, %t; want %s, %t",
			test.in, string(out), found, test.out, test.found)

		assert.Falsef(t, (found && (string(out) != test.out)),
			"Wrong result for '%s': got %s, %t; want %s, %t",
			test.in, string(out), found, test.out, test.found)
	}

	// With fixTrailingSlash = false
	for _, test := range tests {
		out, found, err := tree.findCaseInsensitive(test.in, false)
		assert.FailOnError(t, err, "")

		if test.slash {
			// test needs a trailingSlash fix. It must not be found!
			assert.Falsef(t, found, "Found without fixTrailingSlash: %s; got %s", test.in, string(out))
		} else {
			assert.Equalf(t, test.found, found,
				"Wrong result for '%s': got %s, %t; want %s, %t",
				test.in, string(out), found, test.out, test.found)

			assert.Falsef(t, (found && (string(out) != test.out)),
				"Wrong result for '%s': got %s, %t; want %s, %t",
				test.in, string(out), found, test.out, test.found)
		}
	}
}

func TestTreeInvalidNodeType(t *testing.T) {
	const panicMsg = "invalid node type"

	tree := &node{}
	_ = tree.add("/", "/")
	_ = tree.add("/:page", "/:page")

	// set invalid node type
	tree.edges[0].nType = 42

	// normal lookup
	_, _, _, err := tree.find("/test")
	assert.Equal(t, panicMsg, err.Error())

	_, _, err = tree.findCaseInsensitive("/test", true)
	assert.Equal(t, panicMsg, err.Error())
}

func checkRequests(t *testing.T, tree *node, requests testRequests) {
	for _, request := range requests {
		handler, pp, _, _ := tree.find(request.path)

		if handler == nil {
			assert.Truef(t, request.nilHandler, "value mismatch for route '%s': Expected non-nil value", request.path)
		} else if request.nilHandler {
			t.Errorf("value mismatch for route '%s': Expected nil value", request.path)
		} else {
			assert.Truef(t,
				reflect.DeepEqual(handler, request.route),
				"value mismatch for route '%s': Wrong value (%s != %s)",
				request.path, request.route, request.route)
		}

		assert.Truef(t,
			reflect.DeepEqual(pp, request.pp),
			"PathParams mismatch for route '%s'", request.path)
	}
}

func checkPriorities(t *testing.T, n *node) uint32 {
	var prio uint32
	for i := range n.edges {
		prio += checkPriorities(t, n.edges[i])
	}

	if n.value != nil {
		prio++
	}

	assert.Equalf(t, n.priority, prio,
		"priority mismatch for node '%s': is %d, should be %d",
		n.path, n.priority, prio)

	return prio
}

func checkMaxParams(t *testing.T, n *node) uint8 {
	var maxParams uint8
	for i := range n.edges {
		params := checkMaxParams(t, n.edges[i])
		if params > maxParams {
			maxParams = params
		}
	}
	if n.nType > root && !n.wildChild {
		maxParams++
	}

	assert.Equalf(t, n.maxParams, maxParams,
		"maxParams mismatch for node '%s': is %d, should be %d",
		n.path, n.maxParams, maxParams)

	return maxParams
}

func printChildren(n *node, prefix string) {
	fmt.Printf(" %02d:%02d %s%s[%d] %v %t %d \r\n",
		n.priority,
		n.maxParams,
		prefix,
		n.path,
		len(n.edges),
		n.value,
		n.wildChild,
		n.nType)

	for l := len(n.path); l > 0; l-- {
		prefix += " "
	}

	for _, child := range n.edges {
		printChildren(child, prefix)
	}
}
