// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// aahframework.org/ahttp source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ahttp

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	"aahframework.org/essentials.v0"
)

const (
	jsonpReqParamKey = "callback"
	ajaxHeaderValue  = "XMLHttpRequest"
)

var requestPool = &sync.Pool{New: func() interface{} { return &Request{} }}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package methods
//___________________________________

// ParseRequest method populates the given aah framework `ahttp.Request`
// instance from Go HTTP request.
func ParseRequest(r *http.Request, req *Request) *Request {
	req.Scheme = Scheme(r)
	req.Host = Host(r)
	req.Proto = r.Proto
	req.Method = r.Method
	req.Path = r.URL.Path
	req.Header = r.Header
	req.Referer = getReferer(r.Header)
	req.UserAgent = r.Header.Get(HeaderUserAgent)
	req.IsGzipAccepted = strings.Contains(r.Header.Get(HeaderAcceptEncoding), "gzip")
	req.raw = r
	return req
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Request
//___________________________________

// Request type extends `http.Request` and provides multiple helper methods
// per industry RFC guideline for aah framework.
type Request struct {
	// Scheme value is protocol, refer to method `ahttp.Scheme`.
	Scheme string

	// Host value is HTTP 'Host' header (e.g. 'example.com:8080').
	Host string

	// Proto value is current HTTP request protocol. (e.g. HTTP/1.1, HTTP/2.0)
	Proto string

	// Method value is HTTP verb from request e.g. `GET`, `POST`, etc.
	Method string

	// Path value is request relative URL Path e.g. `/app/login.html`.
	Path string

	// Header is request HTTP headers
	Header http.Header

	// PathParams value is URL path parameters.
	PathParams PathParams

	// Referer value is HTTP 'Referrer' (or 'Referer') header.
	Referer string

	// UserAgent value is HTTP 'User-Agent' header.
	UserAgent string

	// IsGzipAccepted is true if the HTTP client accepts Gzip response,
	// otherwise false.
	IsGzipAccepted bool

	raw               *http.Request
	locale            *Locale
	contentType       *ContentType
	acceptContentType *ContentType
	acceptEncoding    *AcceptSpec
}

// AcceptContentType method returns negotiated value.
//
// The resolve order is-
//
// 1) URL extension
//
// 2) Accept header (As per RFC7231 and vendor type as per RFC4288)
//
// Most quailfied one based on quality factor otherwise default is Plain text.
func (r *Request) AcceptContentType() *ContentType {
	if r.acceptContentType == nil {
		r.acceptContentType = NegotiateContentType(r.Unwrap())
	}
	return r.acceptContentType
}

// SetAcceptContentType method is used to set Accept ContentType instance.
func (r *Request) SetAcceptContentType(contentType *ContentType) *Request {
	r.acceptContentType = contentType
	return r
}

// AcceptEncoding method returns negotiated value from HTTP Header the `Accept-Encoding`
// As per RFC7231.
//
// Most quailfied one based on quality factor.
func (r *Request) AcceptEncoding() *AcceptSpec {
	if r.acceptEncoding == nil {
		if specs := ParseAcceptEncoding(r.Unwrap()); specs != nil {
			r.acceptEncoding = specs.MostQualified()
		}
	}
	return r.acceptEncoding
}

// SetAcceptEncoding method is used to accept encoding spec instance.
func (r *Request) SetAcceptEncoding(encoding *AcceptSpec) *Request {
	r.acceptEncoding = encoding
	return r
}

// ClientIP method returns remote client IP address aka Remote IP.
//
// Refer to method `ahttp.ClientIP`.
func (r *Request) ClientIP() string {
	return ClientIP(r.Unwrap())
}

// Cookie method returns a named cookie from HTTP request otherwise error.
func (r *Request) Cookie(name string) (*http.Cookie, error) {
	return r.Unwrap().Cookie(name)
}

// Cookies method returns all the cookies from HTTP request.
func (r *Request) Cookies() []*http.Cookie {
	return r.Unwrap().Cookies()
}

// ContentType method returns the parsed value of HTTP header `Content-Type` per RFC1521.
func (r *Request) ContentType() *ContentType {
	if r.contentType == nil {
		r.contentType = ParseContentType(r.Unwrap())
	}
	return r.contentType
}

// SetContentType method is used to set ContentType instance.
func (r *Request) SetContentType(contType *ContentType) *Request {
	r.contentType = contType
	return r
}

// Locale method returns negotiated value from HTTP Header `Accept-Language`
// per RFC7231.
func (r *Request) Locale() *Locale {
	if r.locale == nil {
		r.locale = NegotiateLocale(r.Unwrap())
	}
	return r.locale
}

// SetLocale method is used to set locale instance in to aah request.
func (r *Request) SetLocale(locale *Locale) *Request {
	r.locale = locale
	return r
}

// IsJSONP method returns true if request URL query string has "callback=function_name".
// otherwise false.
func (r *Request) IsJSONP() bool {
	return !ess.IsStrEmpty(r.QueryValue(jsonpReqParamKey))
}

// IsAJAX method returns true if request header `X-Requested-With` is
// `XMLHttpRequest` otherwise false.
func (r *Request) IsAJAX() bool {
	return r.Header.Get(HeaderXRequestedWith) == ajaxHeaderValue
}

// URL method return underlying request URL instance.
func (r *Request) URL() *url.URL {
	return r.Unwrap().URL
}

// PathValue method returns value for given Path param key otherwise empty string.
// For eg.: /users/:userId => PathValue("userId")
func (r *Request) PathValue(key string) string {
	return r.PathParams.Get(key)
}

// QueryValue method returns value for given URL query param key
// otherwise empty string.
func (r *Request) QueryValue(key string) string {
	return r.URL().Query().Get(key)
}

// QueryArrayValue method returns array value for given URL query param key
// otherwise empty string slice.
func (r *Request) QueryArrayValue(key string) []string {
	if values, found := r.URL().Query()[key]; found {
		return values
	}
	return []string{}
}

// FormValue method returns value for given form key otherwise empty string.
func (r *Request) FormValue(key string) string {
	return r.Unwrap().FormValue(key)
}

// FormArrayValue method returns array value for given form key
// otherwise empty string slice.
func (r *Request) FormArrayValue(key string) []string {
	if r.Unwrap().Form != nil {
		if values, found := r.Unwrap().Form[key]; found {
			return values
		}
	}
	return []string{}
}

// FormFile method returns the first file for the provided form key otherwise
// returns error. It is caller responsibility to close the file.
func (r *Request) FormFile(key string) (multipart.File, *multipart.FileHeader, error) {
	return r.Unwrap().FormFile(key)
}

// Body method returns the HTTP request body.
func (r *Request) Body() io.ReadCloser {
	return r.Unwrap().Body
}

// Unwrap method returns the underlying *http.Request instance of Go HTTP server,
// direct interaction with raw object is not encouraged. Use it appropriately.
func (r *Request) Unwrap() *http.Request {
	return r.raw
}

// SaveFile method saves an uploaded multipart file for given key from the HTTP
// request into given destination
func (r *Request) SaveFile(key, dstFile string) (int64, error) {
	if ess.IsStrEmpty(dstFile) || ess.IsStrEmpty(key) {
		return 0, errors.New("ahttp: key or dstFile is empty")
	}

	if ess.IsDir(dstFile) {
		return 0, errors.New("ahttp: dstFile should not be a directory")
	}

	uploadedFile, _, err := r.FormFile(key)
	if err != nil {
		return 0, err
	}
	defer ess.CloseQuietly(uploadedFile)

	return saveFile(uploadedFile, dstFile)
}

// Reset method resets request instance for reuse.
func (r *Request) Reset() {
	r.Scheme = ""
	r.Host = ""
	r.Proto = ""
	r.Method = ""
	r.Path = ""
	r.Header = nil
	r.PathParams = nil
	r.Referer = ""
	r.UserAgent = ""
	r.IsGzipAccepted = false

	r.raw = nil
	r.locale = nil
	r.contentType = nil
	r.acceptContentType = nil
	r.acceptEncoding = nil
}

func (r *Request) cleanupMutlipart() {
	if r.Unwrap().MultipartForm != nil {
		r.Unwrap().MultipartForm.RemoveAll()
	}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// PathParams
//___________________________________

// PathParams struct holds the path parameter key and values.
type PathParams map[string]string

// Get method returns the value for the given key otherwise empty string.
func (p PathParams) Get(key string) string {
	if value, found := p[key]; found {
		return value
	}
	return ""
}

// Len method returns count of total no. of values.
func (p PathParams) Len() int {
	return len(p)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

func getReferer(hdr http.Header) string {
	referer := hdr.Get(HeaderReferer)
	if len(referer) > 0 {
		return referer
	}
	return hdr.Get("Referrer")
}

func saveFile(r io.Reader, destFile string) (int64, error) {
	f, err := os.OpenFile(destFile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		return 0, fmt.Errorf("ahttp: %s", err)
	}
	defer ess.CloseQuietly(f)

	return io.Copy(f, r)
}
