// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org/essentials source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ess

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncodeBase64(t *testing.T) {
	value := "this is gonna be encoded in base64"

	valueBytes := EncodeToBase64([]byte(value))
	assert.NotNil(t, valueBytes)

	decoded, err := DecodeBase64(valueBytes)
	assert.Nil(t, err)
	assert.NotNil(t, decoded)

	_, err = DecodeBase64([]byte("dGhpcyBpcyBnb25uYSBiZSBlbmGVkIGluIGJhc2U2NA=="))
	assert.NotNil(t, err)
	assert.Equal(t, ErrBase64Decode, err)
}
