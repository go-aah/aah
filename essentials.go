// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org/essentials source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ess

// Valuer interface is general purpose to `Set` and `Get` operations.
type Valuer interface {
	Get(key string) interface{}
	Set(key string, value interface{})
}
