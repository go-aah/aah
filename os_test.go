// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/essentials source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ess

import (
	"testing"

	"aahframework.org/test/assert"
)

func TestMkDirAll(t *testing.T) {
	defer removeAllFiles("testdata/path")

	err := MkDirAll("testdata/path/to/create", 0755)
	assert.FailOnError(t, err, "")

	err = MkDirAll("testdata/path/to/create/for/test", 0755)
	assert.FailOnError(t, err, "")

	err = MkDirAll("testdata/path/to/create/for/test", 0755)
	assert.FailOnError(t, err, "")

	err = MkDirAll("/var/testdata/[^[]", 0755)
	assert.NotNil(t, err)
}
