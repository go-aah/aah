// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"bytes"
	"fmt"
	"io/ioutil"
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

	testStaticServe(t, e, "http://localhost:8080/static", "static", "", "", `<a href="/static/">Moved Permanently</a>`, true)

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

func TestParseStaticCacheMap(t *testing.T) {
	appConfig, _ = config.ParseString(`
		cache {
		  static {
		    default_cache_control = "public, max-age=31536000"

				mime_types {
		      css_js {
		        mime = "text/css, application/javascript"
		        cache_control = "public, max-age=2628000, must-revalidate, proxy-revalidate"
		      }

		      images {
		        mime = "image/jpeg, image/png, image/gif, image/svg+xml, image/x-icon"
		        cache_control = "public, max-age=2628000, must-revalidate, proxy-revalidate"
		      }
		    }
		  }
		}
	`)

	parseStaticMimeCacheMap(&Event{})
	assert.Equal(t, "public, max-age=2628000, must-revalidate, proxy-revalidate", cacheHeader("image/png"))
	assert.Equal(t, "public, max-age=31536000", cacheHeader("application/x-font-ttf"))
	appConfig = nil
}

func TestStaticDetectContentType(t *testing.T) {
	v, _ := detectStaticContentType("image1.svg", nil)
	assert.Equal(t, "image/svg+xml", v)

	v, _ = detectStaticContentType("image2.png", nil)
	assert.Equal(t, "image/png", v)

	v, _ = detectStaticContentType("image3.jpg", nil)
	assert.Equal(t, "image/jpeg", v)

	v, _ = detectStaticContentType("image4.jpeg", nil)
	assert.Equal(t, "image/jpeg", v)

	v, _ = detectStaticContentType("file.pdf", nil)
	assert.Equal(t, "application/pdf", v)

	v, _ = detectStaticContentType("file.js", nil)
	assert.Equal(t, "application/javascript", v)

	v, _ = detectStaticContentType("file.txt", nil)
	assert.Equal(t, "text/plain; charset=utf-8", v)

	v, _ = detectStaticContentType("file.html", nil)
	assert.Equal(t, "text/html; charset=utf-8", v)

	v, _ = detectStaticContentType("file.xml", nil)
	assert.Equal(t, "application/xml", v)

	v, _ = detectStaticContentType("file.json", nil)
	assert.Equal(t, "application/json", v)

	v, _ = detectStaticContentType("file.css", nil)
	assert.Equal(t, "text/css; charset=utf-8", v)

	content, _ := ioutil.ReadFile(filepath.Join(getTestdataPath(), "test-image.noext"))
	v, _ = detectStaticContentType("test-image.noext", bytes.NewReader(content))
	assert.Equal(t, "image/png", v)
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
	if !strings.Contains(w.Body.String(), result) {
		fmt.Println(w.Body.String(), result)
	}

	assert.True(t, strings.Contains(w.Body.String(), result))
}
