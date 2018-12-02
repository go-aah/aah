// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

// Package ess provides a essentials and helper for the application
// development and usage. Such encoding, secure string, secure random string,
// filepath access, GUID, etc.
package ess

// Valuer interface is general purpose to `Set` and `Get` operations.
type Valuer interface {
	Get(key string) interface{}
	Set(key string, value interface{})
}
