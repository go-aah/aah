// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org/vfs source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package vfs

import (
	"io"
	"os"
)

// FileSystem interface implements access to a collection of named files.
// The elements in a file path are separated by slash ('/', U+002F) characters,
// regardless of host operating system convention.
//
// aah vfs is Read-Only.
type FileSystem interface {
	Open(name string) (File, error)
	Lstat(name string) (os.FileInfo, error)
	Stat(name string) (os.FileInfo, error)
	ReadFile(filename string) ([]byte, error)
	ReadDir(dirname string) ([]os.FileInfo, error)
}

// File interface returned by a vfs.FileSystem's Open method.
//
// aah vfs is Read-Only.
type File interface {
	ReadSeekCloser
	Readdir(n int) ([]os.FileInfo, error)
	Readdirnames(n int) (names []string, err error)
	Stat() (os.FileInfo, error)
}

// ReadSeekCloser interface can Read, Seek, and Close.
type ReadSeekCloser interface {
	io.Reader
	io.Seeker
	io.Closer
}

// Gziper interface is to identify node data is gzipped or not.
type Gziper interface {
	IsGzip() bool
	RawBytes() []byte
}
