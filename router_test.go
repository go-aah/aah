// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
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

	"aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0-unstable"
	"aahframework.org/test.v0/assert"
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Test Path Params methods
//___________________________________

func TestRouterPathParamGet(t *testing.T) {
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
	router, err := createRouter("routes.conf")
	assert.FailNowOnError(t, err, "")

	// After loading just couple of assertion
	reqCancelBooking1 := createHTTPRequest("localhost:8080", "/hotels/12345/cancel")
	reqCancelBooking1.Method = ahttp.MethodPost
	domain := router.FindDomain(reqCancelBooking1)
	route, pathParam, rts := domain.Lookup(reqCancelBooking1)
	assert.Equal(t, "cancel_booking", route.Name)
	assert.Equal(t, "12345", pathParam.Get("id"))
	assert.False(t, rts)
	assert.Equal(t, 1, pathParam.Len())

	// possible redirect trailing slash
	reqCancelBooking2 := createHTTPRequest("localhost:8080", "/hotels/12345/cancel/")
	reqCancelBooking2.Method = ahttp.MethodPost
	domain = router.FindDomain(reqCancelBooking2)
	_, _, rts = domain.Lookup(reqCancelBooking2)
	assert.True(t, rts)

	// Lookup by name
	cancelBooking := domain.LookupByName("cancel_booking")
	assert.Equal(t, "hotels_group", cancelBooking.ParentName)
	assert.Equal(t, "cancel_booking", cancelBooking.Name)
	assert.Equal(t, "Hotel", cancelBooking.Controller)
	assert.Equal(t, "POST", cancelBooking.Method)

	routeNotFound := domain.LookupByName("cancel_booking_not_found")
	assert.Nil(t, routeNotFound)

	// Method missing
	err = domain.AddRoute(&Route{
		Name: "MethodMissing",
		Path: "/:user/test",
	})
	assert.Equal(t, "router: method value is empty", err.Error())

	err = domain.AddRoute(&Route{
		Name:   "ErrorRoute",
		Path:   "/:user/test",
		Method: "GET",
	})
	assert.True(t, strings.HasPrefix(err.Error(), "wildcard route ':user' conflicts"))

	domain.Port = ""
	assert.Equal(t, "localhost", domain.key())
}

func TestRouterWildcardSubdomain(t *testing.T) {
	router, err := createRouter("routes.conf")
	assert.FailNowOnError(t, err, "")

	reqCancelBooking := createHTTPRequest("localhost:8080", "/hotels/12345/cancel")
	reqCancelBooking.Method = ahttp.MethodPost
	domain := router.FindDomain(reqCancelBooking)
	assert.Equal(t, "localhost", domain.Host)

	rootDomain := router.RootDomain()
	assert.Equal(t, "localhost", rootDomain.Host)
	assert.Equal(t, "8080", rootDomain.Port)

	reqWildcardUsername1 := createHTTPRequest("username1.localhost:8080", "/")
	reqWildcardUsername1.Method = ahttp.MethodGet
	domain = router.FindDomain(reqWildcardUsername1)
	assert.Equal(t, "*.localhost", domain.Host)
	assert.Equal(t, "8080", domain.Port)

	route1, _, rts1 := domain.Lookup(reqWildcardUsername1)
	assert.False(t, rts1)
	assert.Equal(t, "index", route1.Name)
	assert.Equal(t, "wildcard/AppController", route1.Controller)
	assert.Equal(t, "/", route1.Path)

	reqWildcardUsername2 := createHTTPRequest("username2.localhost:8080", "/")
	reqWildcardUsername2.Method = ahttp.MethodGet
	domain = router.FindDomain(reqWildcardUsername2)
	assert.Equal(t, "*.localhost", domain.Host)
	assert.Equal(t, "8080", domain.Port)

	route2, _, rts2 := domain.Lookup(reqWildcardUsername2)
	assert.False(t, rts2)
	assert.Equal(t, "index", route2.Name)
	assert.Equal(t, "wildcard/AppController", route2.Controller)
	assert.Equal(t, "/", route2.Path)
}

func TestRouterStaticLoadConfiguration(t *testing.T) {
	router, err := createRouter("routes.conf")
	assert.FailNowOnError(t, err, "")

	// After loading just couple assertion for static

	// /favicon.ico
	req1 := createHTTPRequest("localhost:8080", "/favicon.ico")
	req1.Method = ahttp.MethodGet
	domain := router.FindDomain(req1)
	route, pathParam, rts := domain.Lookup(req1)
	assert.NotNil(t, pathParam)
	assert.False(t, rts)
	assert.True(t, route.IsStatic)
	assert.Equal(t, "/public/img/favicon.png", route.File)
	assert.Equal(t, "", route.Dir)
	assert.False(t, route.IsDir())
	assert.True(t, route.IsFile())

	// /static/img/aahframework.png
	req2 := createHTTPRequest("localhost:8080", "/static/img/aahframework.png")
	req2.Method = ahttp.MethodGet
	domain = router.FindDomain(req2)
	route, pathParam, rts = domain.Lookup(req2)
	assert.NotNil(t, pathParam)
	assert.False(t, rts)
	assert.True(t, route.IsStatic)
	assert.Equal(t, "/public", route.Dir)
	assert.Equal(t, "/img/aahframework.png", pathParam.Get("filepath"))
	assert.Equal(t, "", route.File)
	assert.True(t, route.IsDir())
	assert.False(t, route.IsFile())
}

func TestRouterErrorLoadConfiguration(t *testing.T) {
	router, err := createRouter("routes-error.conf")
	assert.NotNilf(t, err, "expected error loading '%v'", "routes-error.conf")
	assert.NotNil(t, router)
	assert.True(t, strings.HasPrefix(err.Error(), "syntax error line"))
}

func TestRouterErrorHostLoadConfiguration(t *testing.T) {
	router, err := createRouter("routes-no-hostname.conf")
	assert.NotNilf(t, err, "expected error loading '%v'", "routes-no-hostname.conf")
	assert.NotNil(t, router)
	assert.Equal(t, "'localhost.host' key is missing", err.Error())
}

func TestRouterErrorPathLoadConfiguration(t *testing.T) {
	router, err := createRouter("routes-path-error.conf")
	assert.NotNilf(t, err, "expected error loading '%v'", "routes-path-error.conf")
	assert.NotNil(t, router)
	assert.Equal(t, "'app_index.path' key is missing", err.Error())
}

func TestRouterErrorControllerLoadConfiguration(t *testing.T) {
	router, err := createRouter("routes-controller-error.conf")
	assert.NotNilf(t, err, "expected error loading '%v'", "routes-controller-error.conf")
	assert.NotNil(t, router)
	assert.Equal(t, "'app_index.controller' key is missing", err.Error())
}

func TestRouterErrorNotFoundLoadConfiguration(t *testing.T) {
	router, err := createRouter("routes-notfound-controller-error.conf")
	assert.NotNilf(t, err, "expected error loading '%v'", "routes-notfound-controller-error.conf")
	assert.NotNil(t, router)
	assert.Equal(t, "'not_found.controller' key is missing", err.Error())

	router, err = createRouter("routes-notfound-action-error.conf")
	assert.NotNilf(t, err, "expected error loading '%v'", "routes-notfound-action-error.conf")
	assert.NotNil(t, router)
	assert.Equal(t, "'not_found.action' key is missing", err.Error())
}

func TestRouterErrorStaticPathLoadConfiguration(t *testing.T) {
	router, err := createRouter("routes-static-path-error.conf")
	assert.NotNilf(t, err, "expected error loading '%v'", "routes-static-path-error.conf")
	assert.NotNil(t, router)
	assert.Equal(t, "'static.public.path' key is missing", err.Error())
}

func TestRouterErrorStaticPathPatternLoadConfiguration(t *testing.T) {
	router, err := createRouter("routes-static-path-pattern-error.conf")
	assert.NotNilf(t, err, "expected error loading '%v'", "routes-static-path-pattern-error.conf")
	assert.NotNil(t, router)
	assert.Equal(t, "'static.public.path' parameters can not be used with static", err.Error())
}

func TestRouterErrorStaticDirFileLoadConfiguration(t *testing.T) {
	router, err := createRouter("routes-static-dir-file-error.conf")
	assert.NotNilf(t, err, "expected error loading '%v'", "routes-static-dir-file-error.conf")
	assert.NotNil(t, router)
	assert.Equal(t, "'static.public.dir' & 'static.public.file' key(s) cannot be used together", err.Error())
}

func TestRouterErrorStaticNoDirFileLoadConfiguration(t *testing.T) {
	router, err := createRouter("routes-static-no-dir-file-error.conf")
	assert.NotNilf(t, err, "expected error loading '%v'", "routes-static-no-dir-file-error.conf")
	assert.NotNil(t, router)
	assert.Equal(t, "either 'static.public.dir' or 'static.public.file' key have to be present", err.Error())
}

func TestRouterErrorStaticPathBeginSlashLoadConfiguration(t *testing.T) {
	router, err := createRouter("routes-static-path-slash-error.conf")
	assert.NotNilf(t, err, "expected error loading '%v'", "routes-static-path-slash-error.conf")
	assert.NotNil(t, router)
	assert.Equal(t, "'static.public.path' [static], path must begin with '/'", err.Error())
}

func TestRouterErrorRoutesPathBeginSlashLoadConfiguration(t *testing.T) {
	router, err := createRouter("routes-path-slash-error.conf")
	assert.NotNilf(t, err, "expected error loading '%v'", "routes-path-slash-error.conf")
	assert.NotNil(t, router)
	assert.Equal(t, "'app_index.path' [login], path must begin with '/'", err.Error())
}

func TestRouterNoDomainRoutesFound(t *testing.T) {
	router, err := createRouter("routes-no-domains.conf")
	assert.Equal(t, ErrNoDomainRoutesConfigFound, err)
	assert.NotNil(t, router)
}

func TestRouterDomainConfig(t *testing.T) {
	router, err := createRouter("routes.conf")
	assert.FailNowOnError(t, err, "")

	domain := router.FindDomain(createHTTPRequest("localhost:8080", ""))
	assert.NotNil(t, domain)

	domain = router.FindDomain(createHTTPRequest("www.aahframework.org", ""))
	assert.Nil(t, domain)
}

func TestRouterDomainAddresses(t *testing.T) {
	router, err := createRouter("routes.conf")
	assert.FailNowOnError(t, err, "")

	addresses := router.DomainAddresses()
	assert.True(t, len(addresses) == 2)
}

func TestRouterRegisteredActions(t *testing.T) {
	router, err := createRouter("routes.conf")
	assert.FailNowOnError(t, err, "")

	methods := router.RegisteredActions()
	assert.NotNil(t, methods)
}

func TestRouterIsDefaultAction(t *testing.T) {
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

func TestRouterDomainAllowed(t *testing.T) {
	router, err := createRouter("routes.conf")
	assert.FailNowOnError(t, err, "")

	req := createHTTPRequest("localhost:8080", "/login")
	domain := router.FindDomain(req)
	allow := domain.Allowed(ahttp.MethodGet, "/login")
	assert.NotNil(t, allow)
	assert.False(t, ess.IsStrEmpty(allow))

	domain = router.FindDomain(req)
	allow = domain.Allowed(ahttp.MethodPost, "*")
	assert.NotNil(t, allow)
	assert.True(t, strings.Contains(allow, ahttp.MethodPost))
	assert.True(t, strings.Contains(allow, ahttp.MethodGet))

	// domain not exists
	reqNotExists := createHTTPRequest("notexists:8080", "/")
	domain = router.FindDomain(reqNotExists)
	assert.Nil(t, domain)
}

func TestRouterDomainReverseURL(t *testing.T) {
	router, err := createRouter("routes.conf")
	assert.FailNowOnError(t, err, "")

	req := createHTTPRequest("localhost:8080", "/")
	domain := router.FindDomain(req)

	// route name not exists
	emptyURL := domain.ReverseURLm("not_exists_routename", map[string]interface{}{})
	assert.Equal(t, "", emptyURL)
	emptyURL = domain.ReverseURL("not_exists_routename")
	assert.Equal(t, "", emptyURL)

	// not enough arguments
	emptyURL = domain.ReverseURLm("book_hotels", map[string]interface{}{})
	assert.Equal(t, "", emptyURL)
	emptyURL = domain.ReverseURL("book_hotels")
	assert.Equal(t, "", emptyURL)

	// incorrect key name scenario
	emptyURL = domain.ReverseURLm("book_hotels", map[string]interface{}{
		"idvalue": "12345678",
	})
	assert.Equal(t, "", emptyURL)

	// index route
	indexURL := domain.ReverseURLm("app_index", map[string]interface{}{})
	assert.Equal(t, "/", indexURL)
	indexURL = domain.ReverseURL("app_index")
	assert.Equal(t, "/", indexURL)

	// static URL
	loginURL := domain.ReverseURLm("login", map[string]interface{}{})
	assert.Equal(t, "/login", loginURL)
	loginURL = domain.ReverseURL("login")
	assert.Equal(t, "/login", loginURL)

	// success scenario
	bookingURL := domain.ReverseURLm("book_hotels", map[string]interface{}{
		"id": "12345678",
	})
	assert.Equal(t, "/hotels/12345678/booking", bookingURL)
	bookingURL = domain.ReverseURL("book_hotels", 12345678)
	assert.Equal(t, "/hotels/12345678/booking", bookingURL)

	bookingURL = domain.ReverseURLm("book_hotels", map[string]interface{}{
		"id":     "12345678",
		"param1": "param1value",
		"param2": "param2value",
	})
	assert.Equal(t, "/hotels/12345678/booking?param1=param1value&param2=param2value", bookingURL)

	bookingURL = domain.ReverseURL("book_hotels", 12345678, "param1value", "param2value")
	assert.Equal(t, "", bookingURL)
}

func TestRouterDomainAddRoute(t *testing.T) {
	domain := &Domain{
		Host:   "aahframework.org",
		trees:  make(map[string]*node),
		routes: make(map[string]*Route),
	}

	route1 := &Route{
		Name:       "route1",
		Path:       "/info/:user/project/:project",
		Method:     "GET",
		Controller: "Info",
		Action:     "ShowProject",
	}
	err := domain.AddRoute(route1)
	assert.FailNowOnError(t, err, "unexpected error")

	route2 := &Route{
		Name:       "index",
		Path:       "/",
		Method:     "GET",
		Controller: "App",
		Action:     "Index",
	}
	err = domain.AddRoute(route2)
	assert.FailNowOnError(t, err, "unexpected error")

	routeError := &Route{
		Name:       "route_error",
		Path:       "/",
		Method:     "GET",
		Controller: "App",
		Action:     "Index",
	}
	err = domain.AddRoute(routeError)
	assert.True(t, strings.Contains(err.Error(), "value is already registered"))
}

func TestRouterConfigNotExists(t *testing.T) {
	router, err := createRouter("routes-not-exists.conf")
	assert.NotNil(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "router: configuration does not exists"))
	assert.Nil(t, router.config)
}

func TestRouterNamespaceConfig(t *testing.T) {
	wd, _ := os.Getwd()
	appCfg, _ := config.ParseString("")
	router := New(filepath.Join(wd, "testdata", "routes-namespace.conf"), appCfg)
	err := router.Load()
	assert.FailNowOnError(t, err, "")

	routes := router.Domains["localhost:8080"].routes
	assert.NotNil(t, routes)
	assert.Equal(t, 4, len(routes))
	assert.Equal(t, "/v1/users", routes["create_user"].Path)
	assert.Equal(t, "POST", routes["create_user"].Method)
	assert.Equal(t, "/v1/users/:id/settings", routes["disable_user"].Path)
	assert.Equal(t, "GET", routes["disable_user"].Method)

	router = New(filepath.Join(wd, "testdata", "routes-namespace-action-error.conf"), appCfg)
	err = router.Load()
	assert.NotNil(t, err)
	assert.Equal(t, "'list_users.action' key is missing, it seems to be multiple HTTP methods", err.Error())
}

func createRouter(filename string) (*Router, error) {
	_ = log.SetLevel("TRACE")
	wd, _ := os.Getwd()
	appCfg, _ := config.ParseString(`routes {
			localhost {
				host = "localhost"
				port = "8080"
			}
		}`)

	router := New(filepath.Join(wd, "testdata", filename), appCfg)
	err := router.Load()
	return router, err
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
