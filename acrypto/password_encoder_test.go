// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package acrypto

import (
	"strings"
	"testing"

	"aahframework.org/test.v0/assert"
)

func TestCryptoPasswordEncoder(t *testing.T) {
	encoder, err := CreatePasswordEncoder("bcrypt")
	assert.Nil(t, err)
	assert.NotNil(t, encoder)

	result := encoder.Compare([]byte("$2y$10$2A4GsJ6SmLAMvDe8XmTam.MSkKojdobBVJfIU7GiyoM.lWt.XV3H6"), []byte("welcome123"))
	assert.True(t, result)

	result = encoder.Compare([]byte("$2y$10$2A4GsJ6SmLAMvDe8XmTam.MSkKojdobBVJfIU7GiyoM.lWt.XV3H6"), []byte("nomatch"))
	assert.False(t, result)

	// Unsupport password encoder type
	encoder, err = CreatePasswordEncoder("sha256")
	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), "acrypto: unsupported encoder type"))
	assert.Nil(t, encoder)
}
