// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"path/filepath"
	"testing"

	"aahframework.org/config.v0"
	"aahframework.org/router.v0-unstable"
	"aahframework.org/test.v0/assert"
)

func TestRouterTemplateFuncs(t *testing.T) {
	appCfg, _ := config.ParseString("")
	cfgDir := filepath.Join(getTestdataPath(), appConfigDir())
	err := initRoutes(cfgDir, appCfg)
	assert.Nil(t, err)
	assert.NotNil(t, AppRouter())

	ctx := &Context{
		Req: getAahRequest("GET", "http://localhost:8080/doc/v0.3/mydoc.html", ""),
	}

	viewArgs := map[string]interface{}{}
	viewArgs["Host"] = "localhost:8080"

	url1 := tmplURL(viewArgs, "version_home#welcome", "v0.1")
	assert.Equal(t, "//localhost:8080/doc/v0.1#welcome", string(url1))

	url2 := tmplURLm(viewArgs, "show_doc", map[string]interface{}{
		"version": "v0.2",
		"content": "getting-started.html",
	})
	assert.Equal(t, "//localhost:8080/doc/v0.2/getting-started.html", string(url2))

	url3 := tmplURL(viewArgs)
	assert.Equal(t, "#", string(url3))

	url4 := tmplURL(viewArgs, "host")
	assert.Equal(t, "//localhost:8080", string(url4))

	ctx.Reset()
}

func TestRouterMisc(t *testing.T) {
	domain := &router.Domain{Host: "localhost"}
	result := composeRouteURL(domain, "/path", "my-head")
	assert.Equal(t, "//localhost/path#my-head", result)
}
