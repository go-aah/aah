// Copyright (c) 2016 Jeevanandam M (https://github.com/jeevatkm)
// Copyright (c) 2013 Julien Schmidt (https://github.com/julienschmidt)
// All rights reserved.
// Use of this radix_tree.go source code is governed by a BSD-style license that can be found
// in the LICENSE file at https://raw.githubusercontent.com/julienschmidt/httprouter/master/LICENSE.
// Previous author last modified on Feb 6, 2016, then I have modified it.

package router

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/go-aah/test/assert"
)

type testRequests []struct {
	path       string
	nilHandler bool
	route      string
	ps         Params
}

type testRoute struct {
	path     string
	conflict bool
}

func TestCountParams(t *testing.T) {
	if countParams("/path/:param1/static/*catch-all") != 2 {
		t.Fail()
	}
	if countParams(strings.Repeat("/:param", 256)) != 255 {
		t.Fail()
	}
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
		tree.addRoute(route, route)
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
		_ = tree.addRoute(route, route)
	}

	//printChildren(tree, "")

	checkRequests(t, tree, testRequests{
		{"/", false, "/", nil},
		{"/cmd/test/", false, "/cmd/:tool/", Params{Param{"tool", "test"}}},
		{"/cmd/test", true, "", Params{Param{"tool", "test"}}},
		{"/cmd/test/3", false, "/cmd/:tool/:sub", Params{Param{"tool", "test"}, Param{"sub", "3"}}},
		{"/src/", false, "/src/*filepath", Params{Param{"filepath", "/"}}},
		{"/src/some/file.png", false, "/src/*filepath", Params{Param{"filepath", "/some/file.png"}}},
		{"/search/", false, "/search/", nil},
		{"/search/someth!ng+in+ünìcodé", false, "/search/:query", Params{Param{"query", "someth!ng+in+ünìcodé"}}},
		{"/search/someth!ng+in+ünìcodé/", true, "", Params{Param{"query", "someth!ng+in+ünìcodé"}}},
		{"/user_gopher", false, "/user_:name", Params{Param{"name", "gopher"}}},
		{"/user_gopher/about", false, "/user_:name/about", Params{Param{"name", "gopher"}}},
		{"/files/js/inc/framework.js", false, "/files/:dir/*filepath", Params{Param{"dir", "js"}, Param{"filepath", "/inc/framework.js"}}},
		{"/info/gordon/public", false, "/info/:user/public", Params{Param{"user", "gordon"}}},
		{"/info/gordon/project/go", false, "/info/:user/project/:project", Params{Param{"user", "gordon"}, Param{"project", "go"}}},
	})

	checkPriorities(t, tree)
	checkMaxParams(t, tree)
}

func testRoutes(t *testing.T, routes []testRoute) {
	tree := &node{}

	for _, route := range routes {
		err := tree.addRoute(route.path, nil)
		if route.conflict {
			if err == nil {
				t.Errorf("no error for conflicting route '%s'", route.path)
			}
		} else if err != nil {
			t.Errorf("unexpected error for route '%s': %v", route.path, err)
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
		err := tree.addRoute(route, route)
		if err != nil {
			t.Fatalf("error inserting route '%s': %v", route, err)
		}

		// Add again
		err = tree.addRoute(route, nil)
		if err == nil {
			t.Fatalf("no error while inserting duplicate route '%s", route)
		}
	}

	// printChildren(tree, "")

	checkRequests(t, tree, testRequests{
		{"/", false, "/", nil},
		{"/doc/", false, "/doc/", nil},
		{"/src/some/file.png", false, "/src/*filepath", Params{Param{"filepath", "/some/file.png"}}},
		{"/search/someth!ng+in+ünìcodé", false, "/search/:query", Params{Param{"query", "someth!ng+in+ünìcodé"}}},
		{"/user_gopher", false, "/user_:name", Params{Param{"name", "gopher"}}},
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
		err := tree.addRoute(route, nil)
		if err == nil {
			t.Fatalf("no error while inserting route with empty wildcard name '%s", route)
		}
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
		err := tree.addRoute(route, nil)
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
		err := tree.addRoute(route, route)
		if err != nil {
			t.Fatalf("error inserting route '%s': %v", route, err)
		}
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

	err := tree.addRoute("/:test", "/:test")
	if err != nil {
		t.Fatalf("error inserting test route: %v", err)
	}

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
		err := tree.addRoute(route, route)
		if err != nil {
			t.Fatalf("error inserting route '%s': %v", route, err)
		}
	}

	// Check out == in for all registered routes
	// With fixTrailingSlash = true
	for _, route := range routes {
		out, found, err := tree.findCaseInsensitive(route, true)
		if !found && err == nil {
			t.Errorf("Route '%s' not found!", route)
		} else if string(out) != route {
			t.Errorf("Wrong result for route '%s': %s", route, string(out))
		}
	}

	// With fixTrailingSlash = false
	for _, route := range routes {
		out, found, err := tree.findCaseInsensitive(route, false)
		if !found && err == nil {
			t.Errorf("Route '%s' not found!", route)
		} else if string(out) != route {
			t.Errorf("Wrong result for route '%s': %s", route, string(out))
		}
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
		if found != test.found || (found && (string(out) != test.out)) || err != nil {
			t.Errorf("Wrong result for '%s': got %s, %t; want %s, %t",
				test.in, string(out), found, test.out, test.found)
			return
		}
	}

	// With fixTrailingSlash = false
	for _, test := range tests {
		out, found, err := tree.findCaseInsensitive(test.in, false)
		if test.slash || err != nil {
			if found { // test needs a trailingSlash fix. It must not be found!
				t.Errorf("Found without fixTrailingSlash: %s; got %s", test.in, string(out))
			}
		} else {
			if found != test.found || (found && (string(out) != test.out)) || err != nil {
				t.Errorf("Wrong result for '%s': got %s, %t; want %s, %t",
					test.in, string(out), found, test.out, test.found)
				return
			}
		}
	}
}

func TestTreeInvalidNodeType(t *testing.T) {
	const panicMsg = "invalid node type"

	tree := &node{}
	_ = tree.addRoute("/", "/")
	_ = tree.addRoute("/:page", "/:page")

	// set invalid node type
	tree.edges[0].nType = 42

	// normal lookup
	_, _, _, err := tree.find("/test")
	assert.Equal(t, "invalid node type", err.Error())

	_, _, err = tree.findCaseInsensitive("/test", true)
	assert.Equal(t, "invalid node type", err.Error())
}

func checkRequests(t *testing.T, tree *node, requests testRequests) {
	for _, request := range requests {
		handler, ps, _, _ := tree.find(request.path)

		if handler == nil {
			if !request.nilHandler {
				t.Errorf("value mismatch for route '%s': Expected non-nil value", request.path)
			}
		} else if request.nilHandler {
			t.Errorf("value mismatch for route '%s': Expected nil value", request.path)
		} else {
			if !reflect.DeepEqual(handler, request.route) {
				t.Errorf("value mismatch for route '%s': Wrong value (%s != %s)", request.path, request.route, request.route)
			}
		}

		if !reflect.DeepEqual(ps, request.ps) {
			t.Errorf("Params mismatch for route '%s'", request.path)
		}
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

	if n.priority != prio {
		t.Errorf(
			"priority mismatch for node '%s': is %d, should be %d",
			n.path, n.priority, prio,
		)
	}

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

	if n.maxParams != maxParams {
		t.Errorf(
			"maxParams mismatch for node '%s': is %d, should be %d",
			n.path, n.maxParams, maxParams,
		)
	}

	return maxParams
}

func printChildren(n *node, prefix string) {
	fmt.Printf(" %02d:%02d %s%s[%d] %v %t %d \r\n", n.priority, n.maxParams, prefix, n.path, len(n.edges), n.value, n.wildChild, n.nType)
	for l := len(n.path); l > 0; l-- {
		prefix += " "
	}
	for _, child := range n.edges {
		printChildren(child, prefix)
	}
}
