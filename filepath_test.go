// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/essentails source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ess

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-aah/test/assert"
)

func TestIsFileExists(t *testing.T) {
	assert.Equal(t, true, IsFileExists("testdata/sample.txt"))
	assert.Equal(t, false, IsFileExists("testdata/sample-not-exists.txt"))

	assert.Equal(t, true, IsFileExists("testdata"))
	assert.Equal(t, false, IsFileExists("testdata-not-exists"))
}

func TestIsDirEmpty(t *testing.T) {
	assert.Equal(t, false, IsDirEmpty("testdata"))
	assert.Equal(t, true, IsDirEmpty("testdata-not-exists.txt"))

	_ = MkDirAll("testdata/path/isdirempty", 0755)
	assert.Equal(t, true, IsDirEmpty("testdata/path/isdirempty"))
}

func TestApplyFileMode(t *testing.T) {
	fileName := "testdata/FileMode.txt"
	defer removeFiles(fileName)

	err := ioutil.WriteFile(fileName,
		[]byte(`This file is for file permission testing`), 0700)
	assert.FailOnError(t, err, "file permission issue")

	fileInfo, err := os.Stat(fileName)
	assert.FailOnError(t, err, "couldn't to file stat")
	if fileInfo.Mode() != os.FileMode(0700) {
		t.Errorf("expected file mode: 0700 got %v", fileInfo.Mode())
	}

	err = ApplyFileMode(fileName, 0755)
	assert.FailOnError(t, err, "couldn't apply file permission")

	fileInfo, err = os.Stat(fileName)
	assert.FailOnError(t, err, "couldn't to file stat")
	if fileInfo.Mode() != os.FileMode(0755) {
		t.Errorf("expected file mode: 0755 got %v", fileInfo.Mode())
	}

	// expected to fail
	err = ApplyFileMode("/var", 0755)
	assert.NotNil(t, err)
}

func TestLineCntByFilePath(t *testing.T) {
	count := LineCnt("testdata/sample.txt")
	assert.Equal(t, 20, count)

	count = LineCnt("testdata/sample-not.txt")
	assert.Equal(t, 0, count)
}

func TestLineCntByReader(t *testing.T) {
	file, err := os.Open("testdata/sample.txt")
	assert.FailOnError(t, err, "unable to open file")
	defer CloseQuietly(file)

	assert.Equal(t, 20, LineCntr(file))
}

func TestWalk(t *testing.T) {
	pwd, _ := os.Getwd()
	fileName := filepath.Join(pwd, "testdata/symlinktest.txt")
	newName1 := filepath.Join(pwd, "testdata/symlinktest1.txt")
	newName2 := filepath.Join(pwd, "testdata/symlinktest2.txt")
	newName3 := filepath.Join(pwd, "testdata/symlinkdata1")

	defer func() {
		removeFiles(fileName, newName1, newName2, newName3)
		removeAllFiles("testdata/symlinkdata",
			"/tmp/symlinktest",
			"testdata/symlinkdata1")
	}()

	err := ioutil.WriteFile(fileName,
		[]byte(`This file is for file permission testing 1`), 0755)
	assert.FailOnError(t, err, "unable to create file")

	err = MkDirAll("testdata/symlinkdata", 0755)
	assert.FailOnError(t, err, "")

	err = ioutil.WriteFile("testdata/symlinkdata/file1.txt",
		[]byte(`This file is for file permission testing 2`), 0755)
	assert.FailOnError(t, err, "unable to create file")

	// preparing symlink for test
	err = os.Symlink(fileName, newName1)
	assert.FailOnError(t, err, "unable to create symlink")

	err = os.Symlink(fileName, newName2)
	assert.FailOnError(t, err, "unable to create symlink")

	err = os.Symlink(filepath.Join(pwd, "testdata/symlinkdata"), newName3)
	assert.FailOnError(t, err, "unable to create symlink")

	err = CopyDir("/tmp/symlinktest", "testdata", Excludes{})
	assert.FailOnError(t, err, "")
}

func TestExcludes(t *testing.T) {
	errExcludes := Excludes{
		".*",
		"DS_Store.bak",
		"[^",
		"[]a]",
	}
	assert.Equal(t, true, errExcludes.Validate() != nil)

	excludes := Excludes{
		".*",
		"*.bak",
		"*.tmp",
		"tmp",
	}
	assert.Equal(t, nil, excludes.Validate())
}

func TestCopyFile(t *testing.T) {
	_, err := CopyFile("testdata/file-found.txt", "testdata/file-not-exists.txt")
	assert.NotNil(t, err)

	_, err = CopyFile("testdata/sample.txt", "testdata/sample.txt")
	assert.NotNil(t, err)

	_, err = CopyFile("/var/you-will-not-be-able-to-create.txt", "testdata/sample.txt")
	assert.NotNil(t, err)
}

func TestCopyDir(t *testing.T) {
	err := CopyDir("/tmp/target", "testdata/not-exists-dir", Excludes{})
	assert.NotNil(t, err)

	err = CopyDir("/tmp/target", "testdata/sample.txt", Excludes{})
	assert.NotNil(t, err)

	err = CopyDir("/tmp", "testdata", Excludes{})
	assert.NotNil(t, err)

	err = CopyDir("/tmp/target", "testdata", Excludes{"[]a]"})
	assert.NotNil(t, err)

	pwd, _ := os.Getwd()
	err = CopyDir("/tmp/test1", pwd, Excludes{"test*", "*conf", ".*"})
	assert.FailNowOnError(t, err, "copy directory failed")

	removeAllFiles("/tmp/test1")
}
