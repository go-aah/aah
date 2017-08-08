// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/ahttp source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

// Package ahttp is to cater HTTP helper methods for aah framework.
// Like parse HTTP headers, ResponseWriter, content type, etc.
package ahttp

import (
	"io"
	"net/http"
)

// Version no. of aah framework ahttp library
const Version = "0.10-dev"

// HTTP Method names
const (
	MethodGet     = http.MethodGet
	MethodHead    = http.MethodHead
	MethodOptions = http.MethodOptions
	MethodPost    = http.MethodPost
	MethodPut     = http.MethodPut
	MethodPatch   = http.MethodPatch
	MethodDelete  = http.MethodDelete
	MethodConnect = http.MethodConnect
	MethodTrace   = http.MethodTrace
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

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package methods
//___________________________________

// AcquireRequest method populates the given aah framework `ahttp.Request`
// instance from Go HTTP request.
func AcquireRequest(r *http.Request) *Request {
	req := requestPool.Get().(*Request)
	return ParseRequest(r, req)
}

// ReleaseRequest method resets the instance value and puts back to pool.
func ReleaseRequest(r *Request) {
	if r != nil {
		r.Reset()
		requestPool.Put(r)
	}
}

// AcquireResponseWriter method wraps given writer and returns the aah response writer.
func AcquireResponseWriter(w http.ResponseWriter) ResponseWriter {
	rw := responsePool.Get().(*Response)
	rw.w = w
	return rw
}

// ReleaseResponseWriter method puts response writer back to pool.
func ReleaseResponseWriter(aw ResponseWriter) {
	if aw != nil {
		if gw, ok := aw.(*GzipResponse); ok {
			releaseGzipResponse(gw)
		} else {
			releaseResponse(aw.(*Response))
		}
	}
}

// WrapGzipWriter wraps `ahttp.ResponseWriter` with Gzip writer.
func WrapGzipWriter(w io.Writer) ResponseWriter {
	gr := grPool.Get().(*GzipResponse)
	gr.gw = acquireGzipWriter(w)
	gr.r = w.(*Response)
	return gr
}
