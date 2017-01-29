// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ahttp

import (
	"fmt"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"strings"

	"aahframework.org/essentials"
)

const jsonpReqParamKey = "callback"

type (
	// Request is extends `http.Request` for aah framework
	Request struct {
		// Host value of the HTTP 'Host' header (e.g. 'example.com:8080').
		Host string

		// Method request method e.g. `GET`, `POST`, etc.
		Method string

		// Path the request URL Path e.g. `/booking/hotel.html`.
		Path string

		// Header request HTTP headers
		Header http.Header

		// Payload holds the value from HTTP request for `Content-Type`
		// JSON and XML.
		Payload string

		// ContentType the parsed HTTP header `Content-Type`.
		ContentType *ContentType

		// AcceptContentType negotiated value from HTTP Header `Accept`.
		// The resolve order is-
		// 1) URL extension
		// 2) Accept header.
		// Most quailfied one based on quality factor otherwise default is HTML.
		AcceptContentType *ContentType

		// AcceptEncoding negotiated value from HTTP Header the `Accept-Encoding`
		// Most quailfied one based on quality factor.
		AcceptEncoding AcceptSpec

		// Params contains values from Path, Query, Form and File.
		Params *Params

		// Referer value of the HTTP 'Referrer' (or 'Referer') header.
		Referer string

		// UserAgent value of the HTTP 'User-Agent' header.
		UserAgent string

		// ClientIP remote client IP address.
		ClientIP string

		// Locale negotiated value from HTTP Header `Accept-Language`.
		Locale *Locale

		// IsJSONP is true if request query string has "callback=function_name".
		IsJSONP bool

		// Raw an object that Go HTTP server provied, Direct interaction with
		// raw object is not encouraged.
		Raw *http.Request
	}

	// Params structure holds value of Path, Query, Form and File.
	Params struct {
		Path  map[string]string
		Query url.Values
		Form  url.Values
		File  map[string][]*multipart.FileHeader
	}
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Global methods
//___________________________________

// ParseRequest method populates the given aah framework `ahttp.Request`
// instance from Go HTTP request.
func ParseRequest(r *http.Request, req *Request) *Request {
	req.Host = r.Host
	req.Method = r.Method
	req.Path = r.URL.Path
	req.Header = r.Header
	req.ContentType = ParseContentType(r)
	req.AcceptContentType = NegotiateContentType(r)
	req.Params = &Params{Query: r.URL.Query()}
	req.Referer = getReferer(r.Header)
	req.UserAgent = r.Header.Get(HeaderUserAgent)
	req.ClientIP = ClientIP(r)
	req.Locale = NegotiateLocale(r)
	req.IsJSONP = isJSONPReqeust(r)
	req.Raw = r

	return req
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Request methods
//___________________________________

// Reset method resets request instance for reuse.
func (r *Request) Reset() {
	r.Header = nil
	r.ContentType = nil
	r.AcceptContentType = nil
	r.Params = nil
	r.Referer = ""
	r.UserAgent = ""
	r.ClientIP = ""
	r.Locale = nil
	r.IsJSONP = false
	r.Raw = nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Params methods
//___________________________________

// PathValue method return value for given Path param key otherwise empty string.
func (p *Params) PathValue(key string) string {
	if p.Path != nil {
		if value, found := p.Path[key]; found {
			return value
		}
	}
	return ""
}

// QueryValue method return value for given query (aka URL) param key
// otherwise empty string.
func (p *Params) QueryValue(key string) string {
	return p.Query.Get(key)
}

// QueryArrayValue method return array value for given query (aka URL)
// param key otherwise empty string.
func (p *Params) QueryArrayValue(key string) []string {
	if values, found := p.Query[key]; found {
		return values
	}
	return []string{}
}

// FormValue methos returns value for given form key otherwise empty string.
func (p *Params) FormValue(key string) string {
	if p.Form != nil {
		return p.Form.Get(key)
	}
	return ""
}

// FormArrayValue methos returns value for given form key otherwise empty string.
func (p *Params) FormArrayValue(key string) []string {
	if p.Form != nil {
		if values, found := p.Form[key]; found {
			return values
		}
	}
	return []string{}
}

// FormFile method returns the first file for the provided form key otherwise
// returns error. It is caller responsibility to close the file.
func (p *Params) FormFile(key string) (multipart.File, *multipart.FileHeader, error) {
	if p.File != nil {
		if fh := p.File[key]; len(fh) > 0 {
			f, err := fh[0].Open()
			return f, fh[0], err
		}
		return nil, nil, fmt.Errorf("error file is missing: %s", key)
	}
	return nil, nil, nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

// ClientIP returns IP address from HTTP request, typically known as Client IP or
// Remote IP. It parses the IP in the order of X-Forwarded-For, X-Real-IP
// and finally `http.Request.RemoteAddr`.
func ClientIP(req *http.Request) string {
	// Header X-Forwarded-For
	if fwdFor := req.Header.Get(HeaderXForwardedFor); !ess.IsStrEmpty(fwdFor) {
		index := strings.Index(fwdFor, ",")
		if index == -1 {
			return strings.TrimSpace(fwdFor)
		}
		return strings.TrimSpace(fwdFor[:index])
	}

	// Header X-Real-Ip
	if realIP := req.Header.Get(HeaderXRealIP); !ess.IsStrEmpty(realIP) {
		return strings.TrimSpace(realIP)
	}

	// Remote Address
	if remoteAddr, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		return strings.TrimSpace(remoteAddr)
	}

	return ""
}

func getReferer(hdr http.Header) string {
	referer := hdr.Get(HeaderReferer)

	if ess.IsStrEmpty(referer) {
		referer = hdr.Get("Referrer")
	}

	return referer
}

func isJSONPReqeust(r *http.Request) bool {
	query := r.URL.Query()
	callback := query.Get(jsonpReqParamKey)
	return !ess.IsStrEmpty(callback)
}
