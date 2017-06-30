// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package crypto

import "golang.org/x/crypto/bcrypt"

// BcryptEncoder struct implements `PasswordEncoder` interface for `bycrpt`
// hashing.
type BcryptEncoder struct {
}

// Compare method compares given hash and password using bcrypt.
func (be *BcryptEncoder) Compare(hash, password []byte) bool {
	err := bcrypt.CompareHashAndPassword(hash, password)
	return err == nil
}
