// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package acrypto

import "fmt"

// PasswordEncoder interface is used to encode and compare given hash and password
// based chosen hashing type. Such as `bcrypt`, `sha1`, `sha256`, `sha512` and `md5`.
type PasswordEncoder interface {
	Compare(hash, password []byte) bool
}

// CreatePasswordEncoder method creates the instance of password encoder password,
// based on given type. Currently `bcrypt` is supported.
func CreatePasswordEncoder(etype string) (PasswordEncoder, error) {
	switch etype {
	case "bcrypt":
		return &BcryptEncoder{}, nil
	default:
		return nil, fmt.Errorf("acrypto: unsupported encoder type '%v'", etype)
	}
}
