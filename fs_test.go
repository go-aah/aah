// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/ahttp source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ahttp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aahframework.org/test.v0-unstable/assert"
)

func TestNoDirDisplay(t *testing.T) {
	path := testdataPath()

	fs := Dir(path, false)
	assert.NotNil(t, fs)

	f1, err1 := fs.Open("file1.txt")
	assert.NotNil(t, f1)
	assert.Nil(t, err1)

	f2, err2 := fs.Open("dir1")
	assert.Nil(t, f2)
	assert.Equal(t, "directory listing not allowed", err2.Error())

	f3, err3 := fs.Open("file11.txt")
	assert.Nil(t, f3)
	assert.True(t, strings.Contains(err3.Error(), "no such file or directory"))
}

func TestDirDisplay(t *testing.T) {
	path := testdataPath()

	fs := Dir(path, true)
	assert.NotNil(t, fs)

	f1, err1 := fs.Open("dir1")
	st, _ := f1.Stat()
	assert.Equal(t, "dir1", st.Name())
	assert.Nil(t, err1)

	f2, err2 := fs.Open("file1\x00.txt")
	assert.Nil(t, f2)
	assert.Equal(t, "http: invalid character in file path", err2.Error())
}

func testdataPath() string {
	wd, _ := os.Getwd()
	return filepath.Join(wd, "testdata")
}
