// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package acrypto

import (
	"crypto/aes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHMACSignAndVerify(t *testing.T) {
	key := "467b2d53632646a0a9c6cc0d498a7559"
	text := "This is the text gonna be signed and verifed"

	signedText := SignString(key, text, "sha-512")
	result, err := VerifyString(key, text, signedText, "sha-512")
	assert.Nil(t, err)
	assert.True(t, result)

	result, err = VerifyString(key, text, signedText[:86], "sha-512")
	assert.NotNil(t, err)
	assert.Equal(t, "illegal base64 data at input byte 84", err.Error())
	assert.False(t, result)

	// Errors
	st := "pxed-Ed1Q4zciwaJ9goDOk4pRP3O1Xqh7uR03AqKeEo_n1TOxwWwe7bv7AIB_cjHSR5CUP-q7FZ7OnXPgLmY8g=="
	result, err = VerifyString(key, text, st, "sha-512")
	assert.Nil(t, err)
	assert.False(t, result)
}

func TestAESEncrptAndDecrypt(t *testing.T) {
	key := "467b2d53632646a0a9c6cc0d498a7559"
	text := "This is the text gonna be encrypted and decrypted"

	encryptedText, err := AESEncryptString(key, text)
	assert.Nil(t, err)

	result, err := AESDecryptString(key, encryptedText)
	assert.Nil(t, err)
	assert.Equal(t, text, result)

	// Errors
	encryptedText, err = AESEncryptString(key[:30], text)
	assert.NotNil(t, err)
	assert.Equal(t, "crypto/aes: invalid key size 30", err.Error())
	assert.Equal(t, "", encryptedText)

	result, err = AESDecryptString(key[:30], encryptedText)
	assert.NotNil(t, err)
	assert.Equal(t, "crypto/aes: invalid key size 30", err.Error())
	assert.Equal(t, "", result)

	result, err = AESDecryptString(key, "57kKnKkUEffzst7rn3YNGlBQqlTPcKePC7Ew-V3idrES8A9U7NVCebQHbd9UoFEuAMWjfmMRJAPDcSiaoxButT4")
	assert.NotNil(t, err)
	assert.Equal(t, "illegal base64 data at input byte 84", err.Error())
	assert.Equal(t, "", result)

	ciperBlock, _ := aes.NewCipher([]byte(key))
	b, err1 := AESDecrypt(ciperBlock, []byte("text"))
	assert.NotNil(t, err1)
	assert.Nil(t, b)
	assert.True(t, (ErrUnableToDecrypt == err1))
}
