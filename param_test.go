// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"aahframework.org/ahttp.v0"
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
	viewArgs[keyRequestParams] = aahReq1.Params

	v1 := tmplQueryParam(viewArgs, "_ref")
	assert.Equal(t, "true", v1)

	v2 := tmplFormParam(viewArgs, "email")
	assert.Equal(t, "welcome@welcome.com", v2)

	v3 := tmplPathParam(viewArgs, "userId")
	assert.Equal(t, "100001", v3)
}
