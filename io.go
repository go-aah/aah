// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/essentials source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ess

import "io"

// CloseQuietly closes `io.Closer` quietly. Very handy, where you do not care
// about error while `Close()` and helpful for code quality too.
func CloseQuietly(c ...interface{}) {
	for _, v := range c {
		if d, ok := v.(io.Closer); ok {
			_ = d.Close()
		}
	}
}
