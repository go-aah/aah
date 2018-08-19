// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package acrypto

import (
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"aahframework.org/essentials"
	"golang.org/x/crypto/scrypt"
)

// ScryptEncoder struct implements `PasswordEncoder` interface for `scrypt`
// hashing.
type ScryptEncoder struct {
	n       int // cpu/memory cost
	r       int // blocksize
	p       int // parallelization
	saltLen int // random salt bytes length
	dkLen   int // derived key length
}

// Generate method returns the `scrypt` password hash based on configured
// values at `security.password_encoder.scrypt.*`.
func (se *ScryptEncoder) Generate(password []byte) ([]byte, error) {
	salt := ess.GenerateSecureRandomKey(se.saltLen)

	dkHash, err := scrypt.Key(password, salt, se.n, se.r, se.p, se.dkLen)
	if err != nil {
		return nil, err
	}

	// Format: n-cpu/mem-cost$r-blocksize$p-parallelization$salt$derived-key-hash
	return []byte(fmt.Sprintf("%d$%d$%d$%s$%s", se.n, se.r, se.p,
		base64.URLEncoding.EncodeToString(salt),
		base64.URLEncoding.EncodeToString(dkHash))), err
}

// Compare method compares given hash password and password using `scrypt`.
func (se *ScryptEncoder) Compare(hash, password []byte) bool {
	parts := strings.Split(string(hash), hashDelim)
	if len(parts) != 5 {
		// invalid hash
		return false
	}

	n, err := strconv.Atoi(parts[0])
	if err != nil {
		// invalid hash
		return false
	}

	r, err := strconv.Atoi(parts[1])
	if err != nil {
		// invalid hash
		return false
	}

	p, err := strconv.Atoi(parts[2])
	if err != nil {
		// invalid hash
		return false
	}

	salt, err := base64.URLEncoding.DecodeString(parts[3])
	if err != nil {
		// invalid hash
		return false
	}

	dkHash, err := base64.URLEncoding.DecodeString(parts[4])
	if err != nil {
		// invalid hash
		return false
	}

	otherHash, err := scrypt.Key(password, salt, n, r, p, len(dkHash))
	if err != nil {
		// unable to generate other hash from cleartext
		return false
	}

	return (subtle.ConstantTimeCompare(dkHash, otherHash) == 1)
}
