// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package router

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-aah/aah/ahttp"
	"github.com/go-aah/essentials"
	"github.com/go-aah/test/assert"
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Test Path Params methods
//___________________________________

func TestPathParamGet(t *testing.T) {
	pathParameters := PathParams{
		PathParam{"dir", "js"},
		PathParam{"filepath", "/inc/framework.js"},
	}

	fp := pathParameters.Get("filepath")
	assert.Equal(t, "/inc/framework.js", fp)

	notfound := pathParameters.Get("notfound")
	assert.Equal(t, "", notfound)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Test Router methods
//___________________________________

func TestRouterLoadConfiguration(t *testing.T) {
	router := createRouter("routes.conf")

	err := router.Load()
	assert.FailNowOnError(t, err, "")

	// TODO validate routes
}

func TestRouterErrorLoadConfiguration(t *testing.T) {
	router := createRouter("routes-error.conf")

	err := router.Load()
	assert.NotNilf(t, err, "expected error loading ''%v'", "routes-error.conf")
}

func TestRouterNoDomainRoutesFound(t *testing.T) {
	router := createRouter("routes-no-domains.conf")

	err := router.Load()
	assert.Equal(t, ErrNoRoutesConfigFound, err)
}

func TestRouterReloadConfiguration(t *testing.T) {
	router := createRouter("routes.conf")

	err := router.Reload()
	assert.FailNowOnError(t, err, "")

	// TODO validate routes after reload
}

func TestRouterDomainConfig(t *testing.T) {
	router := createRouter("routes.conf")
	err := router.Load()
	assert.FailNowOnError(t, err, "")

	domain := router.Domain(createHTTPRequest("localhost:8000", ""))
	assert.NotNil(t, domain)

	domain = router.Domain(createHTTPRequest("www.aahframework.org", ""))
	assert.Nil(t, domain)
}

func TestRouterDomainAllowed(t *testing.T) {
	router := createRouter("routes.conf")
	err := router.Load()
	assert.FailNowOnError(t, err, "")

	req := createHTTPRequest("localhost:8000", "/")
	allow := router.Allowed(req)
	assert.NotNil(t, allow)
	assert.True(t, strings.Contains(allow, ahttp.MethodOptions))

	domain := router.Domain(req)
	allow = domain.allowed("POST", "*")
	assert.NotNil(t, allow)
	assert.True(t, strings.Contains(allow, ahttp.MethodOptions))

	// domain not exists
	reqNotExists := createHTTPRequest("notexists:8000", "/")
	allow = router.Allowed(reqNotExists)
	assert.True(t, ess.IsStrEmpty(allow))

	// TODO do more
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Test Domain methods
//___________________________________

func TestDomainAddRoute(t *testing.T) {
	domain := &Domain{
		Host: "aahframework.org",
	}

	route1 := &Route{
		Name:       "route1",
		Path:       "/info/:user/project/:project",
		Method:     "GET",
		Controller: "Info",
		Action:     "ShowProject",
	}
	err := domain.addRoute(route1)
	assert.FailNowOnError(t, err, "unexpected error")

	route2 := &Route{
		Name:       "index",
		Path:       "/",
		Method:     "GET",
		Controller: "App",
		Action:     "Index",
	}
	err = domain.addRoute(route2)
	assert.FailNowOnError(t, err, "unexpected error")

	routeError := &Route{
		Name:       "route_error",
		Path:       "/",
		Method:     "GET",
		Controller: "App",
		Action:     "Index",
	}
	err = domain.addRoute(routeError)
	assert.True(t, strings.Contains(err.Error(), "value is already registered"))
}

func createRouter(filename string) *Router {
	wd, _ := os.Getwd()
	return New(filepath.Join(wd, "testdata", filename))
}

func createHTTPRequest(host, path string) *ahttp.Request {
	req := &ahttp.Request{
		Request: &http.Request{Host: host},
	}

	if !ess.IsStrEmpty(path) {
		req.URL = &url.URL{Path: path}
	}

	return req
}
