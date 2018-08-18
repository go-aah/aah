// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/essentials source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ess

import (
	"os"
	"testing"

	"aahframework.org/test.v0/assert"
)

func TestCloseQuietly(t *testing.T) {
	file, err := os.Open("testdata/sample.txt")

	assert.FailOnError(t, err, "")

	CloseQuietly(file, nil)
}
