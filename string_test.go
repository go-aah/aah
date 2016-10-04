// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/essentails source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ess

import (
	"testing"

	"aahframework.org/test/assert"
)

func TestStringEmptyNotEmpty(t *testing.T) {
	assert.Equal(t, false, IsStrEmpty("    Welcome    "))

	assert.Equal(t, true, IsStrEmpty("        "))
}
