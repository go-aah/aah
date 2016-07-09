// Copyright (c) 2016 Jeevanandam M (https://github.com/jeevatkm)
// essentails source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ess

import (
	"os"
	"testing"
)

func TestCloseQuietly(t *testing.T) {
	file, err := os.Open("testdata/sample.txt")

	failOnError(t, err)

	CloseQuietly(file)
}
