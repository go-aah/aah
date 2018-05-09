// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org/vfs source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package vfs

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
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

// readDirNames reads the directory named by dirname and returns
// a sorted list of directory entries.
func readDirNames(fs FileSystem, dirname string) ([]string, error) {
	f, err := fs.Open(dirname)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	names, err := f.Readdirnames(-1)
	if err != nil {
		return nil, err
	}

	sort.Strings(names)
	return names, nil
}

// walk recursively descends path.
func walk(fs FileSystem, fpath string, info os.FileInfo, walkFn filepath.WalkFunc) error {
	err := walkFn(fpath, info, nil)
	if err != nil {
		if info.IsDir() && err == filepath.SkipDir {
			return nil
		}
		return err
	}

	if !info.IsDir() {
		return nil
	}

	names, err := readDirNames(fs, fpath)
	if err != nil {
		return walkFn(fpath, info, err)
	}

	for _, name := range names {
		filename := path.Join(fpath, name)
		fi, err := fs.Lstat(filename)
		if err != nil {
			if err := walkFn(filename, fi, err); err != nil && err != filepath.SkipDir {
				return err
			}
		} else {
			err = walk(fs, filename, fi, walkFn)
			if err != nil {
				if !fi.IsDir() || err != filepath.SkipDir {
					return err
				}
			}
		}
	}
	return nil
}

// sanitize prepares a valid UTF-8 string as a raw string constant.
// https://github.com/golang/tools/blob/master/godoc/static/gen.go
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
