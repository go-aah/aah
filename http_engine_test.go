// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/test.v0/assert"
)

func TestHTTPEngineTestRequests(t *testing.T) {
	importPath := filepath.Join(testdataBaseDir(), "webapp1")
	ts := newTestServer(t, importPath)
	defer ts.Close()

	t.Logf("Test Server URL [Engine Handling]: %s", ts.URL)

	// declare functions
	testOnRequest := func(e *Event) {
		ctx := e.Data.(*Context)
		ctx.Log().Info("Application OnRequest extension point")
	}

	testOnPreReply := func(e *Event) {
		ctx := e.Data.(*Context)
		ctx.Log().Info("Application OnPreReply extension point")
	}

	testOnHeaderReply := func(e *Event) {
		hdr := e.Data.(http.Header)
		hdr.Add("X-Event-OnHeaderReply", "Application OnHeaderReply extension point")
	}

	testOnPostReply := func(e *Event) {
		ctx := e.Data.(*Context)
		ctx.Log().Info("Application OnPostReply extension point")
	}

	testOnPreAuth := func(e *Event) {
		ctx := e.Data.(*Context)
		ctx.Log().Info("Application OnPreAuth extension point")
	}

	testOnPostAuth := func(e *Event) {
		ctx := e.Data.(*Context)
		ctx.Log().Info("Application OnPostAuth extension point")
	}

	// Adding Server extension points
	he := ts.app.HTTPEngine()
	he.OnRequest(func(e *Event) {
		t.Log("Application OnRequest extension point")
	})
	he.OnRequest(testOnRequest)

	he.OnPreReply(func(e *Event) {
		t.Log("Application OnPreReply extension point")
	})
	he.OnPreReply(testOnPreReply)

	he.OnHeaderReply(func(e *Event) {
		t.Log("Application OnHeaderReply extension point")
	})
	he.OnHeaderReply(testOnHeaderReply)

	he.OnPostReply(func(e *Event) {
		t.Log("Application OnPostReply extension point")
	})
	he.OnPostReply(testOnPostReply)

	he.OnPreAuth(func(e *Event) {
		t.Log("Application OnPreAuth extension point")
	})
	he.OnPreAuth(testOnPreAuth)

	he.OnPostAuth(func(e *Event) {
		t.Log("Application OnPostAuth extension point")
	})
	he.OnPostAuth(testOnPostAuth)

	ts.app.errorMgr.SetHandler(func(ctx *Context, err *Error) bool {
		ctx.Log().Infof("Centrallized error handler called : %s", err)
		t.Logf("Centrallized error handler called : %s", err)
		ctx.Reply().Header("X-Centrallized-ErrorHandler", "true")
		return false
	})

	httpClient := new(http.Client)

	// Panic Flow test HTML - /trigger-panic
	t.Log("Panic Flow test - /trigger-panic")
	resp, err := httpClient.Get(ts.URL + "/trigger-panic")
	assert.Nil(t, err)
	assert.Equal(t, 500, resp.StatusCode)
	assert.Equal(t, "text/html; charset=utf-8", resp.Header.Get(ahttp.HeaderContentType))
	assert.Equal(t, "true", resp.Header.Get("X-Centrallized-ErrorHandler"))
	assert.Equal(t, "true", resp.Header.Get("X-Cntrl-ErrorHandler"))
	assert.True(t, strings.Contains(responseBody(resp), "500 Internal Server Error"))

	// Panic Flow test JSON - /trigger-panic
	t.Log("Panic Flow test JSON - /trigger-panic")
	req, err := http.NewRequest(ahttp.MethodGet, ts.URL+"/trigger-panic", nil)
	assert.Nil(t, err)
	req.Header.Set(ahttp.HeaderAccept, "application/json")
	resp, err = httpClient.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, 500, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get(ahttp.HeaderContentType))
	assert.Equal(t, "true", resp.Header.Get("X-Centrallized-ErrorHandler"))
	assert.Equal(t, "true", resp.Header.Get("X-Cntrl-ErrorHandler"))
	assert.True(t, strings.Contains(responseBody(resp), `"message":"Internal Server Error"`))

	// Panic Flow test XML - /trigger-panic
	t.Log("Panic Flow test XML - /trigger-panic")
	req, err = http.NewRequest(ahttp.MethodGet, ts.URL+"/trigger-panic", nil)
	assert.Nil(t, err)
	req.Header.Set(ahttp.HeaderAccept, "application/xml")
	resp, err = httpClient.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, 500, resp.StatusCode)
	assert.Equal(t, "application/xml; charset=utf-8", resp.Header.Get(ahttp.HeaderContentType))
	assert.Equal(t, "true", resp.Header.Get("X-Centrallized-ErrorHandler"))
	assert.Equal(t, "true", resp.Header.Get("X-Cntrl-ErrorHandler"))
	assert.True(t, strings.Contains(responseBody(resp), `<message>Internal Server Error</message>`))

	// GET XML non-pretty response - /get-xml
	t.Log("GET XML non-pretty response - /get-xml")
	resp, err = httpClient.Get(ts.URL + "/get-xml")
	assert.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "application/xml; charset=utf-8", resp.Header.Get(ahttp.HeaderContentType))
	assert.Equal(t, "120", resp.Header.Get(ahttp.HeaderContentLength))
	assert.True(t, strings.Contains(responseBody(resp), "<Message>This is XML payload result</Message>"))

	// GET JSONP non-pretty response - /get-jsonp?callback=welcome1
	t.Log("GET JSONP non-pretty response - /get-jsonp?callback=welcome1")
	resp, err = httpClient.Get(ts.URL + "/get-jsonp?callback=welcome1")
	assert.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "application/javascript; charset=utf-8", resp.Header.Get(ahttp.HeaderContentType))
	assert.Equal(t, "139", resp.Header.Get(ahttp.HeaderContentLength))
	assert.True(t, strings.HasPrefix(responseBody(resp), `welcome1({"ProductID":190398398,"ProductName":"JSONP product","Username"`))

	// GET JSONP non-pretty response no callback input - /get-jsonp
	t.Log("GET JSONP non-pretty response no callback input - /get-jsonp")
	resp, err = httpClient.Get(ts.URL + "/get-jsonp")
	assert.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "application/javascript; charset=utf-8", resp.Header.Get(ahttp.HeaderContentType))
	assert.Equal(t, "128", resp.Header.Get(ahttp.HeaderContentLength))
	assert.True(t, strings.HasPrefix(responseBody(resp), `{"ProductID":190398398,"ProductName":"JSONP product","Username"`))

	// GET SecureJSON response - /secure-json
	t.Log("GET SecureJSON response - /secure-json")
	resp, err = httpClient.Get(ts.URL + "/secure-json")
	assert.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get(ahttp.HeaderContentType))
	assert.Equal(t, "135", resp.Header.Get(ahttp.HeaderContentLength))
	assert.True(t, strings.HasPrefix(responseBody(resp), `)]}',`))

	// GET Binary bytes - /binary-bytes
	t.Log("GET Binary bytes - /binary-bytes")
	resp, err = httpClient.Get(ts.URL + "/binary-bytes")
	assert.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "text/plain; charset=utf-8", resp.Header.Get(ahttp.HeaderContentType))
	// assert.Equal(t, "23", resp.Header.Get(ahttp.HeaderContentLength))
	assert.True(t, strings.Contains(responseBody(resp), "This is my Binary Bytes"))

	// GET Send File - /send-file
	t.Log("GET Send File - /send-file")
	resp, err = httpClient.Get(ts.URL + "/send-file")
	assert.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "text/css", resp.Header.Get(ahttp.HeaderContentType))
	assert.Equal(t, "inline; filename=aah.css", resp.Header.Get(ahttp.HeaderContentDisposition))
	// assert.Equal(t, "700", resp.Header.Get(ahttp.HeaderContentLength))
	assert.True(t, strings.Contains(responseBody(resp), "Minimal aah framework application template CSS."))

	// GET Hey Cookies - /hey-cookies
	t.Log("GET Send File - /hey-cookies")
	resp, err = httpClient.Get(ts.URL + "/hey-cookies")
	assert.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "text/plain; charset=utf-8", resp.Header.Get(ahttp.HeaderContentType))
	assert.True(t, strings.Contains(responseBody(resp), "Hey I'm sending cookies for you :)"))
	cookieStr := strings.Join(resp.Header["Set-Cookie"], "||")
	assert.NotEqual(t, "", cookieStr)
	assert.True(t, strings.Contains(cookieStr, `test_cookie_1="This is test cookie value 1"`))
	assert.True(t, strings.Contains(cookieStr, `test_cookie_2="This is test cookie value 2"`))

	// OPTIONS request - /get-xml
	t.Log("OPTIONS request - /get-xml")
	req, err = http.NewRequest(ahttp.MethodOptions, ts.URL+"/get-xml", nil)
	assert.Nil(t, err)
	resp, err = httpClient.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "GET, OPTIONS", resp.Header.Get(ahttp.HeaderAllow))
	assert.Equal(t, "0", resp.Header.Get(ahttp.HeaderContentLength))

	// POST - Method Not allowed - /binary-bytes
	t.Log("POST - Method Not allowed - /binary-bytes")
	resp, err = httpClient.Post(ts.URL+"/binary-bytes", ahttp.ContentTypeJSON.String(), strings.NewReader(`{"message":"accept this request"}`))
	assert.Nil(t, err)
	assert.Equal(t, 405, resp.StatusCode)
	assert.Equal(t, "GET, OPTIONS", resp.Header.Get(ahttp.HeaderAllow))
	assert.Equal(t, "text/html; charset=utf-8", resp.Header.Get(ahttp.HeaderContentType))
	assert.True(t, strings.Contains(responseBody(resp), "405 Method Not Allowed"))
}

func TestServerRedirect(t *testing.T) {
	a := newApp()
	a.cfg = config.NewEmpty()

	// www redirect
	t.Log("www redirect")
	a.cfg, _ = config.ParseString(`
		server {
			redirect {
				enable = true
				to = "www"
				code = 307
			}
		}
	`)

	type redirectTestCase struct {
		label    string
		fromURL  string
		status   int
		location string
	}

	runtestcase := func(testcases []redirectTestCase) {
		for _, tc := range testcases {
			t.Run(tc.label, func(t *testing.T) {
				w := httptest.NewRecorder()
				r := httptest.NewRequest(ahttp.MethodGet, tc.fromURL, nil)
				a.he.doRedirect(w, r)
				assert.Equal(t, tc.status, w.Code)
				assert.Equal(t, tc.location, w.Header().Get(ahttp.HeaderLocation))
			})
		}
	}

	testcases := []redirectTestCase{
		{
			label:    "www domain",
			fromURL:  "http://aahframework.org/home.html?rt=login",
			status:   http.StatusTemporaryRedirect,
			location: "http://www.aahframework.org/home.html?rt=login",
		},
		{
			label:    "www subdomain",
			fromURL:  "http://docs.aahframework.org",
			status:   http.StatusTemporaryRedirect,
			location: "http://www.docs.aahframework.org/",
		},
		{
			label:    "www domain already correct",
			fromURL:  "http://www.aahframework.org",
			status:   http.StatusOK,
			location: "",
		},
		{
			label:    "www subdomain already correct",
			fromURL:  "http://www.docs.aahframework.org",
			status:   http.StatusOK,
			location: "",
		},
	}

	runtestcase(testcases)

	// www redirect
	t.Log("non-www redirect")
	a.cfg, _ = config.ParseString(`
		server {
			redirect {
				enable = true
			}
		}
	`)

	testcases = []redirectTestCase{
		{
			label:    "non-www domain",
			fromURL:  "http://www.aahframework.org/home.html?rt=login",
			status:   http.StatusMovedPermanently,
			location: "http://aahframework.org/home.html?rt=login",
		},
		{
			label:    "non-www subdomain",
			fromURL:  "http://www.docs.aahframework.org",
			status:   http.StatusMovedPermanently,
			location: "http://docs.aahframework.org/",
		},
		{
			label:    "non-www domain already correct",
			fromURL:  "http://aahframework.org",
			status:   http.StatusOK,
			location: "",
		},
		{
			label:    "non-www subdomain already correct",
			fromURL:  "http://docs.aahframework.org",
			status:   http.StatusOK,
			location: "",
		},
	}

	runtestcase(testcases)
}

func newContext(w http.ResponseWriter, r *http.Request) *Context {
	ctx := &Context{}

	if r != nil {
		ctx.Req = ahttp.AcquireRequest(r)
	}

	if w != nil {
		ctx.Res = ahttp.AcquireResponseWriter(w)
	}

	return ctx
}
