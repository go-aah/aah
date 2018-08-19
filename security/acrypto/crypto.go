// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package acrypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"hash"
	"strings"

	"aahframework.org/essentials.v0"
)

var (
	// ErrUnableToDecrypt returned for decrypt errors.
	ErrUnableToDecrypt = errors.New("security/crypto: unable to decrypt")
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package Encrypt/Decrypt methods
//___________________________________

// AESEncryptString is convenient method to do AES encryption.
//
// The key argument should be the AES key, either 16, 24, or 32 bytes
// to select AES-128, AES-192, or AES-256.
func AESEncryptString(key, text string) (string, error) {
	ciperBlock, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}
	encrypted := AESEncrypt(ciperBlock, []byte(text))
	return base64.URLEncoding.EncodeToString(encrypted), nil
}

// AESDecryptString is convenient method to do AES decryption.
// It decrypts the encrypted text with given key.
func AESDecryptString(key, encryptedText string) (string, error) {
	ciperBlock, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	bytes, err := base64.URLEncoding.DecodeString(encryptedText)
	if err != nil {
		return "", err
	}

	textBytes, err := AESDecrypt(ciperBlock, bytes)
	if err != nil {
		return "", err
	}

	return string(textBytes), nil
}

// AESEncrypt method encrypts a given value with given key block in CTR mode.
func AESEncrypt(block cipher.Block, value []byte) []byte {
	iv := ess.GenerateSecureRandomKey(block.BlockSize())

	// encrypt it
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(value, value)

	// iv + encryptedtext
	return append(iv, value...)
}

// AESDecrypt method decrypts a given value with the given key block in CTR mode.
func AESDecrypt(block cipher.Block, value []byte) ([]byte, error) {
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
// Package Sign/Verify methods
//___________________________________

// SignString method signs the given text using provided key with HMAC SHA.
//
// Supported SHA's are SHA-1, SHA-224, SHA-256, SHA-384, SHA-512.
func SignString(key, text, sha string) string {
	return base64.URLEncoding.EncodeToString(Sign([]byte(key), []byte(text), sha))
}

// VerifyString method verifies the signed text and text using provide key with
// HMAC SHA. Returns true if sign is valid otherwise false.
//
// Supported SHA's are SHA-1, SHA-224, SHA-256, SHA-384, SHA-512.
func VerifyString(key, text, signedText, sha string) (bool, error) {
	macText, err := base64.URLEncoding.DecodeString(signedText)
	if err != nil {
		return false, err
	}
	return Verify([]byte(key), []byte(text), macText, sha), nil
}

// Sign method signs a given value using HMAC and given SHA name.
func Sign(key, value []byte, sha string) []byte {
	mac := hmac.New(hashFunc(sha), key)
	defer mac.Reset()
	_, _ = mac.Write(value)
	return mac.Sum(nil)
}

// Verify method verifies given key, value and mac is valid. If valid
// it returns true otherwise false.
func Verify(key, value, mac []byte, sha string) bool {
	otherMac := Sign(key, value, sha)
	return hmac.Equal(mac, otherMac)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

func hashFunc(alg string) func() hash.Hash {
	switch strings.ToLower(alg) {
	case "sha-512":
		return sha512.New
	case "sha-384":
		return sha512.New384
	case "sha-256":
		return sha256.New
	case "sha-224":
		return sha256.New224
	case "sha-1":
		return sha1.New
	default:
		return nil
	}
}
