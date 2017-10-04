// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package acrypto

import "golang.org/x/crypto/bcrypt"

// BcryptEncoder struct implements `PasswordEncoder` interface for `bcrypt`
// hashing.
type BcryptEncoder struct {
	cost int
}

// Generate method returns the `bcrypt` password hash based on configured
// cost at `security.password_encoder.bcrypt.*`.
func (be *BcryptEncoder) Generate(password []byte) ([]byte, error) {
	return bcrypt.GenerateFromPassword(password, be.cost)
}

// Compare method compares given password hash and password using bcrypt.
func (be *BcryptEncoder) Compare(hash, password []byte) bool {
	err := bcrypt.CompareHashAndPassword(hash, password)
	return err == nil
}
