// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/ahttp source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

// Package ahttp is to cater HTTP helper methods for aah framework.
// Like parse HTTP headers, ResponseWriter, content type, etc.
package ahttp

import (
	"io"
	"net"
	"net/http"
	"strings"

	"aahframework.org/essentials.v0"
)

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

// URI Protocol scheme names
const (
	SchemeHTTP  = "http"
	SchemeHTTPS = "https"
	SchemeFTP   = "ftp"
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
		r.cleanupMutlipart()
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

// Scheme method is to identify value of protocol value. It's is derived
// one, Go language doesn't provide directly.
//  - `X-Forwarded-Proto` is not empty, returns as-is
//  - `X-Forwarded-Protocol` is not empty, returns as-is
//  - `http.Request.TLS` is not nil or `X-Forwarded-Ssl == on` returns `https`
//  - `X-Url-Scheme` is not empty, returns as-is
//  - returns `http`
func Scheme(r *http.Request) string {
	if scheme := r.Header.Get(HeaderXForwardedProto); scheme != "" {
		return scheme
	}

	if scheme := r.Header.Get(HeaderXForwardedProtocol); scheme != "" {
		return scheme
	}

	if r.TLS != nil || r.Header.Get(HeaderXForwardedSsl) == "on" {
		return "https"
	}

	if scheme := r.Header.Get(HeaderXUrlScheme); scheme != "" {
		return scheme
	}

	return "http"
}

// Host method is to correct Hosyt source value from HTTP request.
func Host(r *http.Request) string {
	if r.URL.Host == "" {
		return r.Host
	}
	return r.URL.Host
}

// ClientIP method returns remote Client IP address aka Remote IP.
// It parses in the order of given set of headers otherwise it uses default
// default header set `X-Forwarded-For`, `X-Real-IP`, "X-Appengine-Remote-Addr"
// and finally `http.Request.RemoteAddr`.
func ClientIP(r *http.Request, hdrs ...string) string {
	if len(hdrs) == 0 {
		hdrs = []string{"X-Forwarded-For", "X-Real-IP", "X-Appengine-Remote-Addr"}
	}

	for _, hdrKey := range hdrs {
		if hv := r.Header.Get(hdrKey); !ess.IsStrEmpty(hv) {
			index := strings.Index(hv, ",")
			if index == -1 {
				return strings.TrimSpace(hv)
			}
			return strings.TrimSpace(hv[:index])
		}
	}

	// Remote Address
	if remoteAddr, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return strings.TrimSpace(remoteAddr)
	}

	return ""
}
