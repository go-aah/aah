// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aahframe.work/aah/ahttp"
	"github.com/stretchr/testify/assert"
)

func TestStaticFilesDelivery(t *testing.T) {
	importPath := filepath.Join(testdataBaseDir(), "webapp1")
	ts := newTestServer(t, importPath)
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
	testcases := []struct {
		label    string
		filename string
		result   string
	}{
		{
			label:    "svg",
			filename: "image1.svg",
			result:   "image/svg+xml",
		},
		{
			label:    "png",
			filename: "image2.png",
			result:   "image/png",
		},
		{
			label:    "jpg",
			filename: "image3.jpg",
			result:   "image/jpeg",
		},
		{
			label:    "jpeg",
			filename: "image4.jpeg",
			result:   "image/jpeg",
		},
		{
			label:    "pdf",
			filename: "file.pdf",
			result:   "application/pdf",
		},
		{
			label:    "javascript",
			filename: "file.js",
			result:   "application/javascript",
		},
		{
			label:    "txt",
			filename: "file.txt",
			result:   "text/plain; charset=utf-8",
		},
		{
			label:    "xml",
			filename: "file.xml",
			result:   "application/xml",
		},
		{
			label:    "css",
			filename: "file.css",
			result:   "text/css; charset=utf-8",
		},
		{
			label:    "html",
			filename: "file.html",
			result:   "text/html; charset=utf-8",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.label, func(t *testing.T) {
			v, _ := detectFileContentType(tc.filename, nil)
			assert.Equal(t, tc.result, v)
		})
	}

	content, _ := ioutil.ReadFile(filepath.Join(testdataBaseDir(), "test-image.noext"))
	v, _ := detectFileContentType("test-image.noext", bytes.NewReader(content))
	assert.Equal(t, "image/png", v)
}

func TestStaticCacheHeader(t *testing.T) {
	sm := staticManager{
		mimeCacheHdrMap: map[string]string{
			"text/css":               "public, max-age=604800, proxy-revalidate",
			"application/javascript": "public, max-age=604800, proxy-revalidate",
			"image/png":              "public, max-age=604800, proxy-revalidate",
		},
		defaultCacheHdr: "public, max-age=31536000",
	}

	str := sm.cacheHeader("application/json")
	assert.Equal(t, "public, max-age=31536000", str)

	str = sm.cacheHeader("image/png")
	assert.Equal(t, "public, max-age=604800, proxy-revalidate", str)

	str = sm.cacheHeader("application/json; charset=utf-8")
	assert.Equal(t, "public, max-age=31536000", str)

	str = sm.cacheHeader("text/css")
	assert.Equal(t, "public, max-age=604800, proxy-revalidate", str)
}

func TestStaticWriteFileError(t *testing.T) {
	importPath := filepath.Join(testdataBaseDir(), "webapp1")
	ts := newTestServer(t, importPath)
	defer ts.Close()

	t.Logf("Test Server URL [Static Write File Error]: %s", ts.URL)

	sm := ts.app.staticMgr
	req := httptest.NewRequest(ahttp.MethodGet, "http://localhost:8080/assets/js/myfile.js", nil)

	w1 := httptest.NewRecorder()
	sm.writeError(ahttp.AcquireResponseWriter(w1), ahttp.AcquireRequest(req), os.ErrPermission)
	assert.Equal(t, "403 Forbidden", responseBody(w1.Result()))

	w2 := httptest.NewRecorder()
	sm.writeError(ahttp.AcquireResponseWriter(w2), ahttp.AcquireRequest(req), nil)
	assert.Equal(t, "500 Internal Server Error", responseBody(w2.Result()))
}
