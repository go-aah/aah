// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/log.v0"
	"aahframework.org/security.v0"
	"aahframework.org/test.v0/assert"
)

func TestErrorHandler(t *testing.T) {
	// 400
	ctx1 := &Context{
		Req:     getAahRequest("GET", "http://localhost:8080", ""),
		subject: security.AcquireSubject(),
		reply:   acquireReply(),
	}
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
	ctx2 := &Context{
		Req:     getAahRequest("GET", "http://localhost:8080", ""),
		subject: security.AcquireSubject(),
		reply:   acquireReply(),
	}
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

	SetErrorHandler(func(ctx *Context, err *Error) bool {
		t.Log(ctx, err)
		return false
	})

	// 403
	ctx3 := &Context{
		Req:     getAahRequest("GET", "http://localhost:8080", ""),
		subject: security.AcquireSubject(),
		reply:   acquireReply(),
	}
	ctx3.Reply().ContentType("text/plain")
	handleError(ctx3, &Error{
		Code:    http.StatusForbidden,
		Message: http.StatusText(http.StatusForbidden),
	})
	assert.NotNil(t, ctx3.Reply().Rdr)
	plain := ctx3.Reply().Rdr.(*Text)
	assert.NotNil(t, plain)
	assert.Equal(t, "[403 Forbidden]", fmt.Sprint(plain.Values))
}

func TestErrorDefaultHandler(t *testing.T) {
	appCfg, _ := config.ParseString("")
	viewDir := filepath.Join(getTestdataPath(), appViewsDir())
	err := initViewEngine(viewDir, appCfg)
	assert.Nil(t, err)
	assert.NotNil(t, AppViewEngine())

	// 400
	r1 := httptest.NewRequest("GET", "http://localhost:8080/get-involved.html", nil)
	ctx1 := &Context{Req: ahttp.AcquireRequest(r1), subject: security.AcquireSubject(), reply: acquireReply()}
	ctx1.Reply().ContentType(ahttp.ContentTypeHTML.String())
	defaultErrorHandlerFunc(ctx1, &Error{Code: http.StatusNotFound, Message: "Test message"})
	html := ctx1.Reply().Rdr.(*HTML)
	t.Logf("%+v\n", html)
	assert.True(t, defaultErrorHTMLTemplate == html.Template)
	assert.Equal(t, "404.html", html.Filename)

	// 500
	r2 := httptest.NewRequest("GET", "http://localhost:8080/get-involved.html", nil)
	ctx2 := &Context{Req: ahttp.AcquireRequest(r2), subject: security.AcquireSubject(), reply: acquireReply()}
	ctx2.Reply().ContentType(ahttp.ContentTypeHTML.String())
	defaultErrorHandlerFunc(ctx2, &Error{Code: http.StatusInternalServerError, Message: "Test message"})
	html = ctx2.Reply().Rdr.(*HTML)
	t.Logf("%+v\n", html)
	assert.True(t, defaultErrorHTMLTemplate == html.Template)
	assert.Equal(t, "500.html", html.Filename)
}

type testErrorController struct {
}

func (tec *testErrorController) HandleError(err *Error) bool {
	log.Info("I have handler it")
	return true
}

func TestErrorCallControllerHandler(t *testing.T) {
	// 400
	ctx1 := &Context{
		Req:     getAahRequest("GET", "http://localhost:8080", ""),
		target:  &testErrorController{},
		subject: security.AcquireSubject(),
		reply:   acquireReply(),
	}
	ctx1.Reply().ContentType("application/json")
	handleError(ctx1, &Error{
		Code:    http.StatusBadRequest,
		Message: http.StatusText(http.StatusBadRequest),
	})
}
