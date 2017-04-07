// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/essentials source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ess

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	mrand "math/rand"
	"sync"
	"time"
)

const (
	letterBytes   = "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

var (
	mRandSrc mrand.Source
	mr       *sync.Mutex
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Random String methods
//___________________________________

// RandomString method generates the random string for given length using
// `crypto/rand`.
func RandomString(length int) string {
	return hex.EncodeToString(GenerateRandomKey(length / 2))
}

// RandomStringbm method generates the random string for given length using
// `math/rand.Source` and byte mask.
func RandomStringbm(length int) string {
	return string(GenerateRandomKeybm(length))
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Random key methods
//___________________________________

// GenerateRandomKey method generates the random bytes for given length using
// `crypto/rand`.
func GenerateRandomKey(length int) []byte {
	k := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, k); err != nil {
		// fallback to math based random key generater
		return GenerateRandomKeybm(length)
	}
	return k
}

// GenerateRandomKeybm method generates the random bytes for given length using
// `math/rand.Source` and byte mask.
// StackOverflow Ref - http://stackoverflow.com/a/31832326
func GenerateRandomKeybm(length int) []byte {
	b := make([]byte, length)
	// A randSrc() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := length-1, randSrc(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = randSrc(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return b
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

func randSrc() int64 {
	mr.Lock()
	defer mr.Unlock()
	return mRandSrc.Int63()
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// init
//___________________________________

func init() {
	mRandSrc = mrand.NewSource(time.Now().UnixNano())
	mr = &sync.Mutex{}
}
