// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org/vfs source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package vfs

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

var _ FileSystem = (*Mount)(nil)

// Mount struct represents mount of single physical directory into virtual directory.
//
// Mount implements `vfs.FileSystem`, its a combination of package `os` and `ioutil`
// focused on Read-Only operations.
type Mount struct {
	vroot string
	proot string
	tree  *node
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Mount's FileSystem interface
//______________________________________________________________________________

// Open method behaviour is same as `os.Open`.
func (m Mount) Open(name string) (File, error) {
	f, err := m.open(name)
	if os.IsNotExist(err) {
		return m.openPhysical(name)
	}
	return f, err
}

// Lstat method behaviour is same as `os.Lstat`.
func (m Mount) Lstat(name string) (os.FileInfo, error) {
	f, err := m.open(name)
	if os.IsNotExist(err) {
		return os.Lstat(m.toPhysicalPath(name))
	}
	return f, err
}

// Stat method behaviour is same as `os.Stat`
func (m Mount) Stat(name string) (os.FileInfo, error) {
	f, err := m.open(name)
	if os.IsNotExist(err) {
		return os.Stat(m.toPhysicalPath(name))
	}
	return f, err
}

// ReadFile method behaviour is same as `ioutil.ReadFile`.
func (m Mount) ReadFile(name string) ([]byte, error) {
	f, err := m.Open(name)
	if os.IsNotExist(err) {
		f, err = m.openPhysical(name)
	}

	if err != nil {
		return nil, err
	}

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}

	if fi.IsDir() {
		return nil, &os.PathError{Op: "read", Path: name, Err: errors.New("is a directory")}
	}

	return ioutil.ReadAll(f)
}

// ReadDir method behaviour is same as `ioutil.ReadDir`.
func (m Mount) ReadDir(dirname string) ([]os.FileInfo, error) {
	f, err := m.open(dirname)
	if os.IsNotExist(err) {
		return ioutil.ReadDir(m.toPhysicalPath(dirname))
	}

	if !f.IsDir() {
		return nil, &os.PathError{Op: "read", Path: dirname, Err: errors.New("is a file")}
	}

	list := append([]os.FileInfo{}, f.node.childInfos...)
	sort.Sort(byName(list))

	return list, nil
}

// Glob method somewhat similar to `filepath.Glob`, since aah vfs does pattern
// match only on `filepath.Base` value.
func (m Mount) Glob(pattern string) ([]string, error) {
	var matches []string
	f, err := m.open(pattern)
	if os.IsNotExist(err) {
		flist, err := filepath.Glob(m.toPhysicalPath(pattern))
		if err != nil {
			return nil, err
		}
		for _, p := range flist {
			matches = append(matches, m.toVirtualPath(p))
		}
		return matches, nil
	}

	base := path.Base(pattern)
	for _, c := range f.childs {
		match, err := filepath.Match(base, c.Name())
		if err != nil {
			return nil, err
		}
		if match {
			matches = append(matches, c.Path)
		}
	}

	return matches, nil
}

// IsExists method is helper to find existence.
func (m Mount) IsExists(name string) bool {
	_, err := m.Lstat(name)
	return err == nil
}

// String method Stringer interface.
func (m Mount) String() string {
	return fmt.Sprintf("mount(%s => %s)", m.vroot, m.proot)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Mount adding file and directory
//______________________________________________________________________________

// Name method returns mounted path.
func (m *Mount) Name() string {
	return m.vroot
}

// AddDir method is to add directory node into VFS from mounted source directory.
func (m *Mount) AddDir(fi os.FileInfo) error {
	return m.addNode(fi, nil)
}

// AddFile method is to add file node into VFS from mounted source directory.
func (m *Mount) AddFile(fi os.FileInfo, data []byte) error {
	return m.addNode(fi, data)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Mount unexported methods
//______________________________________________________________________________

func (m Mount) cleanDir(p string) string {
	dp := strings.TrimPrefix(p, m.vroot)
	return path.Dir(dp)
}

func (m Mount) open(name string) (*file, error) {
	if m.tree == nil {
		return nil, os.ErrInvalid
	}

	name = path.Clean(name)
	if m.vroot == name { // extact match, root dir
		return newFile(m.tree), nil
	}

	return m.tree.find(strings.TrimPrefix(name, m.vroot))
}

func (m Mount) openPhysical(name string) (File, error) {
	pname := m.toPhysicalPath(name)
	if _, err := os.Lstat(pname); os.IsNotExist(err) {
		return nil, err
	}
	return os.Open(pname)
}

func (m Mount) toPhysicalPath(name string) string {
	return filepath.Clean(filepath.FromSlash(filepath.Join(m.proot, name[len(m.vroot):])))
}

func (m *Mount) toVirtualPath(name string) string {
	return filepath.Clean(filepath.ToSlash(filepath.Join(m.vroot, name[len(m.proot):])))
}

func (m *Mount) addNode(fi os.FileInfo, data []byte) error {
	mountPath := fi.(*NodeInfo).Path
	t, err := m.tree.findNode(m.cleanDir(mountPath))
	switch {
	case err != nil:
		return err
	case t == nil:
		return nil
	}

	n := newNode(mountPath, fi)
	if data != nil {
		n.data = data
	}
	t.addChild(n)

	return nil

}
