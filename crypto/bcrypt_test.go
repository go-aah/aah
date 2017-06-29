// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package crypto

import (
	"testing"

	"aahframework.org/test.v0/assert"
)

func TestBcryptHashing(t *testing.T) {
	hashPassword, _ := BcryptGeneratePassword("welcome@123", 10)
	result := BcryptComparePassword([]byte(hashPassword), []byte("welcome@123"))
	assert.True(t, result)

	hashPassword, _ = BcryptGeneratePassword("welcome@123", 3)
	result = BcryptComparePassword([]byte(hashPassword), []byte("welcome@123"))
	assert.True(t, result)

	hashPassword, _ = BcryptGeneratePassword("welcome@123", 10)
	result = BcryptComparePassword([]byte(hashPassword), []byte("welcome@1234"))
	assert.False(t, result)
}
