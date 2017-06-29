// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package crypto

import "golang.org/x/crypto/bcrypt"

// BcryptComparePassword method compares given hash and password is equal. If
// equal it returns true otherwise false.
func BcryptComparePassword(hash, password []byte) bool {
	err := bcrypt.CompareHashAndPassword(hash, password)
	return err == nil
}

// BcryptGeneratePassword method generates the `bcrypt` hash password for
// given string. Minimum cost is 4 and Maximum cost is 31.
// It recommended to use cost value 10 and above.
func BcryptGeneratePassword(password string, cost int) (string, error) {
	if cost < bcrypt.MinCost {
		// If it is below 4 set it 4.
		cost = bcrypt.MinCost
	}

	bytes, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	return string(bytes), err
}
