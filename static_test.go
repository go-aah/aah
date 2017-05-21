// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"aahframework.org/config.v0"
	"aahframework.org/router.v0"
	"aahframework.org/test.v0/assert"
)

func TestStaticDirectoryListing(t *testing.T) {
	// Dir Scenario
	r1 := httptest.NewRequest("GET", "http://localhost:8080/assets/..\\..\\/css/", nil)
	w1 := httptest.NewRecorder()
	appCfg, _ := config.ParseString("")
	e := newEngine(appCfg)
	ctx := e.prepareContext(w1, r1)
	ctx.route = &router.Route{
		IsStatic: true,
		Dir:      "assets",
	}
	ctx.Req.Params.Path = map[string]string{
		"filepath": "..\\..\\/css/",
	}

	pathSeparator = '\\'
	err := e.serveStatic(ctx)
	assert.Nil(t, err)
	pathSeparator = filepath.Separator

	// File Scenario
	ctx.route = &router.Route{
		IsStatic: true,
		File:     "testsample.js",
	}

	file, err := getFilepath(ctx)
	assert.Nil(t, err)
	assert.Equal(t, "static/testsample.js", file)
}

func TestStaticMisc(t *testing.T) {
	// File extension check for gzip
	v1 := checkGzipRequired("sample.css")
	assert.True(t, v1)

	v2 := checkGzipRequired("font.otf")
	assert.True(t, v2)

	// directoryList for read error
	r1 := httptest.NewRequest("GET", "http://localhost:8080/assets/css/app.css", nil)
	w1 := httptest.NewRecorder()
	f, err := os.Open(filepath.Join(getTestdataPath(), "static", "test.txt"))
	assert.Nil(t, err)

	directoryList(w1, r1, f)
	assert.Equal(t, "Error reading directory", w1.Body.String())
}
