// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/essentials source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ess

import (
	"encoding/base64"
	"errors"
)

// ErrBase64Decode returned when given string unable to do base64 decode.
var ErrBase64Decode = errors.New("encoding/base64: decode error")

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Encode/Decode Base64 methods
//___________________________________

// EncodeToBase64 method encodes given bytes into base64 bytes.
// Reference: https://github.com/golang/go/blob/master/src/encoding/base64/base64.go#L169
func EncodeToBase64(v []byte) []byte {
	encoded := make([]byte, base64.URLEncoding.EncodedLen(len(v)))
	base64.URLEncoding.Encode(encoded, v)
	return encoded
}

// DecodeBase64 method decodes given base64 into bytes.
// Reference: https://github.com/golang/go/blob/master/src/encoding/base64/base64.go#L384
func DecodeBase64(v []byte) ([]byte, error) {
	decoded := make([]byte, base64.URLEncoding.DecodedLen(len(v)))
	b, err := base64.URLEncoding.Decode(decoded, v)
	if err != nil {
		return nil, ErrBase64Decode
	}
	return decoded[:b], nil
}
