// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/essentials source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ess

import (
	"io/ioutil"
	"os"
	"runtime"
	"testing"

	"aahframework.org/test/assert"
)

func TestIsFileExists(t *testing.T) {
	testdataPath := getTestdataPath()

	assert.Equal(t, true, IsFileExists(join(testdataPath, "sample.txt")))
	assert.Equal(t, false, IsFileExists(join(testdataPath, "sample-not-exists.txt")))

	assert.Equal(t, true, IsFileExists(testdataPath))
	assert.Equal(t, false, IsFileExists("testdata-not-exists"))
}

func TestIsDirEmpty(t *testing.T) {
	testdataPath := getTestdataPath()

	assert.Equal(t, false, IsDirEmpty(testdataPath))
	assert.Equal(t, true, IsDirEmpty("testdata-not-exists.txt"))

	dirPath := join(testdataPath, "path", "isdirempty")
	_ = MkDirAll(dirPath, 0755)
	assert.Equal(t, true, IsDirEmpty(dirPath))
}

func TestIsDir(t *testing.T) {
	testdataPath := getTestdataPath()

	result := IsDir(testdataPath)
	assert.True(t, result)

	result = IsDir(join(testdataPath, "sample.txt"))
	assert.False(t, result)

	result = IsDir(join(testdataPath, "not-exists.txt"))
	assert.False(t, result)
}

func TestApplyFileMode(t *testing.T) {
	fileName := join(getTestdataPath(), "FileMode.txt")
	defer DeleteFiles(fileName)

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

	if runtime.GOOS != "windows" {
		// expected to fail
		err = ApplyFileMode("/var", 0755)
		assert.NotNil(t, err)
	}
}

func TestLineCntByFilePath(t *testing.T) {
	count := LineCnt(join(getTestdataPath(), "sample.txt"))
	assert.Equal(t, 20, count)

	count = LineCnt(join(getTestdataPath(), "sample-not.txt"))
	assert.Equal(t, 0, count)
}

func TestLineCntByReader(t *testing.T) {
	file, err := os.Open(join(getTestdataPath(), "sample.txt"))
	assert.FailOnError(t, err, "unable to open file")
	defer CloseQuietly(file)

	assert.Equal(t, 20, LineCntr(file))
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
	assert.FailOnError(t, err, "unable to create file")

	err = MkDirAll(join(testdataPath, "symlinkdata"), 0755)
	assert.FailOnError(t, err, "")

	err = ioutil.WriteFile(join(testdataPath, "symlinkdata", "file1.txt"),
		[]byte(`This file is for file permission testing 2`), 0755)
	assert.FailOnError(t, err, "unable to create file")

	// preparing symlink for test
	err = os.Symlink(fileName, newName1)
	assert.FailOnError(t, err, "unable to create symlink")

	err = os.Symlink(fileName, newName2)
	assert.FailOnError(t, err, "unable to create symlink")

	err = os.Symlink(join(testdataPath, "symlinkdata"), newName3)
	assert.FailOnError(t, err, "unable to create symlink")

	err = CopyDir(join(tmpDir, "symlinktest"), testdataPath, Excludes{})
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
	testdataPath := getTestdataPath()

	_, err := CopyFile(
		join(testdataPath, "file-found.txt"),
		join(testdataPath, "file-not-exists.txt"),
	)
	assert.NotNil(t, err)

	_, err = CopyFile(
		join(testdataPath, "sample.txt"),
		join(testdataPath, "sample.txt"),
	)
	assert.NotNil(t, err)

	if runtime.GOOS != "windows" {
		_, err = CopyFile(
			"/var/you-will-not-be-able-to-create.txt",
			join(testdataPath, "sample.txt"),
		)
		assert.NotNil(t, err)
	}
}

func TestCopyDir(t *testing.T) {
	testdataPath := getTestdataPath()
	tmpDir := os.TempDir()

	defer DeleteFiles(join(tmpDir, "test1"))

	err1 := CopyDir(
		join(tmpDir, "target"),
		join(testdataPath, "not-exists-dir"),
		Excludes{},
	)
	assert.NotNil(t, err1)

	err2 := CopyDir(
		join(tmpDir, "target"),
		join(testdataPath, "sample.txt"),
		Excludes{},
	)
	assert.NotNil(t, err2)

	// err3 := CopyDir(tmpDir, testdataPath, Excludes{})
	// assert.True(t, strings.HasPrefix(err3.Error(), "destination dir already exists"))

	err4 := CopyDir(join(tmpDir, "target"), testdataPath, Excludes{"[]a]"})
	assert.NotNil(t, err4)

	pwd, _ := os.Getwd()
	err5 := CopyDir(join(tmpDir, "test1"), pwd, Excludes{"test*", "*conf", ".*"})
	assert.FailNowOnError(t, err5, "copy directory failed")
}

func TestStripExt(t *testing.T) {
	name1 := StripExt("/sample/path/to/file/working.txt")
	assert.Equal(t, "/sample/path/to/file/working", name1)

	name2 := StripExt("woriking-fine.pdf")
	assert.Equal(t, "woriking-fine", name2)

	name3 := StripExt("")
	assert.Equal(t, "", name3)
}

func TestDirPaths(t *testing.T) {
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

	dirs, err := DirsPath(join(testdataPath, "dirpaths"))
	assert.FailOnError(t, err, "unable to get directory list")
	assert.True(t, IsSliceContainsString(dirs, path1))
	assert.True(t, IsSliceContainsString(dirs, path11))
	assert.True(t, IsSliceContainsString(dirs, path12))
	assert.True(t, IsSliceContainsString(dirs, path21))
	assert.True(t, IsSliceContainsString(dirs, path22))
	assert.False(t, IsSliceContainsString(dirs, join(path22, "not-exists")))
}
