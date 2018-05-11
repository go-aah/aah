// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org/vfs source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package vfs

import (
	"bytes"
	"compress/gzip"
	"os"
	"path"
	"path/filepath"
	"sort"
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package unexported methods
//______________________________________________________________________________

func newNode(name string, fi os.FileInfo) *node {
	return &node{
		NodeInfo:   newNodeInfo(name, fi),
		childInfos: make([]os.FileInfo, 0),
		childs:     make(map[string]*node),
	}
}

func newNodeInfo(name string, fi os.FileInfo) *NodeInfo {
	return &NodeInfo{
		Path:     name,
		Dir:      fi.IsDir(),
		DataSize: fi.Size(),
		Time:     fi.ModTime(),
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
	defer func() { _ = f.Close() }()

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
			if err = walkFn(filename, fi, err); err != nil && err != filepath.SkipDir {
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

// byName implements sort.Interface
type byName []os.FileInfo

func (f byName) Len() int           { return len(f) }
func (f byName) Less(i, j int) bool { return f[i].Name() < f[j].Name() }
func (f byName) Swap(i, j int)      { f[i], f[j] = f[j], f[i] }
