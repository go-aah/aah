// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"io/ioutil"
	"net/http/httptest"
	"testing"

	"aahframework.org/ahttp"
	"aahframework.org/config"
	"aahframework.org/log"
	"aahframework.org/router"
	"github.com/stretchr/testify/assert"
)

func TestContextSubdomain(t *testing.T) {
	testSubdomainValue(t, "username1.sample.com", "username1", true)

	testSubdomainValue(t, "username2.sample.com", "username2", true)

	testSubdomainValue(t, "admin.username1.sample.com", "admin", true)

	testSubdomainValue(t, "sample.com", "", false)
}

func testSubdomainValue(t *testing.T, host, subdomain string, isSubdomain bool) {
	ctx := &Context{
		Req:    &ahttp.Request{Host: host},
		domain: &router.Domain{IsSubDomain: isSubdomain},
	}

	assert.Equal(t, subdomain, ctx.Subdomain())
}

func TestContextSetURL(t *testing.T) {
	a := newApp()
	a.cfg = config.NewEmpty()
	err := a.initLog()
	assert.Nil(t, err)

	a.Log().(*log.Logger).SetWriter(ioutil.Discard)

	req := httptest.NewRequest("POST", "http://localhost:8080/users/edit", nil)
	ctx := newContext(nil, req)
	ctx.a = a

	assert.Equal(t, "localhost:8080", ctx.Req.Host)
	assert.Equal(t, "POST", ctx.Req.Method)
	assert.Equal(t, "/users/edit", ctx.Req.Path)
	assert.False(t, ctx.decorated)

	// No effects, since decorated is false
	ctx.SetURL("http://status.localhost:8080/maintenance")
	assert.Equal(t, "/users/edit", ctx.Req.Path)
	assert.Equal(t, "localhost:8080", ctx.Req.Host)

	// now it affects
	ctx.decorated = true
	ctx.SetURL("http://status.localhost:8080/maintenance")
	assert.True(t, ctx.decorated)
	assert.Equal(t, "status.localhost:8080", ctx.Req.Host)
	assert.Equal(t, "POST", ctx.Req.Method) // no change expected
	assert.Equal(t, "/maintenance", ctx.Req.Path)

	// incorrect URL
	ctx.SetURL("http://status. localhost :8080//maintenance")
	assert.Equal(t, "status.localhost:8080", ctx.Req.Host)
	assert.Equal(t, "POST", ctx.Req.Method) // no change expected
	assert.Equal(t, "/maintenance", ctx.Req.Path)
}

func TestContextSetMethod(t *testing.T) {
	a := newApp()
	a.cfg = config.NewEmpty()
	err := a.initLog()
	assert.Nil(t, err)

	a.Log().(*log.Logger).SetWriter(ioutil.Discard)

	req := httptest.NewRequest("POST", "http://localhost:8080/users/edit", nil)
	ctx := newContext(nil, req)
	ctx.a = a

	assert.Equal(t, "localhost:8080", ctx.Req.Host)
	assert.Equal(t, "POST", ctx.Req.Method)
	assert.Equal(t, "/users/edit", ctx.Req.Path)
	assert.False(t, ctx.decorated)

	// No effects, since decorated is false
	ctx.SetMethod("GET")
	assert.Equal(t, "POST", ctx.Req.Method)

	// now it affects
	ctx.decorated = true
	ctx.SetMethod("get")
	assert.Equal(t, "GET", ctx.Req.Method)
	assert.Equal(t, "localhost:8080", ctx.Req.Host) // no change expected
	assert.Equal(t, "/users/edit", ctx.Req.Path)    // no change expected
	assert.True(t, ctx.decorated)

	// invalid method
	ctx.SetMethod("nomethod")
	assert.Equal(t, "GET", ctx.Req.Method)
}
