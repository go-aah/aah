// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package acrypto

import (
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"aahframework.org/essentials"
	"golang.org/x/crypto/pbkdf2"
)

// Pbkdf2Encoder struct implements `PasswordEncoder` interface for `pbkdf2`
// hashing.
type Pbkdf2Encoder struct {
	iter    int    // no. of iteration
	dkLen   int    // derived key length
	saltLen int    // random salt bytes length
	hashAlg string // hash algorithm such as sha-1, sha-224, sha-256, sha-384, sha-512
}

// Generate method returns `pbkdf2` password hash based on configured
// values at `security.password_encoder.pbkdf2.*`.
func (pe *Pbkdf2Encoder) Generate(password []byte) ([]byte, error) {
	salt := ess.GenerateSecureRandomKey(pe.saltLen)

	dkHash := pbkdf2.Key(password, salt, pe.iter, pe.dkLen, hashFunc(pe.hashAlg))

	// Format: hash-alg$iteration$salt$derived-key-hash
	return []byte(fmt.Sprintf("%s$%d$%s$%s", pe.hashAlg, pe.iter,
		base64.URLEncoding.EncodeToString(salt),
		base64.URLEncoding.EncodeToString(dkHash))), nil
}

// Compare method compares given hash password and password using `pbkdf2`.
func (pe *Pbkdf2Encoder) Compare(hash, password []byte) bool {
	parts := strings.Split(string(hash), hashDelim)
	if len(parts) != 4 {
		// invalid hash
		return false
	}

	hashAlg := parts[0]

	iter, err := strconv.Atoi(parts[1])
	if err != nil {
		// invalid hash
		return false
	}

	salt, err := base64.URLEncoding.DecodeString(parts[2])
	if err != nil {
		// invalid hash
		return false
	}

	dkHash, err := base64.URLEncoding.DecodeString(parts[3])
	if err != nil {
		// invalid hash
		return false
	}

	otherHash := pbkdf2.Key(password, salt, iter, len(dkHash), hashFunc(hashAlg))

	return (subtle.ConstantTimeCompare(dkHash, otherHash) == 1)
}
