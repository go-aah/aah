// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/ahttp source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ahttp

import (
	"net/http"
	"net/url"
	"testing"

	"aahframework.org/test.v0-unstable/assert"
)

func TestClientIP(t *testing.T) {
	req1 := createRawHTTPRequest(HeaderXForwardedFor, "10.0.0.1, 10.0.0.2")
	ipAddress := ClientIP(req1)
	assert.Equal(t, "10.0.0.1", ipAddress)

	req2 := createRawHTTPRequest(HeaderXForwardedFor, "10.0.0.2")
	ipAddress = ClientIP(req2)
	assert.Equal(t, "10.0.0.2", ipAddress)

	req3 := createRawHTTPRequest(HeaderXRealIP, "10.0.0.3")
	ipAddress = ClientIP(req3)
	assert.Equal(t, "10.0.0.3", ipAddress)

	req4 := createRequestWithHost("127.0.0.1:8080", "192.168.0.1:1234")
	ipAddress = ClientIP(req4)
	assert.Equal(t, "192.168.0.1", ipAddress)

	req5 := createRequestWithHost("127.0.0.1:8080", "")
	ipAddress = ClientIP(req5)
	assert.Equal(t, "", ipAddress)
}

func TestGetReferer(t *testing.T) {
	req1 := createRawHTTPRequest(HeaderReferer, "http://localhost:8080/welcome1.html")
	referer := getReferer(req1.Header)
	assert.Equal(t, "http://localhost:8080/welcome1.html", referer)

	req2 := createRawHTTPRequest("Referrer", "http://localhost:8080/welcome2.html")
	referer = getReferer(req2.Header)
	assert.Equal(t, "http://localhost:8080/welcome2.html", referer)
}

func TestParseRequest(t *testing.T) {
	req := createRequestWithHost("127.0.0.1:8080", "192.168.0.1:1234")
	req.Header = http.Header{}
	req.Method = "GET"
	req.Header.Add(HeaderMethod, "GET")
	req.Header.Add(HeaderContentType, "application/json;charset=utf-8")
	req.Header.Add(HeaderAccept, "application/json;charset=utf-8")
	req.Header.Add(HeaderReferer, "http://localhost:8080/home.html")
	req.Header.Add(HeaderAcceptLanguage, "en-gb;leve=1;q=0.8, da, en;level=2;q=0.7, en-us;q=gg")
	req.URL, _ = url.Parse("http://localhost:8080/welcome1.html?_ref=true")

	aahReq := ParseRequest(req, &Request{})

	assert.Equal(t, "127.0.0.1:8080", aahReq.Host)
	assert.Equal(t, "GET", aahReq.Method)
	assert.Equal(t, "/welcome1.html", aahReq.Path)
	assert.Equal(t, "en-gb;leve=1;q=0.8, da, en;level=2;q=0.7, en-us;q=gg", aahReq.Header.Get(HeaderAcceptLanguage))
	assert.Equal(t, "application/json; charset=utf-8", aahReq.ContentType.String())
	assert.Equal(t, "192.168.0.1", aahReq.ClientIP)
	assert.Equal(t, "http://localhost:8080/home.html", aahReq.Referer)
}

func createRequestWithHost(host, remote string) *http.Request {
	return &http.Request{Host: host, RemoteAddr: remote}
}
