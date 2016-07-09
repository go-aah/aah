// Copyright (c) 2016 Jeevanandam M (https://github.com/jeevatkm)
// resty source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ess

import "testing"

func TestMkDirAll(t *testing.T) {
	removeAllFiles("testdata/path")

	err := MkDirAll("testdata/path/to/create", 0755)
	failOnError(t, err)

	err = MkDirAll("testdata/path/to/create/for/test", 0755)
	failOnError(t, err)

	err = MkDirAll("testdata/path/to/create/for/test", 0755)
	failOnError(t, err)
}
