// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"aahframework.org/ahttp.v0"
	"aahframework.org/test.v0/assert"
)

func TestEngineTestRequests(t *testing.T) {
	importPath := filepath.Join(testdataBaseDir(), "webapp1")
	ts, err := newTestServer(t, importPath)
	assert.Nil(t, err)
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

	testOnAfterReply := func(e *Event) {
		ctx := e.Data.(*Context)
		ctx.Log().Info("Application OnAfterReply extension point")
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
	ts.app.OnRequest(func(e *Event) {
		t.Log("Application OnRequest extension point")
	})
	ts.app.OnRequest(testOnRequest)

	ts.app.OnPreReply(func(e *Event) {
		t.Log("Application OnPreReply extension point")
	})
	ts.app.OnPreReply(testOnPreReply)

	ts.app.OnAfterReply(func(e *Event) {
		t.Log("Application OnAfterReply extension point")
	})
	ts.app.OnAfterReply(testOnAfterReply)

	ts.app.OnPreAuth(func(e *Event) {
		t.Log("Application OnPreAuth extension point")
	})
	ts.app.OnPreAuth(testOnPreAuth)

	ts.app.OnPostAuth(func(e *Event) {
		t.Log("Application OnPostAuth extension point")
	})
	ts.app.OnPostAuth(testOnPostAuth)

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

	// GET Binary bytes - /binary-bytes
	t.Log("GET Binary bytes - /binary-bytes")
	resp, err = httpClient.Get(ts.URL + "/binary-bytes")
	assert.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "text/plain; charset=utf-8", resp.Header.Get(ahttp.HeaderContentType))
	assert.Equal(t, "23", resp.Header.Get(ahttp.HeaderContentLength))
	assert.True(t, strings.Contains(responseBody(resp), "This is my Binary Bytes"))

	// GET Send File - /send-file
	t.Log("GET Send File - /send-file")
	resp, err = httpClient.Get(ts.URL + "/send-file")
	assert.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "text/css", resp.Header.Get(ahttp.HeaderContentType))
	assert.Equal(t, "inline; filename=aah.css", resp.Header.Get(ahttp.HeaderContentDisposition))
	assert.Equal(t, "700", resp.Header.Get(ahttp.HeaderContentLength))
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
