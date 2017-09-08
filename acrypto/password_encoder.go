// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package acrypto

import (
	"errors"
	"fmt"

	"aahframework.org/config.v0"
)

const hashDelim = "$"

var (
	// ErrPasswordEncoderIsNil returned when given password encoder instance is nil.
	ErrPasswordEncoderIsNil = errors.New("acrypto: password encoder is nil")

	passEncoders = make(map[string]PasswordEncoder)
)

// PasswordEncoder interface is used to generate password hash and compare given hash & password
// based chosen hashing type. Such as `bcrypt`, `scrypt` and `pbkdf2`.
//
// Good read about hashing security https://crackstation.net/hashing-security.htm
type PasswordEncoder interface {
	Generate(password []byte) ([]byte, error)
	Compare(hash, password []byte) bool
}

// PasswordAlgorithm method returns the password encoder for given algorithm,
// Otherwise nil. Out-of-the-box supported passowrd algorithms are `bcrypt`, `scrypt`
// and `pbkdf2`. You can add your own if need be via method `AddPasswordEncoder`.
func PasswordAlgorithm(alg string) PasswordEncoder {
	if pe, found := passEncoders[alg]; found {
		return pe
	}
	return nil
}

// AddPasswordAlgorithm method is add password algorithm to encoders list.
// Implementation have to implement interface `PasswordEncoder`.
func AddPasswordAlgorithm(name string, pe PasswordEncoder) error {
	if pe == nil {
		return ErrPasswordEncoderIsNil
	}

	if _, found := passEncoders[name]; found {
		return fmt.Errorf("acrypto: password encoder '%v' is already added", name)
	}

	passEncoders[name] = pe

	return nil
}

// InitPasswordEncoders method initializes the password encoders based defined
// configuration in `security.password_encoder { ... }`
func InitPasswordEncoders(cfg *config.Config) error {
	keyPrefix := "security.password_encoder"

	// bcrypt algorithm config
	if cfg.BoolDefault(keyPrefix+".bcrypt.enable", true) {
		bcryptCost := cfg.IntDefault("key", 10)
		if err := AddPasswordAlgorithm("bcrypt", &BcryptEncoder{cost: bcryptCost}); err != nil {
			return err
		}
	}

	return nil
}
