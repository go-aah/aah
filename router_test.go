// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"aahframework.org/ahttp.v0"
	"aahframework.org/router.v0"
	"aahframework.org/test.v0/assert"
)

func TestRouterTemplateFuncs(t *testing.T) {
	importPath := filepath.Join(testdataBaseDir(), "webapp1")
	ts, err := newTestServer(t, importPath)
	assert.Nil(t, err)
	defer ts.Close()

	t.Logf("Test Server URL [Router Template funcs]: %s", ts.URL)

	err = ts.app.initRouter()
	assert.Nil(t, err)

	err = ts.app.initView()
	assert.Nil(t, err)

	vm := ts.app.viewMgr

	viewArgs := map[string]interface{}{}
	viewArgs["Host"] = "localhost:8080"

	url1 := vm.tmplURL(viewArgs, "version_home#welcome", "v0.1")
	assert.Equal(t, "//localhost:8080/doc/v0.1#welcome", string(url1))

	url2 := vm.tmplURLm(viewArgs, "show_doc", map[string]interface{}{
		"version": "v0.2",
		"content": "getting-started.html",
	})
	assert.Equal(t, "//localhost:8080/doc/v0.2/getting-started.html", string(url2))

	url3 := vm.tmplURL(viewArgs)
	assert.Equal(t, "#", string(url3))

	url4 := vm.tmplURL(viewArgs, "host")
	assert.Equal(t, "//localhost:8080", string(url4))
}

func TestRouterMisc(t *testing.T) {
	domain := &router.Domain{Host: "localhost"}
	result := composeRouteURL(domain, "/path", "my-head")
	assert.Equal(t, "//localhost/path#my-head", result)
}

func TestRouterCORS(t *testing.T) {
	importPath := filepath.Join(testdataBaseDir(), "webapp1")
	ts, err := newTestServer(t, importPath)
	assert.Nil(t, err)
	defer ts.Close()

	t.Logf("Test Server URL [CORS]: %s", ts.URL)

	// CORS NOT enabled
	t.Log("CORS NOT enabled")
	ctx1 := &Context{
		domain: &router.Domain{},
	}
	CORSMiddleware(ctx1, &Middleware{})

	// CORS preflight request
	t.Log("CORS preflight request")
	req3, err := http.NewRequest(ahttp.MethodOptions, ts.URL+"/users/edit", nil)
	assert.Nil(t, err)
	req3.Header.Set(ahttp.HeaderAccessControlRequestMethod, ahttp.MethodPost)
	req3.Header.Set(ahttp.HeaderOrigin, "http://sample.com")
	ctx3 := newContext(httptest.NewRecorder(), req3)
	ctx3.a = ts.app
	ctx3.domain = &router.Domain{CORSEnabled: true}
	ctx3.route = &router.Route{
		CORS: &router.CORS{
			AllowOrigins:     []string{"http://sample.com"},
			AllowMethods:     []string{ahttp.MethodGet, ahttp.MethodPost, ahttp.MethodOptions},
			AllowCredentials: true,
			MaxAge:           "806400",
		},
	}
	CORSMiddleware(ctx3, &Middleware{})

	// CORS regular request
	t.Log("CORS regular request")
	req4, err := http.NewRequest(ahttp.MethodOptions, ts.URL+"/users/edit", nil)
	assert.Nil(t, err)
	req4.Header.Set(ahttp.HeaderOrigin, "http://sample.com")
	ctx4 := newContext(httptest.NewRecorder(), req4)
	ctx4.a = ts.app
	ctx4.domain = &router.Domain{CORSEnabled: true}
	ctx4.route = &router.Route{
		CORS: &router.CORS{
			AllowOrigins:     []string{"http://sample.com"},
			AllowMethods:     []string{ahttp.MethodGet, ahttp.MethodPost, ahttp.MethodOptions},
			ExposeHeaders:    []string{ahttp.HeaderXRequestedWith},
			AllowCredentials: true,
		},
	}
	CORSMiddleware(ctx4, &Middleware{})

	// Preflight invalid origin
	t.Log("Preflight invalid origin")
	req5, err := http.NewRequest(ahttp.MethodOptions, ts.URL+"/users/edit", nil)
	assert.Nil(t, err)
	req5.Header.Set(ahttp.HeaderAccessControlRequestMethod, ahttp.MethodPost)
	req5.Header.Set(ahttp.HeaderOrigin, "http://example.com")
	ctx5 := newContext(httptest.NewRecorder(), req5)
	ctx5.a = ts.app
	ctx5.domain = &router.Domain{CORSEnabled: true}
	ctx5.route = &router.Route{
		CORS: &router.CORS{
			AllowOrigins:     []string{"http://sample.com"},
			AllowMethods:     []string{ahttp.MethodGet, ahttp.MethodPost, ahttp.MethodOptions},
			AllowCredentials: true,
			MaxAge:           "806400",
		},
	}
	CORSMiddleware(ctx5, &Middleware{})
}
