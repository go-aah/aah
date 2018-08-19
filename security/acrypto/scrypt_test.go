// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package acrypto

import (
	"testing"

	"aahframework.org/config.v0"
	"aahframework.org/test.v0/assert"
)

func TestScryptHashing(t *testing.T) {
	// invalid n (cpu/memory cost)
	se := &ScryptEncoder{n: 0, r: 8, p: 1, saltLen: 16, dkLen: 32}
	_, err := se.Generate([]byte("welcome123"))
	assert.NotNil(t, err)
	assert.Equal(t, "scrypt: N must be > 1 and a power of 2", err.Error())

	passEncoders = make(map[string]PasswordEncoder)
	cfg, _ := config.ParseString(`
		security {
			password_encoder {
				scrypt {
					enable = true
		    }
			}
		}
	`)

	err = InitPasswordEncoders(cfg)
	assert.Nil(t, err)

	encoder := PasswordAlgorithm("scrypt")
	assert.NotNil(t, encoder)

	scryptHash, err := encoder.Generate([]byte("welcome123"))
	assert.Nil(t, err)

	result := encoder.Compare(scryptHash, []byte("welcome123"))
	assert.True(t, result)

	result = encoder.Compare(scryptHash, []byte("welcome@123"))
	assert.False(t, result)

	// invalid hash
	hash1 := "x$32768$8$1$c67b04822659c899f34588f750def326$969463d494aaadec7d92f93fe52663a9bf6e8679466153ef645e6dd1564e09bb"
	result = encoder.Compare([]byte(hash1), []byte("welcome123"))
	assert.False(t, result)

	hash2 := "x$8$1$c67b04822659c899f34588f750def326$969463d494aaadec7d92f93fe52663a9bf6e8679466153ef645e6dd1564e09bb"
	result = encoder.Compare([]byte(hash2), []byte("welcome123"))
	assert.False(t, result)

	hash3 := "32768$x$1$c67b04822659c899f34588f750def326$969463d494aaadec7d92f93fe52663a9bf6e8679466153ef645e6dd1564e09bb"
	result = encoder.Compare([]byte(hash3), []byte("welcome123"))
	assert.False(t, result)

	hash4 := "32768$8$x$c67b04822659c899f34588f750def326$969463d494aaadec7d92f93fe52663a9bf6e8679466153ef645e6dd1564e09bb"
	result = encoder.Compare([]byte(hash4), []byte("welcome123"))
	assert.False(t, result)

	hash5 := "32768$8$1$c7b04822659c899f34588f750def326$969463d494aaadec7d92f93fe52663a9bf6e8679466153ef645e6dd1564e09bb"
	result = encoder.Compare([]byte(hash5), []byte("welcome123"))
	assert.False(t, result)

	hash6 := "32768$8$1$c67b04822659c899f34588f750def326$69463d494aaadec7d92f93fe52663a9bf6e8679466153ef645e6dd1564e09bb"
	result = encoder.Compare([]byte(hash6), []byte("welcome123"))
	assert.False(t, result)

	hash7 := "0$8$1$c67b04822659c899f34588f750def326$969463d494aaadec7d92f93fe52663a9bf6e8679466153ef645e6dd1564e09bb"
	result = encoder.Compare([]byte(hash7), []byte("welcome123"))
	assert.False(t, result)
}
