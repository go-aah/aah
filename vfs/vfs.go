// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org/vfs source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

// Package vfs provides Virtual FileSystem (VFS) capability. Typically it reflects
// OS FileSystem behavior in-memory.
//
// aah vfs is Read-Only, even though vfs design nature could support Write
// operations. I have limited it.
//
// The methods should behave the same as those on an *os.File for Read-Only.
package vfs

import (
	"net/http"
	"os"
)

// FileSystem interface implements access to a collection of named files.
// The elements in a file path are separated by slash ('/', U+002F) characters,
// regardless of host operating system convention.
type FileSystem interface {
	Open(name string) (File, error)
	Lstat(name string) (os.FileInfo, error)
	Stat(name string) (os.FileInfo, error)
	ReadFile(filename string) ([]byte, error)
	ReadDir(dirname string) ([]os.FileInfo, error)
	Glob(pattern string) ([]string, error)
	IsExists(name string) bool
}

// File interface returned by a vfs.FileSystem's Open method.
type File interface {
	http.File
	Readdirnames(n int) ([]string, error)
}

// RawBytes interface is to retrieve underlying file's raw bytes.
//
// Note: It could be gzip or non-gzip bytes. Use interface `Gziper`
// to identify byte classification.
type RawBytes interface {
	RawBytes() []byte
}

// Gziper interface is to identify whether the file's raw bytes is gzipped or not.
type Gziper interface {
	RawBytes
	IsGzip() bool
}
