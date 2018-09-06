// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

// +build !windows

package ess

import (
	"io/ioutil"
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApplyFileMode(t *testing.T) {
	fileName := join(getTestdataPath(), "FileMode.txt")
	defer DeleteFiles(fileName)

	err := ioutil.WriteFile(fileName,
		[]byte(`This file is for file permission testing`), 0700)
	assert.Nil(t, err, "file permission issue")

	fileInfo, err := os.Stat(fileName)
	assert.Nil(t, err, "couldn't to file stat")
	if fileInfo.Mode() != os.FileMode(0700) {
		t.Errorf("expected file mode: 0700 got %v", fileInfo.Mode())
	}

	err = ApplyFileMode(fileName, 0755)
	assert.Nil(t, err, "couldn't apply file permission")

	fileInfo, err = os.Stat(fileName)
	assert.Nil(t, err, "couldn't to file stat")
	if fileInfo.Mode() != os.FileMode(0755) {
		t.Errorf("expected file mode: 0755 got %v", fileInfo.Mode())
	}

	if runtime.GOOS != "windows" {
		// expected to fail
		err = ApplyFileMode("/var", 0755)
		assert.NotNil(t, err)
	}
}

func TestWalk(t *testing.T) {
	testdataPath := getTestdataPath()
	fileName := join(testdataPath, "symlinktest.txt")
	newName1 := join(testdataPath, "symlinktest1.txt")
	newName2 := join(testdataPath, "symlinktest2.txt")
	newName3 := join(testdataPath, "symlinkdata1")

	tmpDir := os.TempDir()

	defer func() {
		DeleteFiles(fileName, newName1, newName2, newName3)
		DeleteFiles(
			join(testdataPath, "symlinkdata"),
			join(tmpDir, "symlinktest"),
			join(testdataPath, "symlinkdata1"),
		)
	}()

	err := ioutil.WriteFile(fileName,
		[]byte(`This file is for file permission testing 1`), 0755)
	assert.Nil(t, err, "unable to create file")

	err = MkDirAll(join(testdataPath, "symlinkdata"), 0755)
	assert.Nil(t, err, "")

	err = ioutil.WriteFile(join(testdataPath, "symlinkdata", "file1.txt"),
		[]byte(`This file is for file permission testing 2`), 0755)
	assert.Nil(t, err, "unable to create file")

	// preparing symlink for test
	err = os.Symlink(fileName, newName1)
	assert.Nil(t, err, "unable to create symlink")

	err = os.Symlink(fileName, newName2)
	assert.Nil(t, err, "unable to create symlink")

	err = os.Symlink(join(testdataPath, "symlinkdata"), newName3)
	assert.Nil(t, err, "unable to create symlink")

	err = CopyDir(join(tmpDir, "symlinktest"), testdataPath, Excludes{})
	assert.Nil(t, err, "")
}
