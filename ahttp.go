// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/ahttp source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

// Package ahttp is to cater HTTP helper methods for aah framework.
// Like parse HTTP headers, ResponseWriter, content type, etc.
package ahttp

// Version no. of aah framework ahttp library
const Version = "0.4"

// HTTP Method names
const (
	MethodGet     = "GET"
	MethodHead    = "HEAD"
	MethodOptions = "OPTIONS"
	MethodPost    = "POST"
	MethodPut     = "PUT"
	MethodPatch   = "PATCH"
	MethodDelete  = "DELETE"
	MethodConnect = "CONNECT"
	MethodTrace   = "TRACE"
)

type (
	// Locale value is negotiated from HTTP header `Accept-Language`
	Locale struct {
		Raw      string
		Language string
		Region   string
	}
)
