// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/essentials source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ess

import (
	"runtime"
	"testing"

	"aahframework.org/test/assert"
)

func TestMkDirAll(t *testing.T) {
	testdataPath := getTestdataPath()
	defer DeleteFiles(join(testdataPath, "path"))

	err := MkDirAll(join(testdataPath, "path", "to", "create"), 0755)
	assert.FailOnError(t, err, "")

	err = MkDirAll(join(testdataPath, "path", "to", "create", "for", "test"), 0755)
	assert.FailOnError(t, err, "")

	err = MkDirAll(join(testdataPath, "path", "to", "create", "for", "test"), 0755)
	assert.FailOnError(t, err, "")

	if runtime.GOOS != "windows" {
		err = MkDirAll("/var/testdata/[^[]", 0755)
		assert.NotNil(t, err)
	}
}
