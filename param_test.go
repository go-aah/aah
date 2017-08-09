// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/router.v0-unstable"
	"aahframework.org/test.v0/assert"
)

func TestParamTemplateFuncs(t *testing.T) {
	form := url.Values{}
	form.Add("names", "Test1")
	form.Add("names", "Test 2 value")
	form.Add("username", "welcome")
	form.Add("email", "welcome@welcome.com")
	req1, _ := http.NewRequest("POST", "http://localhost:8080/user/registration?_ref=true&locale=en-CA", strings.NewReader(form.Encode()))
	req1.Header.Add(ahttp.HeaderContentType, ahttp.ContentTypeForm.Raw())
	_ = req1.ParseForm()

	aahReq1 := ahttp.ParseRequest(req1, &ahttp.Request{})
	aahReq1.Params.Form = req1.Form
	aahReq1.Params.Path = make(map[string]string)
	aahReq1.Params.Path["userId"] = "100001"

	viewArgs := map[string]interface{}{}
	viewArgs[KeyViewArgRequestParams] = aahReq1.Params

	v1 := tmplQueryParam(viewArgs, "_ref")
	assert.Equal(t, "true", v1)

	v2 := tmplFormParam(viewArgs, "email")
	assert.Equal(t, "welcome@welcome.com", v2)

	v3 := tmplPathParam(viewArgs, "userId")
	assert.Equal(t, "100001", v3)
}

func TestParamParse(t *testing.T) {
	defer ess.DeleteFiles("testapp.pid")

	// Request Query String
	r1 := httptest.NewRequest("GET", "http://localhost:8080/index.html?lang=en-CA", nil)
	ctx1 := &Context{
		Req:      ahttp.AcquireRequest(r1),
		Res:      ahttp.AcquireResponseWriter(httptest.NewRecorder()),
		route:    &router.Route{MaxBodySize: 5 << 20},
		viewArgs: make(map[string]interface{}),
	}

	e := &engine{}

	assert.Nil(t, ctx1.Req.Locale)
	e.parseRequestParams(ctx1)
	assert.NotNil(t, ctx1.Req.Locale)
	assert.Equal(t, "en", ctx1.Req.Locale.Language)
	assert.Equal(t, "CA", ctx1.Req.Locale.Region)
	assert.Equal(t, "en-CA", ctx1.Req.Locale.String())

	// Request Form Values
	form := url.Values{}
	form.Add("names", "Test1")
	form.Add("names", "Test 2 value")
	form.Add("username", "welcome")
	form.Add("email", "welcome@welcome.com")
	r2, _ := http.NewRequest("POST", "http://localhost:8080/user/registration", strings.NewReader(form.Encode()))
	r2.Header.Add(ahttp.HeaderContentType, ahttp.ContentTypeForm.String())
	ctx2 := &Context{
		Req:      ahttp.AcquireRequest(r2),
		Res:      ahttp.AcquireResponseWriter(httptest.NewRecorder()),
		route:    &router.Route{MaxBodySize: 5 << 20},
		viewArgs: make(map[string]interface{}),
	}

	e.parseRequestParams(ctx2)
	assert.NotNil(t, ctx2.Req.Params.Form)
}

func TestParamParseLocaleFromAppConfiguration(t *testing.T) {
	defer ess.DeleteFiles("testapp.pid")

	cfg, err := config.ParseString(`
		i18n {
			param_name {
				query = "language"
			}
		}
	`)
	appConfig = cfg
	paramInitialize(&Event{})

	assert.Nil(t, err)

	r := httptest.NewRequest("GET", "http://localhost:8080/index.html?language=en-CA", nil)
	ctx1 := &Context{
		Req:      ahttp.AcquireRequest(r),
		viewArgs: make(map[string]interface{}),
	}

	e := &engine{}

	assert.Nil(t, ctx1.Req.Locale)
	e.parseRequestParams(ctx1)
	assert.NotNil(t, ctx1.Req.Locale)
	assert.Equal(t, "en", ctx1.Req.Locale.Language)
	assert.Equal(t, "CA", ctx1.Req.Locale.Region)
	assert.Equal(t, "en-CA", ctx1.Req.Locale.String())
}

func TestParamContentNegotiation(t *testing.T) {
	defer ess.DeleteFiles("testapp.pid")

	e := engine{}

	// Accepted
	isAcceptedExists = true
	acceptedContentTypes = []string{"application/json"}
	r1 := httptest.NewRequest("POST", "http://localhost:8080/v1/userinfo", nil)
	r1.Header.Set(ahttp.HeaderContentType, "application/xml")
	ctx1 := &Context{
		Req:   ahttp.AcquireRequest(r1),
		reply: acquireReply(),
	}
	result1 := e.parseRequestParams(ctx1)
	assert.True(t, result1 == 1)
	isAcceptedExists = false

	// Offered
	isOfferedExists = true
	offeredContentTypes = []string{"application/json"}
	r2 := httptest.NewRequest("POST", "http://localhost:8080/v1/userinfo", nil)
	r2.Header.Set(ahttp.HeaderAccept, "application/xml")
	ctx2 := &Context{
		Req:   ahttp.AcquireRequest(r2),
		reply: acquireReply(),
	}
	result2 := e.parseRequestParams(ctx2)
	assert.True(t, result2 == 1)
	isOfferedExists = false
}
