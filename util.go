// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org/vfs source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package vfs

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"unicode/utf8"
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package methods
//______________________________________________________________________________

// Bytes2QuotedStr method converts byte slice into string take care of
// valid UTF-8 string preparation.
func Bytes2QuotedStr(b []byte) string {
	if len(b) == 0 {
		return ""
	}

	if utf8.Valid(b) {
		b = sanitize(b)
	}

	return fmt.Sprintf("%+q", b)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package unexported methods
//______________________________________________________________________________

func newNode(name string, fi os.FileInfo) *node {
	return &node{
		dir:        fi.IsDir(),
		name:       name,
		modTime:    fi.ModTime(),
		childInfos: make([]os.FileInfo, 0),
		childs:     make(map[string]*node),
	}
}

func newFile(n *node) *file {
	f := &file{node: n}

	if !f.IsDir() {
		// transparent reading for caller regardless of data bytes.
		f.rs = bytes.NewReader(f.node.data)
		if f.IsGzip() {
			r, _ := gzip.NewReader(f.rs)
			f.rs = &gzipData{n: n, r: r}
		}
	}

	return f
}

// https://github.com/golang/tools/blob/master/godoc/static/gen.go
// sanitize prepares a valid UTF-8 string as a raw string constant.
func sanitize(b []byte) []byte {
	// Replace ` with `+"`"+`
	b = bytes.Replace(b, []byte("`"), []byte("`+\"`\"+`"), -1)

	// Replace BOM with `+"\xEF\xBB\xBF"+`
	// (A BOM is valid UTF-8 but not permitted in Go source files.
	// I wouldn't bother handling this, but for some insane reason
	// jquery.js has a BOM somewhere in the middle.)
	return bytes.Replace(b, []byte("\xEF\xBB\xBF"), []byte("`+\"\\xEF\\xBB\\xBF\"+`"), -1)
}

// byName implements sort.Interface
type byName []os.FileInfo

func (f byName) Len() int           { return len(f) }
func (f byName) Less(i, j int) bool { return f[i].Name() < f[j].Name() }
func (f byName) Swap(i, j int)      { f[i], f[j] = f[j], f[i] }
