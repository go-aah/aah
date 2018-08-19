// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org/essentials source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ess

import (
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArchiveZip(t *testing.T) {
	// Prepare data for zip file
	testdataPath := getTestdataPath()
	path1 := join(testdataPath, "dirpaths", "level1", "level2", "level3")
	path11 := join(testdataPath, "dirpaths", "level1", "level1-1")
	path12 := join(testdataPath, "dirpaths", "level1", "level1-2")
	path21 := join(testdataPath, "dirpaths", "level1", "level2", "level2-1")
	path22 := join(testdataPath, "dirpaths", "level1", "level2", "level2-2")
	defer DeleteFiles(join(testdataPath, "dirpaths"))

	_ = MkDirAll(path1, 0755)
	_ = MkDirAll(path11, 0755)
	_ = MkDirAll(path12, 0755)
	_ = MkDirAll(path21, 0755)
	_ = MkDirAll(path22, 0755)

	_ = ioutil.WriteFile(join(path1, "file1.txt"), []byte("file1.txt"), 0600)
	_ = ioutil.WriteFile(join(path11, "file11.txt"), []byte("file11.txt"), 0600)
	_ = ioutil.WriteFile(join(path12, "file12.txt"), []byte("file12.txt"), 0600)
	_ = ioutil.WriteFile(join(path21, "file21.txt"), []byte("file21.txt"), 0600)
	_ = ioutil.WriteFile(join(path22, "file22.txt"), []byte("file22.txt"), 0600)

	zipName := join(testdataPath, "testarchive.zip")
	defer DeleteFiles(zipName)

	err := Zip(zipName, join(testdataPath, "dirpaths"))
	assert.Nil(t, err)
	assert.True(t, IsFileExists(zipName))

	err = Zip(zipName, join(testdataPath, "dirpaths1"))
	assert.NotNil(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "source does not exists:"))

	err = Zip(zipName, join(testdataPath, "dirpaths"))
	assert.NotNil(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "destination archive already exists:"))
}
