// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/ahttp source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ahttp

import (
	"bytes"
	"crypto/tls"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"aahframework.org/essentials.v0"
	"aahframework.org/test.v0/assert"
)

func TestHTTPClientIP(t *testing.T) {
	req1 := createRawHTTPRequest(HeaderXForwardedFor, "10.0.0.1, 10.0.0.2")
	ipAddress := AcquireRequest(req1).ClientIP()
	assert.Equal(t, "10.0.0.1", ipAddress)

	req2 := createRawHTTPRequest(HeaderXForwardedFor, "10.0.0.2")
	ipAddress = AcquireRequest(req2).ClientIP()
	assert.Equal(t, "10.0.0.2", ipAddress)

	req3 := createRawHTTPRequest(HeaderXRealIP, "10.0.0.3")
	ipAddress = AcquireRequest(req3).ClientIP()
	assert.Equal(t, "10.0.0.3", ipAddress)

	req4 := createRequestWithHost("127.0.0.1:8080", "192.168.0.1:1234")
	ipAddress = AcquireRequest(req4).ClientIP()
	assert.Equal(t, "192.168.0.1", ipAddress)

	req5 := createRequestWithHost("127.0.0.1:8080", "")
	ipAddress = AcquireRequest(req5).ClientIP()
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
	assert.True(t, req.URL == aahReq.URL())

	assert.Equal(t, req, aahReq.Unwrap())
	assert.Equal(t, "127.0.0.1:8080", aahReq.Host)
	assert.Equal(t, MethodGet, aahReq.Method)
	assert.Equal(t, "/welcome1.html", aahReq.Path)
	assert.Equal(t, "en-gb;leve=1;q=0.8, da, en;level=2;q=0.7, en-us;q=gg", aahReq.Header.Get(HeaderAcceptLanguage))
	assert.Equal(t, "application/json; charset=utf-8", aahReq.ContentType().String())
	assert.Equal(t, "192.168.0.1", aahReq.ClientIP())
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

	aahReq.SetAcceptContentType(nil)
	assert.NotNil(t, aahReq.AcceptContentType())
	aahReq.SetLocale(nil)
	assert.NotNil(t, aahReq.Locale())
	aahReq.SetContentType(nil)
	assert.NotNil(t, aahReq.ContentType())
	aahReq.SetAcceptEncoding(nil)
	assert.Nil(t, aahReq.AcceptEncoding())

	// Release it
	ReleaseRequest(aahReq)
	assert.Nil(t, aahReq.Header)
	assert.Nil(t, aahReq.Params)
	assert.Nil(t, aahReq.Raw)
	assert.True(t, aahReq.UserAgent == "")
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
	assert.Equal(t, 1, params1.Path.Len())

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
	assert.NotNil(t, aahReq2.Body())
	assert.Equal(t, "welcome", params2.FormValue("username"))
	assert.Equal(t, "welcome@welcome.com", params2.FormValue("email"))
	assert.Equal(t, "Test1", params2.FormArrayValue("names")[0])
	assert.Equal(t, "Test 2 value", params2.FormArrayValue("names")[1])
	assert.True(t, len(params2.FormArrayValue("not-exists")) == 0)
	assert.Equal(t, 0, params2.Path.Len())
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
	scheme1 := IdentifyScheme(req)
	assert.Equal(t, "http", scheme1)

	req.TLS = &tls.ConnectionState{}
	scheme2 := IdentifyScheme(req)
	assert.Equal(t, "https", scheme2)

	req.Header.Set(HeaderXForwardedProto, "https")
	scheme3 := IdentifyScheme(req)
	assert.Equal(t, "https", scheme3)

	req.Header.Set(HeaderXForwardedProto, "http")
	scheme4 := IdentifyScheme(req)
	assert.Equal(t, "http", scheme4)
}

func TestRequestSaveFile(t *testing.T) {
	aahReq, path, teardown := setUpRequestSaveFile(t)
	defer teardown()

	size, err := aahReq.SaveFile("framework", path)
	assert.Nil(t, err)
	assert.Equal(t, int64(0), size)
	_, err = os.Stat(path)
	assert.Nil(t, err)
}

func TestRequestSaveFileFailsValidation(t *testing.T) {
	aahReq, path, teardown := setUpRequestSaveFile(t)
	defer teardown()

	// Empty keys should error out
	_, err := aahReq.SaveFile("", path)
	assert.NotNil(t, err)
	assert.Equal(t, "ahttp: key or dstFile is empty", err.Error())

	// Empty path should error out
	_, err = aahReq.SaveFile("framework", "")
	assert.NotNil(t, err)
	assert.Equal(t, "ahttp: key or dstFile is empty", err.Error())

	// If "path" is a directory, it should error out
	_, err = aahReq.SaveFile("framework", "testdata")
	assert.NotNil(t, err)
	assert.Equal(t, "ahttp: dstFile should not be a directory", err.Error())
}

func TestRequestSaveFileFailsForNotFoundFile(t *testing.T) {
	aahReq, path, teardown := setUpRequestSaveFile(t)
	defer teardown()

	_, err := aahReq.SaveFile("unknown-key", path)
	assert.NotNil(t, err)
	assert.Equal(t, "ahttp: no such key/file: unknown-key", err.Error())
}

func TestRequestSaveFileCannotCreateFile(t *testing.T) {
	aahReq, _, teardown := setUpRequestSaveFile(t)
	defer teardown()

	_, err := aahReq.SaveFile("framework", "/root/aah.txt")
	assert.NotNil(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "ahttp: open /root/aah.txt"))
}

func TestRequestSaveFiles(t *testing.T) {
	aahReq, dir, teardown := setUpRequestSaveFiles(t)
	defer teardown()

	sizes, errs := aahReq.SaveFiles("framework", dir)
	assert.Nil(t, errs)
	assert.Nil(t, sizes)
	_, err := os.Stat(dir + "/aah")
	assert.Nil(t, err)
	_, err = os.Stat(dir + "/aah2")
	assert.Nil(t, err)
}

func TestRequestSaveFilesFailsVaildation(t *testing.T) {
	aahReq, dir, teardown := setUpRequestSaveFiles(t)
	defer teardown()

	// Empty key
	sizes, errs := aahReq.SaveFiles("", dir)
	assert.NotNil(t, errs)
	assert.Equal(t, "ahttp: form file key, '' is empty", errs[0].Error())
	assert.Equal(t, int64(0), sizes[0])

	// Empty directory
	sizes, errs = aahReq.SaveFiles("key", "")
	assert.NotNil(t, errs)
	assert.Equal(t, "ahttp: destination path, '' is not a directory", errs[0].Error())
	assert.Equal(t, int64(0), sizes[0])
}

func TestRequestSaveFilesCannotCreateFile(t *testing.T) {
	aahReq, _, teardown := setUpRequestSaveFiles(t)
	defer teardown()

	sizes, errs := aahReq.SaveFiles("framework", "/root")
	assert.NotNil(t, errs)
	assert.Equal(t, int64(0), sizes[0])

	errMsg := errs[0].Error()
	assert.True(t, ("ahttp: open /root/aah: permission denied" == errMsg ||
		"ahttp: destination path, '/root' is not a directory" == errMsg))
}

func TestRequestSaveFileForExistingFile(t *testing.T) {
	var buf bytes.Buffer

	size, err := saveFile(&buf, "testdata/file1.txt")
	assert.NotNil(t, err)
	assert.Equal(t, "ahttp: open testdata/file1.txt: file exists", err.Error())
	assert.Equal(t, int64(0), size)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// test unexported methods
//___________________________________

func createRequestWithHost(host, remote string) *http.Request {
	url, _ := url.Parse("http://localhost:8080/testpath")
	return &http.Request{URL: url, Host: host, RemoteAddr: remote, Header: http.Header{}}
}

func setUpRequestSaveFile(t *testing.T) (*Request, string, func()) {
	buf := new(bytes.Buffer)
	multipartWriter := multipart.NewWriter(buf)
	_, err := multipartWriter.CreateFormFile("framework", "aah")
	assert.Nil(t, err)

	ess.CloseQuietly(multipartWriter)

	req, _ := http.NewRequest("POST", "http://localhost:8080", buf)
	req.Header.Add(HeaderContentType, multipartWriter.FormDataContentType())
	aahReq := AcquireRequest(req)
	aahReq.Params.File = make(map[string][]*multipart.FileHeader)

	_, header, err := req.FormFile("framework")
	assert.Nil(t, err)

	aahReq.Params.File["framework"] = []*multipart.FileHeader{header}

	path := "testdata/aah.txt"

	return aahReq, path, func() {
		_ = os.Remove(path) //Teardown
	}
}

func setUpRequestSaveFiles(t *testing.T) (*Request, string, func()) {
	buf := new(bytes.Buffer)
	multipartWriter := multipart.NewWriter(buf)
	_, err := multipartWriter.CreateFormFile("framework", "aah")
	assert.Nil(t, err)
	_, err = multipartWriter.CreateFormFile("framework2", "aah2")
	assert.Nil(t, err)

	ess.CloseQuietly(multipartWriter)

	req, _ := http.NewRequest("POST", "http://localhost:8080", buf)
	req.Header.Add(HeaderContentType, multipartWriter.FormDataContentType())
	aahReq := AcquireRequest(req)
	aahReq.Params.File = make(map[string][]*multipart.FileHeader)

	_, header, err := req.FormFile("framework")
	assert.Nil(t, err)
	_, header2, err := req.FormFile("framework2")
	assert.Nil(t, err)

	aahReq.Params.File["framework"] = []*multipart.FileHeader{header, header2}

	dir := "testdata/upload"

	_ = ess.MkDirAll(dir, 0755)
	return aahReq, dir, func() {
		_ = os.RemoveAll(dir)
	}
}
