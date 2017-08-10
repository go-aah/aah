// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/view source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package view

import (
	"path/filepath"
	"strings"
)

var fSeparator = filepath.Separator

func parseKey(baseDir, fpath string) string {
	fpath = trimPathPrefix(fpath, baseDir)
	if fSeparator == '/' {
		fpath = strings.Replace(fpath, "/", "_", -1)
	} else {
		fpath = strings.Replace(fpath, "\\", "_", -1)
	}
	return fpath
}

func trimPathPrefix(fpath, prefix string) string {
	fpath = strings.TrimPrefix(fpath, prefix)
	if fpath[0] == '/' || fpath[0] == '\\' {
		fpath = fpath[1:]
	}
	return fpath
}
