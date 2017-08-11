// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"aahframework.org/ahttp.v0"
	config "aahframework.org/config.v0"
	"aahframework.org/test.v0/assert"
)

func TestErrorHandler(t *testing.T) {
	// 400
	ctx1 := &Context{reply: acquireReply()}
	ctx1.Reply().ContentType("application/json")
	handleError(ctx1, &Error{
		Code:    http.StatusBadRequest,
		Message: http.StatusText(http.StatusBadRequest),
	})
	assert.NotNil(t, ctx1.Reply().Rdr)
	jsonr := ctx1.Reply().Rdr.(*JSON)
	assert.NotNil(t, jsonr)
	assert.NotNil(t, jsonr.Data)
	assert.Equal(t, 400, jsonr.Data.(*Error).Code)
	assert.Equal(t, "Bad Request", jsonr.Data.(*Error).Message)

	// 500
	ctx2 := &Context{reply: acquireReply()}
	ctx2.Reply().ContentType("application/xml")
	handleError(ctx2, &Error{
		Code:    http.StatusInternalServerError,
		Message: http.StatusText(http.StatusInternalServerError),
	})
	assert.NotNil(t, ctx2.Reply().Rdr)
	xmlr := ctx2.Reply().Rdr.(*XML)
	assert.NotNil(t, xmlr)
	assert.NotNil(t, xmlr.Data)
	assert.Equal(t, 500, xmlr.Data.(*Error).Code)
	assert.Equal(t, "Internal Server Error", xmlr.Data.(*Error).Message)

	SetErrorHandler(func(ctx *Context, err *Error) {
		t.Log(ctx, err)
	})
}

func TestErrorDefaultHandler(t *testing.T) {
	appCfg, _ := config.ParseString("")
	viewDir := filepath.Join(getTestdataPath(), appViewsDir())
	err := initViewEngine(viewDir, appCfg)
	assert.Nil(t, err)
	assert.NotNil(t, AppViewEngine())

	// 400
	r1 := httptest.NewRequest("GET", "http://localhost:8080/get-involved.html", nil)
	ctx1 := &Context{Req: ahttp.AcquireRequest(r1), reply: acquireReply()}
	ctx1.Reply().ContentType(ahttp.ContentTypeHTML.String())
	defaultErrorHandler(ctx1, &Error{Code: http.StatusNotFound, Message: "Test message"})
	html := ctx1.Reply().Rdr.(*HTML)
	t.Logf("%+v\n", html)
	assert.True(t, defaultErrorHTMLTemplate == html.Template)
	assert.Equal(t, "404.html", html.Filename)

	// 500
	r2 := httptest.NewRequest("GET", "http://localhost:8080/get-involved.html", nil)
	ctx2 := &Context{Req: ahttp.AcquireRequest(r2), reply: acquireReply()}
	ctx2.Reply().ContentType(ahttp.ContentTypeHTML.String())
	defaultErrorHandler(ctx2, &Error{Code: http.StatusInternalServerError, Message: "Test message"})
	html = ctx2.Reply().Rdr.(*HTML)
	t.Logf("%+v\n", html)
	assert.True(t, defaultErrorHTMLTemplate == html.Template)
	assert.Equal(t, "500.html", html.Filename)
}
