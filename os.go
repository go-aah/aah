// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/essentials source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ess

import (
	"fmt"
	"os"
)

// MkDirAll method creates nested directories with given permission if not exists
func MkDirAll(path string, mode os.FileMode) error {
	if _, err := os.Lstat(path); err != nil {
		if os.IsNotExist(err) {
			if err = os.MkdirAll(path, mode); err != nil {
				return fmt.Errorf("unable to create directory '%v': %v", path, err)
			}
		}
	}
	return nil
}
