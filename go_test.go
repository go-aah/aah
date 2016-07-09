// Copyright (c) 2016 Jeevanandam M (https://github.com/jeevatkm)
// go-aah/essentails source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ess

import "testing"

func TestLookExecutable(t *testing.T) {
	assertEqual(t, "TestLookExecutable - go", true, LookExecutable("go"))

	assertEqual(t, "TestLookExecutable - mygo", false, LookExecutable("mygo"))
}

func TestIsImportPathExists(t *testing.T) {
	assertEqual(t, "TestIsImportPathExists - essentials", true, IsImportPathExists("github.com/go-aah/essentials"))

	assertEqual(t, "TestIsImportPathExists - unknown", false, IsImportPathExists("github.com/go-aah/unknown"))
}

func TestGoPath(t *testing.T) {
	gopath, err := GoPath()
	failOnError(t, err)
	t.Logf("gopath: %v", gopath)
}

func TestIsInGoRoot(t *testing.T) {
	assertEqual(t, "TestIsInGoRoot", true, IsInGoRoot("/usr/local/go/src/github.com/jeevatkm/myapp"))

	assertEqual(t, "TestIsInGoRoot", false, IsInGoRoot("/usr/local/"))
}
