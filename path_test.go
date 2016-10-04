// Copyright (c) 2016 Jeevanandam M (https://github.com/jeevatkm)
// Copyright (c) 2013 Julien Schmidt (https://github.com/julienschmidt)
// All rights reserved.
// Based on the path package, Copyright 2009 The Go Authors.
// Use of this path.go source code is governed by a BSD-style license that can be found
// in the LICENSE file at
// https://raw.githubusercontent.com/julienschmidt/httprouter/master/LICENSE &
// https://raw.githubusercontent.com/golang/go/master/LICENSE

package router

import (
	"runtime"
	"testing"

	"aahframework.org/test/assert"
)

var cleanTests = []struct {
	path, result string
}{
	// Already clean
	{"/", "/"},
	{"/abc", "/abc"},
	{"/a/b/c", "/a/b/c"},
	{"/abc/", "/abc/"},
	{"/a/b/c/", "/a/b/c/"},

	// missing root
	{"", "/"},
	{"abc", "/abc"},
	{"abc/def", "/abc/def"},
	{"a/b/c", "/a/b/c"},

	// Remove doubled slash
	{"//", "/"},
	{"/abc//", "/abc/"},
	{"/abc/def//", "/abc/def/"},
	{"/a/b/c//", "/a/b/c/"},
	{"/abc//def//ghi", "/abc/def/ghi"},
	{"//abc", "/abc"},
	{"///abc", "/abc"},
	{"//abc//", "/abc/"},

	// Remove . elements
	{".", "/"},
	{"./", "/"},
	{"/abc/./def", "/abc/def"},
	{"/./abc/def", "/abc/def"},
	{"/abc/.", "/abc/"},

	// Remove .. elements
	{"..", "/"},
	{"../", "/"},
	{"../../", "/"},
	{"../..", "/"},
	{"../../abc", "/abc"},
	{"/abc/def/ghi/../jkl", "/abc/def/jkl"},
	{"/abc/def/../ghi/../jkl", "/abc/jkl"},
	{"/abc/def/..", "/abc"},
	{"/abc/def/../..", "/"},
	{"/abc/def/../../..", "/"},
	{"/abc/def/../../..", "/"},
	{"/abc/def/../../../ghi/jkl/../../../mno", "/mno"},

	// Combinations
	{"abc/./../def", "/def"},
	{"abc//./../def", "/def"},
	{"abc/../../././../def", "/def"},
}

func TestPathClean(t *testing.T) {
	for _, test := range cleanTests {
		result := CleanPath(test.path)
		assert.Equalf(t, test.result, result, "CleanPath(%q) = %q, want %q", test.path, result, test.result)

		result = CleanPath(test.result)
		assert.Equalf(t, test.result, result, "CleanPath(%q) = %q, want %q", test.result, result, test.result)
	}
}

func TestPathCleanMallocs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping malloc count in short mode")
	}
	if runtime.GOMAXPROCS(0) > 1 {
		t.Log("skipping AllocsPerRun checks; GOMAXPROCS>1")
		return
	}

	for _, test := range cleanTests {
		allocs := testing.AllocsPerRun(100, func() { CleanPath(test.result) })
		if allocs > 0 {
			t.Errorf("CleanPath(%q): %v allocs, want zero", test.result, allocs)
		}
	}
}
