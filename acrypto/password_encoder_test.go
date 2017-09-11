// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package acrypto

import (
	"testing"

	"aahframework.org/test.v0/assert"
)

func TestCryptoPasswordAlgrothim(t *testing.T) {
	passEncoders = make(map[string]PasswordEncoder)
	err := AddPasswordAlgorithm("test1", nil)
	assert.Equal(t, ErrPasswordEncoderIsNil, err)

	err = AddPasswordAlgorithm("bcrypt", &BcryptEncoder{})
	assert.Nil(t, err)

	err = AddPasswordAlgorithm("bcrypt", &BcryptEncoder{})
	assert.Nil(t, err)

	assert.Nil(t, PasswordAlgorithm("notexists"))
}
