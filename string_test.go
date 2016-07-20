// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// essentails source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ess

import (
	"testing"

	"github.com/go-aah/test/assert"
)

func TestStringEmptyNotEmpty(t *testing.T) {
	assert.Equal(t, false, StrIsEmpty("    Welcome    "))

	assert.Equal(t, true, StrIsEmpty("        "))
}
