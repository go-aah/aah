// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"testing"

	"aahframework.org/test.v0/assert"
)

func TestActualType(t *testing.T) {
	ct := actualType((*engine)(nil))
	assert.Equal(t, "aah.engine", ct.String())

	ct = actualType(engine{})
	assert.Equal(t, "aah.engine", ct.String())

	ct = actualType(&engine{})
	assert.Equal(t, "aah.engine", ct.String())
}
