// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"aahframework.org/ahttp.v0"
	"aahframework.org/test.v0/assert"
)

func TestStaticFilesDelivery(t *testing.T) {
	importPath := filepath.Join(testdataBaseDir(), "webapp1")
	ts, err := newTestServer(t, importPath)
	assert.Nil(t, err)
	defer ts.Close()

	t.Logf("Test Server URL [Static Files Delivery]: %s", ts.URL)

	httpClient := new(http.Client)

	// Static File - /robots.txt
	t.Log("Static File - /robots.txt")
	resp, err := httpClient.Get(ts.URL + "/robots.txt")
	assert.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.True(t, strings.Contains(responseBody(resp), "User-agent: *"))
	assert.Equal(t, "no-cache, no-store, must-revalidate", resp.Header.Get(ahttp.HeaderCacheControl))

	// Static File - /assets/css/aah.css
	t.Log("Static File - /assets/css/aah.css")
	resp, err = httpClient.Get(ts.URL + "/assets/css/aah.css")
	assert.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.True(t, strings.Contains(responseBody(resp), "Minimal aah framework application template CSS."))
	assert.Equal(t, "no-cache, no-store, must-revalidate", resp.Header.Get(ahttp.HeaderCacheControl))

	// Directory Listing - /assets
	t.Log("Directory Listing - /assets")
	resp, err = httpClient.Get(ts.URL + "/assets")
	assert.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	body := responseBody(resp)
	assert.True(t, strings.Contains(body, "<title>Listing of /assets/</title>"))
	assert.True(t, strings.Contains(body, "<h1>Listing of /assets/</h1><hr>"))
	assert.True(t, strings.Contains(body, `<a href="robots.txt">robots.txt</a>`))
	assert.Equal(t, "", resp.Header.Get(ahttp.HeaderCacheControl))

	// Static File - /assets/img/aah-framework-logo.png
	t.Log("Static File - /assets/img/aah-framework-logo.png")
	resp, err = httpClient.Get(ts.URL + "/assets/img/aah-framework-logo.png")
	assert.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "image/png", resp.Header.Get(ahttp.HeaderContentType))
	assert.Equal(t, "6990", resp.Header.Get(ahttp.HeaderContentLength))
	assert.Equal(t, "no-cache, no-store, must-revalidate", resp.Header.Get(ahttp.HeaderCacheControl))

	// Static File - /assets/img/notfound/file.txt
	t.Log("Static File - /assets/img/notfound/file.txt")
	resp, err = httpClient.Get(ts.URL + "/assets/img/notfound/file.txt")
	assert.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "0", resp.Header.Get(ahttp.HeaderContentLength))
}

func TestStaticDetectContentType(t *testing.T) {
	v, _ := detectFileContentType("image1.svg", nil)
	assert.Equal(t, "image/svg+xml", v)

	v, _ = detectFileContentType("image2.png", nil)
	assert.Equal(t, "image/png", v)

	v, _ = detectFileContentType("image3.jpg", nil)
	assert.Equal(t, "image/jpeg", v)

	v, _ = detectFileContentType("image4.jpeg", nil)
	assert.Equal(t, "image/jpeg", v)

	v, _ = detectFileContentType("file.pdf", nil)
	assert.Equal(t, "application/pdf", v)

	v, _ = detectFileContentType("file.js", nil)
	assert.Equal(t, "application/javascript", v)

	v, _ = detectFileContentType("file.txt", nil)
	assert.Equal(t, "text/plain; charset=utf-8", v)

	v, _ = detectFileContentType("file.html", nil)
	assert.Equal(t, "text/html; charset=utf-8", v)

	v, _ = detectFileContentType("file.xml", nil)
	assert.Equal(t, "application/xml", v)

	v, _ = detectFileContentType("file.json", nil)
	assert.Equal(t, "application/json", v)

	v, _ = detectFileContentType("file.css", nil)
	assert.Equal(t, "text/css; charset=utf-8", v)

	content, _ := ioutil.ReadFile(filepath.Join(testdataBaseDir(), "test-image.noext"))
	v, _ = detectFileContentType("test-image.noext", bytes.NewReader(content))
	assert.Equal(t, "image/png", v)
}
