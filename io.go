// Copyright (c) 2016 Jeevanandam M (https://github.com/jeevatkm)
// essentails source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ess

import "io"

// CloseQuietly closes `io.Closer` quietly. Very handy and helpful for code
// quality too.
func CloseQuietly(v interface{}) {
	if d, ok := v.(io.Closer); ok {
		_ = d.Close()
	}
}
