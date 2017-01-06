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
// Unexported methods
//___________________________________

// clientIP returns IP address from HTTP request, typically known as Client IP or
// Remote IP. It parses the IP in the order of X-Forwarded-For, X-Real-IP
// and finally `http.Request.RemoteAddr`.
func clientIP(req *http.Request) string {
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
