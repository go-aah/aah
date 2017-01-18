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

	"aahframework.org/aah/ahttp"
	"aahframework.org/essentials"
	"aahframework.org/test/assert"
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
	err := createRouter("routes.conf")
	assert.FailNowOnError(t, err, "")

	// After loading just couple of assertion
	reqCancelBooking1 := createHTTPRequest("localhost:8000", "/hotels/12345/cancel")
	reqCancelBooking1.Method = ahttp.MethodPost
	domain := FindDomain(reqCancelBooking1)
	route, pathParam, rts := domain.Lookup(reqCancelBooking1)
	assert.Equal(t, "cancel_booking", route.Name)
	assert.Equal(t, "12345", pathParam.Get("id"))
	assert.False(t, rts)

	// possible redirect trailing slash
	reqCancelBooking2 := createHTTPRequest("localhost:8000", "/hotels/12345/cancel/")
	reqCancelBooking2.Method = ahttp.MethodPost
	domain = FindDomain(reqCancelBooking2)
	_, _, rts = domain.Lookup(reqCancelBooking2)
	assert.True(t, rts)

	// Lookup by name
	cancelBooking := domain.LookupByName("cancel_booking")
	assert.Equal(t, "hotels_group", cancelBooking.ParentName)
	assert.Equal(t, "cancel_booking", cancelBooking.Name)
	assert.Equal(t, "Hotel", cancelBooking.Controller)
	assert.Equal(t, "POST", cancelBooking.Method)
}

func TestRouterStaticLoadConfiguration(t *testing.T) {
	err := createRouter("routes.conf")
	assert.FailNowOnError(t, err, "")

	// After loading just couple assertion for static

	// /favicon.ico
	req1 := createHTTPRequest("localhost:8000", "/favicon.ico")
	req1.Method = ahttp.MethodGet
	domain := FindDomain(req1)
	route, pathParam, rts := domain.Lookup(req1)
	assert.NotNil(t, pathParam)
	assert.False(t, rts)
	assert.True(t, route.IsStatic)
	assert.Equal(t, "/public/img/favicon.png", route.File)
	assert.Equal(t, "", route.Dir)

	// /static/img/aahframework.png
	req2 := createHTTPRequest("localhost:8000", "/static/img/aahframework.png")
	req2.Method = ahttp.MethodGet
	domain = FindDomain(req2)
	route, pathParam, rts = domain.Lookup(req2)
	assert.NotNil(t, pathParam)
	assert.False(t, rts)
	assert.True(t, route.IsStatic)
	assert.Equal(t, "/public", route.Dir)
	assert.Equal(t, "/img/aahframework.png", pathParam.Get("filepath"))
	assert.Equal(t, "", route.File)
}

func TestRouterErrorLoadConfiguration(t *testing.T) {
	err := createRouter("routes-error.conf")
	assert.NotNilf(t, err, "expected error loading '%v'", "routes-error.conf")
}

func TestRouterErrorHostLoadConfiguration(t *testing.T) {
	err := createRouter("routes-no-hostname.conf")
	assert.NotNilf(t, err, "expected error loading '%v'", "routes-no-hostname.conf")
}

func TestRouterErrorPathLoadConfiguration(t *testing.T) {
	err := createRouter("routes-path-error.conf")
	assert.NotNilf(t, err, "expected error loading '%v'", "routes-path-error.conf")
}

func TestRouterErrorControllerLoadConfiguration(t *testing.T) {
	err := createRouter("routes-controller-error.conf")
	assert.NotNilf(t, err, "expected error loading '%v'", "routes-controller-error.conf")
}

func TestRouterErrorStaticPathLoadConfiguration(t *testing.T) {
	err := createRouter("routes-static-path-error.conf")
	assert.NotNilf(t, err, "expected error loading '%v'", "routes-static-path-error.conf")
}

func TestRouterErrorStaticPathPatternLoadConfiguration(t *testing.T) {
	err := createRouter("routes-static-path-pattern-error.conf")
	assert.NotNilf(t, err, "expected error loading '%v'", "routes-static-path-pattern-error.conf")
}

func TestRouterErrorStaticDirFileLoadConfiguration(t *testing.T) {
	err := createRouter("routes-static-dir-file-error.conf")
	assert.NotNilf(t, err, "expected error loading '%v'", "routes-static-dir-file-error.conf")
}

func TestRouterErrorStaticNoDirFileLoadConfiguration(t *testing.T) {
	err := createRouter("routes-static-no-dir-file-error.conf")
	assert.NotNilf(t, err, "expected error loading '%v'", "routes-static-no-dir-file-error.conf")
}

func TestRouterNoDomainRoutesFound(t *testing.T) {
	err := createRouter("routes-no-domains.conf")
	assert.Equal(t, ErrNoRoutesConfigFound, err)
}

func TestRouterReloadConfiguration(t *testing.T) {
	err := createRouter("routes.conf")
	assert.FailNowOnError(t, err, "")

	err = Reload()
	assert.FailNowOnError(t, err, "")

	// After loading just couple of assertion
	reqCancelBooking1 := createHTTPRequest("localhost:8000", "/hotels/12345/cancel")
	reqCancelBooking1.Method = ahttp.MethodPost
	domain := FindDomain(reqCancelBooking1)
	route, pathParam, rts := domain.Lookup(reqCancelBooking1)
	assert.Equal(t, "cancel_booking", route.Name)
	assert.Equal(t, "12345", pathParam.Get("id"))
	assert.False(t, rts)

	// possible redirect trailing slash
	reqCancelBooking2 := createHTTPRequest("localhost:8000", "/hotels/12345/cancel/")
	reqCancelBooking2.Method = ahttp.MethodPost
	domain = FindDomain(reqCancelBooking2)
	_, _, rts = domain.Lookup(reqCancelBooking2)
	assert.True(t, rts)

	// Lookup by name
	cancelBooking := domain.LookupByName("cancel_booking")
	assert.Equal(t, "hotels_group", cancelBooking.ParentName)
	assert.Equal(t, "cancel_booking", cancelBooking.Name)
	assert.Equal(t, "Hotel", cancelBooking.Controller)
	assert.Equal(t, "POST", cancelBooking.Method)
}

func TestRouterDomainConfig(t *testing.T) {
	err := createRouter("routes.conf")
	assert.FailNowOnError(t, err, "")

	domain := FindDomain(createHTTPRequest("localhost:8000", ""))
	assert.NotNil(t, domain)

	domain = FindDomain(createHTTPRequest("www.aahframework.org", ""))
	assert.Nil(t, domain)
}

func TestRouterDomainAddresses(t *testing.T) {
	err := createRouter("routes.conf")
	assert.FailNowOnError(t, err, "")

	addresses := DomainAddresses()
	assert.Equal(t, "localhost:8000", addresses[0])
}

func TestRegisteredActions(t *testing.T) {
	err := createRouter("routes.conf")
	assert.FailNowOnError(t, err, "")

	methods := RegisteredActions()
	assert.NotNil(t, methods)
}

func TestIsDefaultAction(t *testing.T) {
	v1 := IsDefaultAction("Index")
	assert.True(t, v1)

	v2 := IsDefaultAction("Head")
	assert.True(t, v2)

	v3 := IsDefaultAction("Show")
	assert.False(t, v3)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Test Domain methods
//___________________________________

func TestDomainAllowed(t *testing.T) {
	err := createRouter("routes.conf")
	assert.FailNowOnError(t, err, "")

	req := createHTTPRequest("localhost:8000", "/")
	domain := FindDomain(req)
	allow := domain.Allowed(ahttp.MethodGet, "/")
	assert.NotNil(t, allow)
	assert.True(t, strings.Contains(allow, ahttp.MethodOptions))

	domain = FindDomain(req)
	allow = domain.Allowed(ahttp.MethodPost, "*")
	assert.NotNil(t, allow)
	assert.True(t, strings.Contains(allow, ahttp.MethodOptions))

	// domain not exists
	reqNotExists := createHTTPRequest("notexists:8000", "/")
	domain = FindDomain(reqNotExists)
	assert.Nil(t, domain)
}

func TestDomainReverseURL(t *testing.T) {
	err := createRouter("routes.conf")
	assert.FailNowOnError(t, err, "")

	req := createHTTPRequest("localhost:8000", "/")
	domain := FindDomain(req)

	// route name not exists
	emptyURL := domain.Reverse("not_exists_routename")
	assert.Equal(t, "", emptyURL)

	// non key-value pair value leads to error
	emptyURL = domain.Reverse("book_hotels", 12345678)
	assert.Equal(t, "", emptyURL)

	// not enough arguments
	emptyURL = domain.Reverse("book_hotels")
	assert.Equal(t, "", emptyURL)

	// incorrect key name scenario
	emptyURL = domain.Reverse("book_hotels", map[string]string{
		"idvalue": "12345678",
	})
	assert.Equal(t, "", emptyURL)

	// static URL
	loginURL := domain.Reverse("login")
	assert.Equal(t, "/login", loginURL)

	// success scenario
	bookingURL := domain.Reverse("book_hotels", map[string]string{
		"id": "12345678",
	})
	assert.Equal(t, "/hotels/12345678/booking", bookingURL)

	bookingURL = domain.Reverse("book_hotels", map[string]string{
		"id":     "12345678",
		"param1": "param1value",
		"param2": "param2value",
	})
	assert.Equal(t, "/hotels/12345678/booking?param1=param1value&param2=param2value", bookingURL)
}

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

func createRouter(filename string) error {
	wd, _ := os.Getwd()
	return Load(filepath.Join(wd, "testdata", filename))
}

func createHTTPRequest(host, path string) *ahttp.Request {
	req := &ahttp.Request{
		Raw: &http.Request{Host: host},
	}

	req.Host = req.Raw.Host

	if !ess.IsStrEmpty(path) {
		req.Raw.URL = &url.URL{Path: path}
		req.Path = req.Raw.URL.Path
	}

	return req
}
