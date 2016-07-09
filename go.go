// Copyright (c) 2016 Jeevanandam M (https://github.com/jeevatkm)
// essentails source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ess

import (
	"errors"
	"go/build"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Required variables
var (
	ErrGoPathIsNotSet = errors.New("GOPATH environment variable is not set. " +
		"Please refer to https://golang.org/doc/code.html to configure your Go environment")

	ErrDirNotInGoPath = errors.New("current directory is outside of GOPATH")
)

// LookExecutable looks for an executable binary named file
// in the directories named by the PATH environment variable.
func LookExecutable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// IsImportPathExists returns true if import path found in the GOPATH
// otherwise returns false
func IsImportPathExists(path string) bool {
	_, err := build.Import(path, "", build.FindOnly)
	return err == nil
}

// GoPath returns GOPATH in context with current working directory otherwise
// it returns first directory from GOPATH
func GoPath() (string, error) {
	gopath := build.Default.GOPATH
	if StrIsEmpty(gopath) {
		return "", ErrGoPathIsNotSet
	}

	var currentGoPath string
	workingDir, _ := os.Getwd()
	goPathList := filepath.SplitList(gopath)
	for _, path := range goPathList {
		if strings.HasPrefix(strings.ToLower(workingDir), strings.ToLower(path)) {
			currentGoPath = path
			break
		}

		path, _ = filepath.EvalSymlinks(path)
		if len(path) > 0 && strings.HasPrefix(strings.ToLower(workingDir), strings.ToLower(path)) {
			currentGoPath = path
			break
		}
	}

	if StrIsEmpty(currentGoPath) {
		// current working dir didn't match up,
		// so pick first one
		currentGoPath = goPathList[0]
	}

	return currentGoPath, nil
}
