// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package acrypto

import (
	"testing"

	"aahframework.org/config"
	"github.com/stretchr/testify/assert"
)

func TestBcryptHashing(t *testing.T) {
	passEncoders = make(map[string]PasswordEncoder)
	cfg, _ := config.ParseString(`
		security {
			password_encoder {
		    bcrypt {
		      cost = 10
		    }
			}
		}
	`)

	err := InitPasswordEncoders(cfg)
	assert.Nil(t, err)

	encoder := PasswordAlgorithm("bcrypt")
	assert.NotNil(t, encoder)

	hashPassword, err := encoder.Generate([]byte("welcome@123"))
	assert.Nil(t, err)

	result := encoder.Compare(hashPassword, []byte("welcome@123"))
	assert.True(t, result)
}
