// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org/essentials source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ess

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCloseQuietly(t *testing.T) {
	file, err := os.Open("testdata/sample.txt")

	assert.Nil(t, err, "")

	CloseQuietly(file, nil)
}
