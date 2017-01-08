// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ahttp

import (
	"net"
	"net/http"
	"net/url"
	"strings"

	"aahframework.org/essentials"
)

// Request is extends `http.Request` for aah framework
type Request struct {
	// Host value of the HTTP 'Host' header (e.g. 'example.com:8080').
	Host string

	// Method request method e.g. GET, POST, etc.
	Method string

	// Path the request URL Path e.g. /booking/hotel.html.
	Path string

	// Header request HTTP headers
	Header http.Header

	// ContentType the parsed HTTP header 'Content-Type'.
	ContentType *ContentType

	// AcceptContentType negotiated value from HTTP Header `Accept`.
	// The resolve order is- 1) URL extension 2) Accept header. Most quailfied
	// one based quality factor otherwise default is HTML.
	AcceptContentType *ContentType

	// AcceptEncoding negotiated value from HTTP Header the `Accept-Encoding`
	// Most quailfied one based on quality factor.
	AcceptEncoding AcceptSpec

	// Params contains values from Query string and request body. POST, PUT
	// and PATCH body parameters take precedence over URL query string values.
	Params url.Values

	// PathParams contains values of request path parameters. e.g. /hello/{name}.
	PathParams url.Values

	// Referer value of the HTTP 'Referrer' (or 'Referer') header.
	Referer string

	// ClientIP remote client IP address.
	ClientIP string

	// Locale negotiated value from HTTP Header `Accept-Language`.
	Locale *Locale

	// Raw an object that Go HTTP server provied, Direct interaction with
	// raw object is not encouraged.
	Raw *http.Request
}

// ParseRequest method creates aah framework `ahttp.Request` instance from go
// HTTP request.
func ParseRequest(r *http.Request) *Request {
	req := &Request{
		Host:              r.Host,
		Method:            r.Method,
		Path:              r.URL.Path,
		Header:            r.Header,
		ContentType:       ParseContentType(r),
		AcceptContentType: NegotiateContentType(r),
		Params:            url.Values{},
		PathParams:        url.Values{},
		Referer:           getReferer(r.Header),
		ClientIP:          ClientIP(r),
		Locale:            NegotiateLocale(r),
		Raw:               r,
	}

	return req
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
