// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package session

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
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
