// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org/vfs source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package vfs

import (
	"errors"
	"fmt"
	"io"
	"os"
)

var _ File = (*file)(nil)
var _ Gziper = (*file)(nil)

// File struct represents the virtual file or directory.
//
// Implements interface `vfs.File`.
type file struct {
	*node
	rs  io.ReadSeeker
	pos int
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// File and Directory operations
//______________________________________________________________________________

func (f *file) Read(b []byte) (int, error) {
	return f.rs.Read(b)
}

func (f *file) Seek(offset int64, whence int) (int64, error) {
	return f.rs.Seek(offset, whence)
}

func (f *file) Readdir(count int) ([]os.FileInfo, error) {
	if !f.IsDir() {
		return []os.FileInfo{}, &os.PathError{Op: "read", Path: f.node.name, Err: errors.New("vfs: cannot find the specified path")}
	}

	if f.pos >= len(f.node.childInfos) && count > 0 {
		return nil, io.EOF
	}

	if count <= 0 || count > len(f.node.childInfos)-f.pos {
		count = len(f.node.childInfos) - f.pos
	}

	ci := f.node.childInfos[f.pos : f.pos+count]
	f.pos += count

	return ci, nil
}

func (f *file) Readdirnames(count int) (names []string, err error) {
	var list []string
	infos, err := f.Readdir(count)
	if err != nil {
		return list, err
	}

	for _, v := range infos {
		list = append(list, v.Name())
	}

	return list, nil
}

func (f *file) Stat() (os.FileInfo, error) {
	return f, nil
}

func (f *file) Close() error {
	if f.IsGzip() {
		return f.rs.(io.Closer).Close()
	}
	return nil
}

// String method Stringer interface.
func (f file) String() string {
	return fmt.Sprintf(`file(name=%s dir=%v gzip=%v)`,
		f.node.name, f.IsDir(), f.IsGzip())
}
