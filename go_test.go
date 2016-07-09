// Copyright (c) 2016 Jeevanandam M (https://github.com/jeevatkm)
// resty source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ess

import (
	"fmt"
	"testing"
)

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

	fmt.Println("gopath:", gopath)
}
