// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package session

import (
	"bytes"
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
