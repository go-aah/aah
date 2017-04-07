// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/session source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package session

import (
	"bytes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/gob"

	"aahframework.org/essentials.v0"
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Encode/Decode Gob methods
//___________________________________

// encodeGob method encodes value into gob
func encodeGob(v interface{}) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// decodeGob method decodes given bytes into destination object.
func decodeGob(dst interface{}, src []byte) error {
	dec := gob.NewDecoder(bytes.NewBuffer(src))
	return dec.Decode(dst)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Encryption/Decryption methods
//___________________________________

// encrypt method encrypts a given value using the given block in CTR mode.
func encrypt(block cipher.Block, value []byte) []byte {
	iv := ess.GenerateRandomKey(block.BlockSize())

	// encrypt it
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(value, value)

	// iv + encryptedtext
	return append(iv, value...)
}

// decrypt method decrypts a given value using the given block in CTR mode.
func decrypt(block cipher.Block, value []byte) ([]byte, error) {
	size := block.BlockSize()
	if len(value) > size {
		// split iv and encryptedtext
		iv := value[:size]
		value = value[size:]

		// decrypt it
		stream := cipher.NewCTR(block, iv)
		stream.XORKeyStream(value, value)

		return value, nil
	}
	return nil, ErrUnableToDecrypt
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Signing/Verify methods
//___________________________________

// sign method signs a given value using HMAC.
func sign(key, value []byte) []byte {
	mac := hmac.New(sha256.New, key)
	defer mac.Reset()
	_, _ = mac.Write(value)
	return mac.Sum(nil)
}

// verify method verifies given key, value and mac is valid. If valid
// it returns true otherwise false.
func verify(key, value, mac1 []byte) bool {
	mac2 := sign(key, value)
	return hmac.Equal(mac1, mac2)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Encode/Decode Base64 methods
//___________________________________

// encodeBase64 method encodes a value using base64.
func encodeBase64(v []byte) []byte {
	encoded := make([]byte, base64.URLEncoding.EncodedLen(len(v)))
	base64.URLEncoding.Encode(encoded, v)
	return encoded
}

// decodeBase64 method decodes a value using base64.
func decodeBase64(v []byte) ([]byte, error) {
	decoded := make([]byte, base64.URLEncoding.DecodedLen(len(v)))
	b, err := base64.URLEncoding.Decode(decoded, v)
	if err != nil {
		return nil, ErrBase64Decode
	}
	return decoded[:b], nil
}
