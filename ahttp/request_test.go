// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ahttp

import (
	"net/http"
	"testing"

	"github.com/go-aah/test/assert"
)

func TestClientIP(t *testing.T) {
	req1 := createRequest(HeaderXForwardedFor, "10.0.0.1, 10.0.0.2")
	ipAddress := req1.ClientIP(true)
	assert.Equal(t, "10.0.0.1", ipAddress)

	req2 := createRequest(HeaderXForwardedFor, "10.0.0.2")
	ipAddress = req2.ClientIP(true)
	assert.Equal(t, "10.0.0.2", ipAddress)

	req3 := createRequest(HeaderXRealIP, "10.0.0.3")
	ipAddress = req3.ClientIP(true)
	assert.Equal(t, "10.0.0.3", ipAddress)

	req4 := createRequestWithHost("127.0.0.1:8080", "192.168.0.1:1234")
	ipAddress = req4.ClientIP(false)
	assert.Equal(t, "192.168.0.1", ipAddress)

	req5 := createRequestWithHost("127.0.0.1:8080", "")
	ipAddress = req5.ClientIP(false)
	assert.Equal(t, "", ipAddress)
}

func createRequestWithHost(host, remote string) *Request {
	return &Request{
		Request: &http.Request{Host: host, RemoteAddr: remote},
	}
}
