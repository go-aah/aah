// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"io/ioutil"
	"net/http/httptest"
	"reflect"
	"testing"

	"aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/log.v0"
	"aahframework.org/router.v0"
	"aahframework.org/test.v0/assert"
)

type (
	Anonymous1 struct {
		Name string
	}

	Func1 func(e *Event)

	Level1 struct{ *Context }

	Level2 struct{ Level1 }

	Level3 struct{ Level2 }

	Level4 struct{ Level3 }

	Path1 struct {
		Anonymous Anonymous1
		*Context
	}

	Path2 struct {
		Level1
		Path1
		Level4
		Func1
	}
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
	cfg, _ := config.ParseString("")
	a.cfg = cfg
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
	cfg, _ := config.ParseString("")
	a.cfg = cfg
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

func TestContextEmbeddedAndController(t *testing.T) {
	a := newApp()

	a.AddController((*Level1)(nil), []*MethodInfo{
		{
			Name:       "Index",
			Parameters: []*ParameterInfo{},
		},
	})
	a.AddController((*Level2)(nil), []*MethodInfo{
		{
			Name:       "Scope",
			Parameters: []*ParameterInfo{},
		},
	})
	a.AddController((*Level3)(nil), []*MethodInfo{
		{
			Name: "Testing",
			Parameters: []*ParameterInfo{
				{
					Name: "userId",
					Type: reflect.TypeOf((*int)(nil)),
				},
			},
		},
	})
	a.AddController((*Level4)(nil), nil)
	a.AddController((*Path1)(nil), nil)
	a.AddController((*Path2)(nil), nil)

	testEmbeddedIndexes(t, Level1{}, [][]int{{0}})
	testEmbeddedIndexes(t, Level2{}, [][]int{{0, 0}})
	testEmbeddedIndexes(t, Level3{}, [][]int{{0, 0, 0}})
	testEmbeddedIndexes(t, Level4{}, [][]int{{0, 0, 0, 0}})
	testEmbeddedIndexes(t, Path1{}, [][]int{{1}})
	testEmbeddedIndexes(t, Path2{}, [][]int{{0, 0}, {1, 1}, {2, 0, 0, 0, 0}})
}

func testEmbeddedIndexes(t *testing.T, c interface{}, expected [][]int) {
	actual := findEmbeddedContext(reflect.TypeOf(c))
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Indexes do not match. expected %v actual %v", expected, actual)
	}
}
