// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ahttp

import (
	"net/http"
	"os"
)

// FileOnlyFilesystem extends/wraps `http.FileSystem` to disable directory listing
// fucntionality
type FileOnlyFilesystem struct {
	Fs http.FileSystem
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Global methods
//___________________________________

// Dir method returns a `http.Filesystem` that can be directly used by http.FileServer().
// It works the same as `http.Dir()` also provides ability to disable directory listing
// with `http.FileServer`
func Dir(path string, listDir bool) http.FileSystem {
	fs := http.Dir(path)
	if listDir {
		return fs
	}
	return FileOnlyFilesystem{Fs: fs}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// FileOnlyFilesystem methods
//___________________________________

// Open method is compilance with `http.FileSystem` interface and disables
// directory listing
func (fs FileOnlyFilesystem) Open(name string) (http.File, error) {
	stat, err := os.Lstat(name)
	if err != nil {
		return nil, err
	}

	if stat.IsDir() {
		return nil, os.ErrNotExist
	}

	file, err := fs.Fs.Open(name)
	if err != nil {
		return nil, err
	}

	return file, nil
}
