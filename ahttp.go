// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/ahttp source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

// Package ahttp is to cater HTTP helper methods for aah framework.
// Like parse HTTP headers, ResponseWriter, content type, etc.
package ahttp

import "net/http"

// Version no. of aah framework ahttp library
const Version = "0.7"

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

// TimeFormat is the time format to use when generating times in HTTP
// headers. It is like time.RFC1123 but hard-codes GMT as the time
// zone. The time being formatted must be in UTC for Format to
// generate the correct format.
const TimeFormat = http.TimeFormat

type (
	// Locale value is negotiated from HTTP header `Accept-Language`
	Locale struct {
		Raw      string
		Language string
		Region   string
	}
)
