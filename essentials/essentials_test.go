// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/essentials source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ess

import (
	"os"
	"path/filepath"
)

func getTestdataPath() string {
	pwd, _ := os.Getwd()
	return filepath.Join(pwd, "testdata")
}

func join(elem ...string) string {
	return filepath.Join(elem...)
}
