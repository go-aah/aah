// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package acrypto

import (
	"testing"

	"golang.org/x/crypto/bcrypt"

	"aahframework.org/test.v0/assert"
)

func TestBcryptHashing(t *testing.T) {
	bcryptEncoder := BcryptEncoder{}
	hashPassword, _ := bcrypt.GenerateFromPassword([]byte("welcome@123"), 10)
	result := bcryptEncoder.Compare(hashPassword, []byte("welcome@123"))
	assert.True(t, result)
}
