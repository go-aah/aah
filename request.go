// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ahttp

import (
	"net"
	"net/http"
	"strings"

	"aahframework.org/essentials"
)

// Request is extends `http.Request` for aah framework
type Request struct {
	*http.Request
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Request methods
//___________________________________

// ClientIP returns IP address from HTTP request, typically known as Client IP or
// Remote IP. It parses the IP in the order of X-Forwarded-For, X-Real-IP
// and finally `http.Request.RemoteAddr`.
//
// Set aah application configuration `http.proxy = true`, if you're running
// application behind proxy server like nginx, haproxy, apache, etc. otherwise
// this method might return inaccurate Client IP address.
func (r *Request) ClientIP(proxy bool) string {
	if proxy {
		// Header X-Forwarded-For
		if fwdFor := r.Header.Get(HeaderXForwardedFor); !ess.IsStrEmpty(fwdFor) {
			index := strings.Index(fwdFor, ",")
			if index == -1 {
				return strings.TrimSpace(fwdFor)
			}
			return strings.TrimSpace(fwdFor[:index])
		}

		// Header X-Real-Ip
		if realIP := r.Header.Get(HeaderXRealIP); !ess.IsStrEmpty(realIP) {
			return strings.TrimSpace(realIP)
		}
	}

	if remoteAddr, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return strings.TrimSpace(remoteAddr)
	}

	return ""
}
