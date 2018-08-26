// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ess

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBStrToBytes(t *testing.T) {
	checkBytesValue(t, "2b", int64(2))
	checkBytesValue(t, "2B", int64(2))
	checkBytesValue(t, "2b", int64(2))
	checkBytesValue(t, "2B", int64(2))
}

func TestKBStrToBytes(t *testing.T) {
	checkBytesValue(t, "2kb", int64(2048))
	checkBytesValue(t, "2KB", int64(2048))
	checkBytesValue(t, "2kib", int64(2048))
	checkBytesValue(t, "2KiB", int64(2048))
}

func TestMBStrToBytes(t *testing.T) {
	checkBytesValue(t, "2mb", int64(2097152))
	checkBytesValue(t, "2MB", int64(2097152))
	checkBytesValue(t, "2mib", int64(2097152))
	checkBytesValue(t, "2MiB", int64(2097152))
}

func TestGBStrToBytes(t *testing.T) {
	checkBytesValue(t, "2gb", int64(2147483648))
	checkBytesValue(t, "2GB", int64(2147483648))
	checkBytesValue(t, "2Gib", int64(2147483648))
	checkBytesValue(t, "2GiB", int64(2147483648))
}

func TestTBStrToBytes(t *testing.T) {
	checkBytesValue(t, "2tb", int64(2199023255552))
	checkBytesValue(t, "2TB", int64(2199023255552))
	checkBytesValue(t, "2Tib", int64(2199023255552))
	checkBytesValue(t, "2TiB", int64(2199023255552))
}

func TestErrStrToBytes(t *testing.T) {
	v1, err := StrToBytes("2")
	assert.NotNil(t, err)
	assert.Equal(t, "format: invalid input '2'", err.Error())
	assert.Equal(t, int64(0), v1)
}

func checkBytesValue(t *testing.T, value string, expt int64) {
	v, err := StrToBytes(value)
	assert.Nil(t, err)
	assert.Equal(t, expt, v)
}
