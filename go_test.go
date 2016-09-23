// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/essentails source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ess

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-aah/test/assert"
)

func TestLookExecutable(t *testing.T) {
	assert.Equal(t, true, LookExecutable("go"))

	assert.Equal(t, false, LookExecutable("mygo"))
}

func TestIsImportPathExists(t *testing.T) {
	assert.Equal(t, true, IsImportPathExists("github.com/go-aah/essentials"))

	assert.Equal(t, false, IsImportPathExists("github.com/go-aah/unknown"))
}

func TestGoPath(t *testing.T) {
	gopath, err := GoPath()
	assert.FailOnError(t, err, "")
	t.Logf("gopath: %v", gopath)
}

func TestIsInGoRoot(t *testing.T) {
	goroot := os.Getenv("GOROOT")
	if IsStrEmpty(goroot) {
		goroot = "/usr/local/go"
	}
	assert.Equal(t, true, IsInGoRoot(filepath.Join(goroot, "src/github.com/jeevatkm/myapp")))

	assert.Equal(t, false, IsInGoRoot("/usr/local/"))
}
