// Copyright (c) 2016 Jeevanandam M (https://github.com/jeevatkm)
// go-aah/essentails source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ess

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestIsFileExists(t *testing.T) {
	assertEqual(t, "TestIsFileExists", true, IsFileExists("testdata/sample.txt"))
	assertEqual(t, "TestIsFileExists", false, IsFileExists("testdata/sample-not-exists.txt"))

	assertEqual(t, "TestIsFileExists", true, IsFileExists("testdata"))
	assertEqual(t, "TestIsFileExists", false, IsFileExists("testdata-not-exists"))
}

func TestIsDirEmpty(t *testing.T) {
	assertEqual(t, "TestIsDirEmpty", false, IsDirEmpty("testdata"))
	assertEqual(t, "TestIsDirEmpty", true, IsDirEmpty("testdata-not-exists.txt"))

	_ = MkDirAll("testdata/path/isdirempty", 0755)
	assertEqual(t, "TestIsDirEmpty", true, IsDirEmpty("testdata/path/isdirempty"))
}

func TestApplyFileMode(t *testing.T) {
	fileName := "testdata/FileMode.txt"
	defer removeFiles(fileName)

	err := ioutil.WriteFile(fileName,
		[]byte(`This file is for file permission testing`), 0700)
	failOnError(t, err)

	fileInfo, err := os.Stat(fileName)
	failOnError(t, err)
	if fileInfo.Mode() != os.FileMode(0700) {
		t.Errorf("expected file mode: 0700 got %v", fileInfo.Mode())
	}

	err = ApplyFileMode(fileName, 0755)
	failOnError(t, err)

	fileInfo, err = os.Stat(fileName)
	failOnError(t, err)
	if fileInfo.Mode() != os.FileMode(0755) {
		t.Errorf("expected file mode: 0755 got %v", fileInfo.Mode())
	}

	// expected to fail
	err = ApplyFileMode("/var", 0755)
	if err == nil {
		t.Error("Expected error got nil")
	}
}

func TestLineCntByFilePath(t *testing.T) {
	count := LineCnt("testdata/sample.txt")
	assertEqual(t, "TestLineCntByFilePath", 20, count)

	count = LineCnt("testdata/sample-not.txt")
	assertEqual(t, "TestLineCntByFilePath", 0, count)
}

func TestLineCntByReader(t *testing.T) {
	file, err := os.Open("testdata/sample.txt")
	failOnError(t, err)
	defer CloseQuietly(file)

	assertEqual(t, "TestLineCntByReader", 20, LineCntr(file))
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
	failOnError(t, err)

	err = MkDirAll("testdata/symlinkdata", 0755)
	failOnError(t, err)

	err = ioutil.WriteFile("testdata/symlinkdata/file1.txt",
		[]byte(`This file is for file permission testing 2`), 0755)
	failOnError(t, err)

	// preparing symlink for test
	err = os.Symlink(fileName, newName1)
	failOnError(t, err)

	err = os.Symlink(fileName, newName2)
	failOnError(t, err)

	err = os.Symlink(filepath.Join(pwd, "testdata/symlinkdata"), newName3)
	failOnError(t, err)

	err = CopyDir("/tmp/symlinktest", "testdata", Excludes{})
	failOnError(t, err)
}

func TestExcludes(t *testing.T) {
	errExcludes := Excludes{
		".*",
		"DS_Store.bak",
		"[^",
		"[]a]",
	}
	assertEqual(t, "TestExcludes fail", true, errExcludes.Validate() != nil)

	excludes := Excludes{
		".*",
		"*.bak",
		"*.tmp",
		"tmp",
	}
	assertEqual(t, "TestExcludes success", true, excludes.Validate() == nil)
}

func TestCopyFile(t *testing.T) {
	_, err := CopyFile("testdata/file-found.txt", "testdata/file-not-exists.txt")
	if err == nil {
		t.Error("Expected error got nil")
	}

	_, err = CopyFile("testdata/sample.txt", "testdata/sample.txt")
	if err == nil {
		t.Error("Expected error got nil")
	}

	_, err = CopyFile("/var/you-will-not-be-able-to-create.txt", "testdata/sample.txt")
	if err == nil {
		t.Error("Expected error got nil")
	}
}

func TestCopyDir(t *testing.T) {
	err := CopyDir("/tmp/target", "testdata/not-exists-dir", Excludes{})
	if err == nil {
		t.Error("Expected error got nil")
	}

	err = CopyDir("/tmp/target", "testdata/sample.txt", Excludes{})
	if err == nil {
		t.Error("Expected error got nil")
	}

	err = CopyDir("/tmp", "testdata", Excludes{})
	if err == nil {
		t.Error("Expected error got nil")
	}

	err = CopyDir("/tmp/target", "testdata", Excludes{"[]a]"})
	if err == nil {
		t.Error("Expected error got nil")
	}

	pwd, _ := os.Getwd()
	err = CopyDir("/tmp/test1", pwd, Excludes{"test*", "*conf", ".*"})
	failOnError(t, err)

	removeAllFiles("/tmp/test1")
}
