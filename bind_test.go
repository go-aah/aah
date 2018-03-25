// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/i18n.v0"
	"aahframework.org/router.v0"
	"aahframework.org/security.v0"
	"aahframework.org/test.v0/assert"
)

func TestBindParamTemplateFuncs(t *testing.T) {
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

func TestBindParse(t *testing.T) {
	defer ess.DeleteFiles("testapp.pid")
	requestParsers[ahttp.ContentTypeMultipartForm.Mime] = multipartFormParser
	requestParsers[ahttp.ContentTypeForm.Mime] = formParser

	// Request Query String
	r1 := httptest.NewRequest("GET", "http://localhost:8080/index.html?lang=en-CA", nil)
	ctx1 := &Context{
		Req:      ahttp.AcquireRequest(r1),
		Res:      ahttp.AcquireResponseWriter(httptest.NewRecorder()),
		subject:  security.AcquireSubject(),
		route:    &router.Route{MaxBodySize: 5 << 20},
		values:   make(map[string]interface{}),
		viewArgs: make(map[string]interface{}),
	}

	appI18n = i18n.New()

	assert.Nil(t, ctx1.Req.Locale)
	BindMiddleware(ctx1, &Middleware{})
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
	r2.Header.Set(ahttp.HeaderContentType, ahttp.ContentTypeForm.String())
	ctx2 := &Context{
		Req:      ahttp.AcquireRequest(r2),
		Res:      ahttp.AcquireResponseWriter(httptest.NewRecorder()),
		subject:  security.AcquireSubject(),
		values:   make(map[string]interface{}),
		viewArgs: make(map[string]interface{}),
		route:    &router.Route{MaxBodySize: 5 << 20},
	}

	BindMiddleware(ctx2, &Middleware{})
	assert.NotNil(t, ctx2.Req.Params.Form)
	assert.True(t, len(ctx2.Req.Params.Form) == 3)

	// Request Form Multipart
	r3, _ := http.NewRequest("POST", "http://localhost:8080/user/registration", strings.NewReader(form.Encode()))
	r3.Header.Set(ahttp.HeaderContentType, ahttp.ContentTypeMultipartForm.String())
	ctx3 := &Context{
		Req:      ahttp.AcquireRequest(r3),
		subject:  security.AcquireSubject(),
		values:   make(map[string]interface{}),
		viewArgs: make(map[string]interface{}),
		route:    &router.Route{MaxBodySize: 5 << 20},
	}
	BindMiddleware(ctx3, &Middleware{})
	assert.Nil(t, ctx3.Req.Params.Form)
	assert.False(t, len(ctx3.Req.Params.Form) == 3)
}

func TestBindParamParseLocaleFromAppConfiguration(t *testing.T) {
	defer ess.DeleteFiles("testapp.pid")

	cfg, err := config.ParseString(`
		i18n {
			param_name {
				query = "language"
			}
		}
	`)
	appConfig = cfg
	bindInitialize(&Event{})

	assert.Nil(t, err)

	r := httptest.NewRequest("GET", "http://localhost:8080/index.html?language=en-CA", nil)
	ctx1 := &Context{
		Req:      ahttp.AcquireRequest(r),
		viewArgs: make(map[string]interface{}),
		values:   make(map[string]interface{}),
	}

	assert.Nil(t, ctx1.Req.Locale)
	BindMiddleware(ctx1, &Middleware{})
	assert.NotNil(t, ctx1.Req.Locale)
	assert.Equal(t, "en", ctx1.Req.Locale.Language)
	assert.Equal(t, "CA", ctx1.Req.Locale.Region)
	assert.Equal(t, "en-CA", ctx1.Req.Locale.String())
}

func TestBindParamContentNegotiation(t *testing.T) {
	defer ess.DeleteFiles("testapp.pid")

	errorHandlerFunc = defaultErrorHandlerFunc
	isContentNegotiationEnabled = true

	// Accepted
	acceptedContentTypes = []string{"application/json"}
	r1 := httptest.NewRequest("POST", "http://localhost:8080/v1/userinfo", nil)
	r1.Header.Set(ahttp.HeaderContentType, "application/xml")
	ctx1 := &Context{
		Req:     ahttp.AcquireRequest(r1),
		reply:   acquireReply(),
		subject: security.AcquireSubject(),
	}
	BindMiddleware(ctx1, &Middleware{})
	assert.Equal(t, http.StatusUnsupportedMediaType, ctx1.Reply().err.Code)

	// Offered
	offeredContentTypes = []string{"application/json"}
	r2 := httptest.NewRequest("POST", "http://localhost:8080/v1/userinfo", nil)
	r2.Header.Set(ahttp.HeaderContentType, "application/json")
	r2.Header.Set(ahttp.HeaderAccept, "application/xml")
	ctx2 := &Context{
		Req:     ahttp.AcquireRequest(r2),
		reply:   acquireReply(),
		subject: security.AcquireSubject(),
	}
	BindMiddleware(ctx2, &Middleware{})
	assert.Equal(t, http.StatusNotAcceptable, ctx2.Reply().err.Code)

	isContentNegotiationEnabled = false

	appConfig, _ = config.ParseString(`
		request {
			content_negotiation {
				accepted = ["*/*"]
				offered = ["*/*"]
			}
		}`)
	bindInitialize(&Event{})
	appConfig = nil
}

func TestBindAddValueParser(t *testing.T) {
	err := AddValueParser(reflect.TypeOf(time.Time{}), func(key string, typ reflect.Type, params url.Values) (reflect.Value, error) {
		return reflect.Value{}, nil
	})
	assert.NotNil(t, err)
	assert.Equal(t, "valpar: value parser is already exists", err.Error())
}

func TestBindFormBodyNil(t *testing.T) {
	// Request Body is nil
	r1, _ := http.NewRequest("POST", "http://localhost:8080/user/registration", nil)
	ctx1 := &Context{Req: ahttp.AcquireRequest(r1), subject: security.AcquireSubject()}
	result := formParser(ctx1)
	assert.Equal(t, flowCont, result)
}
