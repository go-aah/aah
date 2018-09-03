// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"aahframe.work/aah/ahttp"
	"aahframe.work/aah/config"
	"aahframe.work/aah/essentials"
	"aahframe.work/aah/log"
	"github.com/stretchr/testify/assert"
)

func TestBindParamContentNegotiation(t *testing.T) {
	defer ess.DeleteFiles("testapp.pid")

	a := newApp()
	cfg, _ := config.ParseString(`request {
    content_negotiation {
      enable = true
      accepted = ["application/json"]
      offered = ["application/json"]
    }
  }`)
	a.cfg = cfg
	err := a.initLog()
	assert.Nil(t, err)

	err = a.initBind()
	assert.Nil(t, err)

	a.Log().(*log.Logger).SetWriter(ioutil.Discard)

	// Accepted
	r1 := httptest.NewRequest("POST", "http://localhost:8080/v1/userinfo", nil)
	r1.Header.Set(ahttp.HeaderContentType, "application/xml")
	ctx1 := newContext(nil, r1)
	ctx1.a = a
	BindMiddleware(ctx1, &Middleware{})
	assert.Equal(t, http.StatusUnsupportedMediaType, ctx1.Reply().err.Code)

	// Offered
	r2 := httptest.NewRequest("POST", "http://localhost:8080/v1/userinfo", nil)
	r2.Header.Set(ahttp.HeaderContentType, "application/json")
	r2.Header.Set(ahttp.HeaderAccept, "application/xml")
	ctx2 := newContext(nil, r2)
	ctx2.a = a
	BindMiddleware(ctx2, &Middleware{})
	assert.Equal(t, http.StatusNotAcceptable, ctx2.Reply().err.Code)
}

func TestBindAddValueParser(t *testing.T) {
	err := AddValueParser(reflect.TypeOf(time.Time{}), func(key string, typ reflect.Type, params url.Values) (reflect.Value, error) {
		return reflect.Value{}, nil
	})
	assert.NotNil(t, err)
	assert.Equal(t, "valpar: value parser is already exists", err.Error())
}

func TestBindValidatorWithValue(t *testing.T) {
	assert.NotNil(t, Validator())

	// Validation failed
	i := 15
	result := ValidateValue(i, "gt=1,lt=10")
	assert.False(t, result)

	emailAddress := "sample@sample"
	result = ValidateValue(emailAddress, "required,email")
	assert.False(t, result)

	numbers := []int{23, 67, 87, 23, 90}
	result = ValidateValue(numbers, "unique")
	assert.False(t, result)

	// validation pass
	i = 9
	result = ValidateValue(i, "gt=1,lt=10")
	assert.True(t, result)

	emailAddress = "sample@sample.com"
	result = ValidateValue(emailAddress, "required,email")
	assert.True(t, result)

	numbers = []int{23, 67, 87, 56, 90}
	result = ValidateValue(numbers, "unique")
	assert.True(t, result)
}

func TestBindParamTemplateFuncs(t *testing.T) {
	a := newApp()
	a.viewMgr = &viewManager{a: a}

	form := url.Values{}
	form.Add("names", "Test1")
	form.Add("names", "Test 2 value")
	form.Add("username", "welcome")
	form.Add("email", "welcome@welcome.com")
	req1, _ := http.NewRequest("POST", "http://localhost:8080/user/registration?_ref=true&locale=en-CA", strings.NewReader(form.Encode()))
	req1.Header.Add(ahttp.HeaderContentType, ahttp.ContentTypeForm.Raw())
	_ = req1.ParseForm()

	aahReq1 := ahttp.ParseRequest(req1, &ahttp.Request{})
	aahReq1.URLParams = ahttp.URLParams{{Key: "userId", Value: "100001"}}

	viewArgs := map[string]interface{}{}
	viewArgs[KeyViewArgRequest] = aahReq1

	v1 := a.viewMgr.tmplQueryParam(viewArgs, "_ref")
	assert.Equal(t, "true", v1)

	v2 := a.viewMgr.tmplFormParam(viewArgs, "email")
	assert.Equal(t, "welcome@welcome.com", v2)

	v3 := a.viewMgr.tmplPathParam(viewArgs, "userId")
	assert.Equal(t, "100001", v3)
}
