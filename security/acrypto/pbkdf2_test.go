// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package acrypto

import (
	"testing"

	"aahframe.work/aah/config"
	"github.com/stretchr/testify/assert"
)

func TestPbkdf2Hashing(t *testing.T) {
	testSubPbkdf2ByHashAlg(t, `
		security {
			password_encoder {
		    pbkdf2 {
					enable = true
		    }
			}
		}
	`)

	testSubPbkdf2ByHashAlg(t, `
	security {
		password_encoder {
			pbkdf2 {
				enable = true
				hash_algorithm = "sha-1"
			}
		}
	}
`)

	testSubPbkdf2ByHashAlg(t, `
	security {
		password_encoder {
			pbkdf2 {
				enable = true
				hash_algorithm = "sha-224"
			}
		}
	}
`)

	testSubPbkdf2ByHashAlg(t, `
	security {
		password_encoder {
			pbkdf2 {
				enable = true
				hash_algorithm = "sha-256"
			}
		}
	}
`)

	testSubPbkdf2ByHashAlg(t, `
	security {
		password_encoder {
			pbkdf2 {
				enable = true
				hash_algorithm = "sha-384"
			}
		}
	}
`)

	cfgStr := `
security {
	password_encoder {
		pbkdf2 {
			enable = true
			hash_algorithm = "sha-34"
		}
	}
}
`

	passEncoders = make(map[string]PasswordEncoder)
	cfg, _ := config.ParseString(cfgStr)

	err := InitPasswordEncoders(cfg)
	assert.NotNil(t, err)
	assert.Equal(t, "acrypto/pbkdf2: invalid sha algorithm 'sha-34'", err.Error())
}

func TestPbkdf2InvalidHash(t *testing.T) {
	passEncoders = make(map[string]PasswordEncoder)
	cfg, _ := config.ParseString(`
		security {
			password_encoder {
		    pbkdf2 {
					enable = true
		    }
			}
		}
	`)

	err := InitPasswordEncoders(cfg)
	assert.Nil(t, err)

	encoder := PasswordAlgorithm("pbkdf2")
	assert.NotNil(t, encoder)

	hash1 := []byte("x$sha512$10000$7e9b5c05f17f6bab5b48f8bb102b9cb5e82f523bebfde43d$a4e2f455f20ca967fc5bc1f6cb84d1d4f79d8b784d77b4427ed29715c13faa70")
	result := encoder.Compare(hash1, []byte("welcome@123"))
	assert.False(t, result)

	hash2 := []byte("sha512$x$7e9b5c05f17f6bab5b48f8bb102b9cb5e82f523bebfde43d$a4e2f455f20ca967fc5bc1f6cb84d1d4f79d8b784d77b4427ed29715c13faa70")
	result = encoder.Compare(hash2, []byte("welcome@123"))
	assert.False(t, result)

	hash3 := []byte("sha512$10000$7e95c05f17f6bab5b48f8bb102b9cb5e82f523bebfde43d$a4e2f455f20ca967fc5bc1f6cb84d1d4f79d8b784d77b4427ed29715c13faa70")
	result = encoder.Compare(hash3, []byte("welcome@123"))
	assert.False(t, result)

	hash4 := []byte("sha512$10000$7e9b5c05f17f6bab5b48f8bb102b9cb5e82f523bebfde43d$a4e2f455f2ca967fc5bc1f6cb84d1d4f79d8b784d77b4427ed29715c13faa70")
	result = encoder.Compare(hash4, []byte("welcome@123"))
	assert.False(t, result)
}

func testSubPbkdf2ByHashAlg(t *testing.T, cfgStr string) {
	passEncoders = make(map[string]PasswordEncoder)
	cfg, _ := config.ParseString(cfgStr)

	err := InitPasswordEncoders(cfg)
	assert.Nil(t, err)

	encoder := PasswordAlgorithm("pbkdf2")
	assert.NotNil(t, encoder)

	hashPassword, err := encoder.Generate([]byte("welcome@123"))
	assert.Nil(t, err)

	result := encoder.Compare(hashPassword, []byte("welcome@123"))
	assert.True(t, result)
}
