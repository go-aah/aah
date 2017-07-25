// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/ahttp source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ahttp

import (
	"crypto/tls"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"aahframework.org/test.v0/assert"
)

func TestHTTPClientIP(t *testing.T) {
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

func TestHTTPGetReferer(t *testing.T) {
	req1 := createRawHTTPRequest(HeaderReferer, "http://localhost:8080/welcome1.html")
	referer := getReferer(req1.Header)
	assert.Equal(t, "http://localhost:8080/welcome1.html", referer)

	req2 := createRawHTTPRequest("Referrer", "http://localhost:8080/welcome2.html")
	referer = getReferer(req2.Header)
	assert.Equal(t, "http://localhost:8080/welcome2.html", referer)
}

func TestHTTPParseRequest(t *testing.T) {
	req := createRequestWithHost("127.0.0.1:8080", "192.168.0.1:1234")
	req.Method = MethodGet
	req.Header.Add(HeaderMethod, MethodGet)
	req.Header.Add(HeaderContentType, "application/json;charset=utf-8")
	req.Header.Add(HeaderAccept, "application/json;charset=utf-8")
	req.Header.Add(HeaderReferer, "http://localhost:8080/home.html")
	req.Header.Add(HeaderAcceptLanguage, "en-gb;leve=1;q=0.8, da, en;level=2;q=0.7, en-us;q=gg")
	req.URL, _ = url.Parse("/welcome1.html?_ref=true")

	aahReq := AcquireRequest(req)

	assert.Equal(t, req, aahReq.Unwrap())
	assert.Equal(t, "127.0.0.1:8080", aahReq.Host)
	assert.Equal(t, MethodGet, aahReq.Method)
	assert.Equal(t, "/welcome1.html", aahReq.Path)
	assert.Equal(t, "en-gb;leve=1;q=0.8, da, en;level=2;q=0.7, en-us;q=gg", aahReq.Header.Get(HeaderAcceptLanguage))
	assert.Equal(t, "application/json; charset=utf-8", aahReq.ContentType.String())
	assert.Equal(t, "192.168.0.1", aahReq.ClientIP)
	assert.Equal(t, "http://localhost:8080/home.html", aahReq.Referer)

	// Query Value
	assert.Equal(t, "true", aahReq.QueryValue("_ref"))
	assert.Equal(t, 1, len(aahReq.QueryArrayValue("_ref")))

	// Path value
	assert.Equal(t, "", aahReq.PathValue("not_exists"))

	// Form value
	assert.Equal(t, "", aahReq.FormValue("no_field"))
	assert.Equal(t, 0, len(aahReq.FormArrayValue("no_field")))

	// Form File
	f, hdr, err := aahReq.FormFile("no_file")
	assert.Nil(t, f)
	assert.Nil(t, hdr)
	assert.Nil(t, err)
	assert.False(t, aahReq.IsJSONP())
	assert.False(t, aahReq.IsAJAX())

	// Reset it
	aahReq.Reset()
	assert.Nil(t, aahReq.Header)
	assert.Nil(t, aahReq.ContentType)
	assert.Nil(t, aahReq.AcceptContentType)
	assert.Nil(t, aahReq.Params)
	assert.Nil(t, aahReq.Locale)
	assert.Nil(t, aahReq.Raw)
	assert.True(t, len(aahReq.UserAgent) == 0)
	assert.True(t, len(aahReq.ClientIP) == 0)
	ReleaseRequest(aahReq)
}

func TestHTTPRequestParams(t *testing.T) {
	// Query & Path Value
	req1 := createRequestWithHost("127.0.0.1:8080", "192.168.0.1:1234")
	req1.Method = MethodPost
	req1.URL, _ = url.Parse("http://localhost:8080/welcome1.html?_ref=true&names=Test1&names=Test%202")

	params1 := AcquireRequest(req1).Params
	params1.Path = make(map[string]string)
	params1.Path["userId"] = "100001"
	assert.Equal(t, "true", params1.QueryValue("_ref"))
	assert.Equal(t, "Test1", params1.QueryArrayValue("names")[0])
	assert.Equal(t, "Test 2", params1.QueryArrayValue("names")[1])
	assert.True(t, len(params1.QueryArrayValue("not-exists")) == 0)
	assert.Equal(t, "100001", params1.PathValue("userId"))
	assert.Equal(t, "", params1.PathValue("accountId"))

	// Form value
	form := url.Values{}
	form.Add("names", "Test1")
	form.Add("names", "Test 2 value")
	form.Add("username", "welcome")
	form.Add("email", "welcome@welcome.com")
	req2, _ := http.NewRequest("POST", "http://localhost:8080/user/registration", strings.NewReader(form.Encode()))
	req2.Header.Add(HeaderContentType, ContentTypeForm.String())
	_ = req2.ParseForm()

	aahReq2 := AcquireRequest(req2)
	aahReq2.Params.Form = req2.Form

	params2 := aahReq2.Params
	assert.Equal(t, "welcome", params2.FormValue("username"))
	assert.Equal(t, "welcome@welcome.com", params2.FormValue("email"))
	assert.Equal(t, "Test1", params2.FormArrayValue("names")[0])
	assert.Equal(t, "Test 2 value", params2.FormArrayValue("names")[1])
	assert.True(t, len(params2.FormArrayValue("not-exists")) == 0)
	ReleaseRequest(aahReq2)

	// File value
	req3, _ := http.NewRequest("POST", "http://localhost:8080/user/registration", nil)
	req3.Header.Add(HeaderContentType, ContentTypeMultipartForm.String())
	aahReq3 := AcquireRequest(req3)
	aahReq3.Params.File = make(map[string][]*multipart.FileHeader)
	aahReq3.Params.File["testfile.txt"] = []*multipart.FileHeader{{Filename: "testfile.txt"}}
	f, fh, err := aahReq3.FormFile("testfile.txt")
	assert.Nil(t, f)
	assert.Equal(t, "testfile.txt", fh.Filename)
	assert.Equal(t, "open : no such file or directory", err.Error())
	ReleaseRequest(aahReq3)
}

func TestHTTPRequestCookies(t *testing.T) {
	req := createRequestWithHost("127.0.0.1:8080", "192.168.0.1:1234")
	req.Method = MethodGet
	req.URL, _ = url.Parse("http://localhost:8080/welcome1.html?_ref=true")
	req.AddCookie(&http.Cookie{
		Name:  "test-1",
		Value: "test-1 value",
	})
	req.AddCookie(&http.Cookie{
		Name:  "test-2",
		Value: "test-2 value",
	})

	aahReq := ParseRequest(req, &Request{})
	assert.NotNil(t, aahReq)
	assert.True(t, len(aahReq.Cookies()) == 2)

	cookie, _ := aahReq.Cookie("test-2")
	assert.Equal(t, "test-2 value", cookie.Value)
}

func TestRequestSchemeDerived(t *testing.T) {
	req := httptest.NewRequest("GET", "http://127.0.0.1:8080/welcome.html", nil)
	scheme1 := identifyScheme(req)
	assert.Equal(t, "http", scheme1)

	req.TLS = &tls.ConnectionState{}
	scheme2 := identifyScheme(req)
	assert.Equal(t, "https", scheme2)

	req.Header.Set(HeaderXForwardedProto, "https")
	scheme3 := identifyScheme(req)
	assert.Equal(t, "https", scheme3)

	req.Header.Set(HeaderXForwardedProto, "http")
	scheme4 := identifyScheme(req)
	assert.Equal(t, "http", scheme4)
}

func createRequestWithHost(host, remote string) *http.Request {
	return &http.Request{Host: host, RemoteAddr: remote, Header: http.Header{}}
}
