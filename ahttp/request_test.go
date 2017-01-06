// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ahttp

import (
	"net/http"
	"testing"

	"aahframework.org/test/assert"
)

func TestClientIP(t *testing.T) {
	req1 := createRawHTTPRequest(HeaderXForwardedFor, "10.0.0.1, 10.0.0.2")
	ipAddress := clientIP(req1)
	assert.Equal(t, "10.0.0.1", ipAddress)

	req2 := createRawHTTPRequest(HeaderXForwardedFor, "10.0.0.2")
	ipAddress = clientIP(req2)
	assert.Equal(t, "10.0.0.2", ipAddress)

	req3 := createRawHTTPRequest(HeaderXRealIP, "10.0.0.3")
	ipAddress = clientIP(req3)
	assert.Equal(t, "10.0.0.3", ipAddress)

	req4 := createRequestWithHost("127.0.0.1:8080", "192.168.0.1:1234")
	ipAddress = clientIP(req4)
	assert.Equal(t, "192.168.0.1", ipAddress)

	req5 := createRequestWithHost("127.0.0.1:8080", "")
	ipAddress = clientIP(req5)
	assert.Equal(t, "", ipAddress)
}

func createRequestWithHost(host, remote string) *http.Request {
	return &http.Request{Host: host, RemoteAddr: remote}
}
