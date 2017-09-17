// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package anticsrf

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"aahframework.org/ahttp.v0"
	"aahframework.org/essentials.v0-unstable"
)

var (
	// As defined in https://tools.ietf.org/html/rfc7231#section-4.2.1
	safeHTTPMethods = []string{ahttp.MethodGet, ahttp.MethodHead, ahttp.MethodOptions, ahttp.MethodTrace}
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package methods
//___________________________________

// IsSafeHTTPMethod method returns true if matches otherwise false.
// Safe methods per defined in https://tools.ietf.org/html/rfc7231#section-4.2.1
func IsSafeHTTPMethod(method string) bool {
	return ess.IsSliceContainsString(safeHTTPMethods, method)
}

// IsSameOrigin method is to check same origin i.e. scheme, host and port.
// Returns true if matches otherwise false.
func IsSameOrigin(a, b *url.URL) bool {
	return (a.Scheme == b.Scheme && a.Host == b.Host)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

// Reference: https://github.com/golang/go/blob/master/src/crypto/cipher/xor.go#L45-L54
func xorBytes(a, b []byte) []byte {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	dst := make([]byte, n)
	for i := 0; i < n; i++ {
		dst[i] = a[i] ^ b[i]
	}
	return dst
}

// toSeconds method converts string value into seconds.
func toSeconds(value string) (int64, error) {
	if strings.HasSuffix(value, "m") || strings.HasSuffix(value, "h") {
		d, err := time.ParseDuration(value)
		if err != nil {
			return 0, err
		}
		return int64(d.Seconds()), nil
	}
	return 0, fmt.Errorf("unsupported time unit '%s' on 'security.anti_csrf.ttl'", value)
}
