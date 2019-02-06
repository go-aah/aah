// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package view

import (
	"strings"
	"sync"
)

// StripPathPrefixAt method strips the given path to path cut position.
//
// For Example:
//
// 	path := "/Users/jeeva/go/src/github.com/go-aah/tutorials/form/views/common/header.html"
// 	result := StripPrefixAt(path, "views/")
func StripPathPrefixAt(str, pathCut string) string {
	if idx := strings.Index(str, pathCut); idx > 0 {
		return str[idx+len(pathCut):]
	}
	return str
}

// TrimPathPrefix method trims given file paths by prefix and
// returns comma separated string.
func TrimPathPrefix(prefix string, fpaths ...string) string {
	var fs []string
	for _, fp := range fpaths {
		fs = append(fs, trimPathPrefix(prefix, fp))
	}
	return strings.Join(fs, ", ")
}

func trimPathPrefix(prefix, fpath string) string {
	fpath = strings.TrimPrefix(fpath, prefix)
	if fpath[0] == '/' || fpath[0] == '\\' {
		fpath = fpath[1:]
	}
	return fpath
}

var builderPool = &sync.Pool{New: func() interface{} { return new(strings.Builder) }}

func acquireBuilder() *strings.Builder {
	return builderPool.Get().(*strings.Builder)
}

func releaseBuilder(b *strings.Builder) {
	b.Reset()
	builderPool.Put(b)
}
