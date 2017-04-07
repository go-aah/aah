// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/essentials source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ess

import (
	"testing"

	"aahframework.org/test.v0/assert"
)

func TestEssRandomKey(t *testing.T) {
	key1 := GenerateRandomKey(32)
	assert.NotNil(t, key1)
	assert.True(t, len(key1) == 32)

	key2 := GenerateRandomKeybm(64)
	assert.NotNil(t, key2)
	assert.True(t, len(key2) == 64)
}

func TestEssRandomString(t *testing.T) {
	str1 := RandomString(32)
	assert.True(t, len(str1) == 32)
	assert.NotNil(t, str1)

	str2 := RandomStringbm(32)
	assert.True(t, len(str2) == 32)
	assert.NotNil(t, str2)
}

func BenchmarkGenerateRandomKey(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GenerateRandomKey(16)
	}
}

func BenchmarkGenerateRandomKeybm(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GenerateRandomKeybm(32)
	}
}
