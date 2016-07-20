// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/essentails source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ess

import "os"

func removeFiles(files ...string) {
	for _, f := range files {
		_ = os.Remove(f)
	}
}

func removeAllFiles(files ...string) {
	for _, f := range files {
		_ = os.RemoveAll(f)
	}
}
