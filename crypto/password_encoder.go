// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package crypto

// PasswordEncoder interface is used to encode and compare given hash and password
// based chosen hashing type. Such as `bycrpt`, `sha1`, `sha256`, `sha512` and `md5`.
type PasswordEncoder interface {
	Compare(hash, password []byte) bool
}
