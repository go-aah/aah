// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package scheme

import (
	"testing"

	"aahframework.org/test.v0/assert"
)

func TestScheme(t *testing.T) {
	// Create auth scheme
	authScheme := New("form")
	assert.NotNil(t, authScheme)

	authScheme = New("basic")
	assert.NotNil(t, authScheme)

	authScheme = New("api")
	assert.Nil(t, authScheme)

	authScheme = New("unknown")
	assert.Nil(t, authScheme)
}
