// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org/ws source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ws

import (
	"fmt"
	"net/http"
	"net/url"

	"aahframework.org/ahttp.v0"
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Request struct and its methods
//______________________________________________________________________________

// Request struct holds information for successful WebSocket connection made.
type Request struct {
	// ID aah assigns Globally Unique Identifier (GUID) using Mongo Object ID
	// algorithm for every WebSocket connection made to aah server.
	//
	// You may use it for tracking, tracing or identifying WebSocket client.
	ID string

	// Host value of the HTTP 'Host' header (e.g. 'example.com:8080').
	Host string

	// Path the request URL Path e.g. `/chatroom/aahframework`.
	Path string

	// Header holds the values of HTTP headers when WebSocket connection made.
	Header http.Header

	pathParams  ahttp.PathParams
	queryParams url.Values
	raw         *http.Request
}

// PathValue method returns value for given Path param key otherwise empty string.
// For eg.: `/discussion/:roomName` => `PathValue("roomName")`.
func (r *Request) PathValue(key string) string {
	if r.pathParams != nil {
		return r.pathParams.Get(key)
	}
	return ""
}

// QueryValue method returns value for given URL query param key
// otherwise empty string.
func (r *Request) QueryValue(key string) string {
	return r.queryParams.Get(key)
}

// QueryArrayValue method returns array value for given URL query param key
// otherwise empty string slice.
func (r *Request) QueryArrayValue(key string) []string {
	if values, found := r.queryParams[key]; found {
		return values
	}
	return []string{}
}

// ClientIP method returns remote Client IP address aka Remote IP.
// It parses in the order of given set of headers otherwise it uses default
// default header set `X-Forwarded-For`, `X-Real-IP`, "X-Appengine-Remote-Addr"
// and finally `http.Request.RemoteAddr`.
func (r *Request) ClientIP() string {
	return ahttp.ClientIP(r.raw)
}

// String request stringer interface.
func (r Request) String() string {
	return fmt.Sprintf("ReqID: %s, Host: %s, Path: %s, Query String: %s",
		r.ID,
		r.Host,
		r.Path,
		r.queryParams.Encode(),
	)
}
