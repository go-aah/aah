// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aahframework.org/config.v0"
	"aahframework.org/router.v0"
	"aahframework.org/test.v0/assert"
)

func TestStaticDirectoryListing(t *testing.T) {
	appCfg, _ := config.ParseString("")
	e := newEngine(appCfg)

	testStaticServe(t, e, "http://localhost:8080/static/css/aah\x00.css", "static", "css/aah\x00.css", "", "500 Internal Server Error", false)

	testStaticServe(t, e, "http://localhost:8080/static/test.txt", "static", "test.txt", "", "This is file content of test.txt", false)

	testStaticServe(t, e, "http://localhost:8080/static", "static", "", "", "403 Directory listing not allowed", false)

	testStaticServe(t, e, "http://localhost:8080/static", "static", "", "", `<a href="/static/">Found</a>`, true)

	testStaticServe(t, e, "http://localhost:8080/static/", "static", "", "", `<title>Listing of /static/</title>`, true)

	testStaticServe(t, e, "http://localhost:8080/robots.txt", "", "", "test.txt", "This is file content of test.txt", false)
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

func testStaticServe(t *testing.T, e *engine, reqURL, dir, filePath, file, result string, listDir bool) {
	r := httptest.NewRequest("GET", reqURL, nil)
	w := httptest.NewRecorder()
	ctx := e.prepareContext(w, r)
	ctx.route = &router.Route{IsStatic: true, Dir: dir, ListDir: listDir, File: file}
	ctx.Req.Params.Path = map[string]string{
		"filepath": filePath,
	}
	appBaseDir = getTestdataPath()
	err := e.serveStatic(ctx)
	appBaseDir = ""
	assert.Nil(t, err)
	assert.True(t, strings.Contains(w.Body.String(), result))
}
