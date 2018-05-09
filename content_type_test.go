// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/ahttp source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ahttp

import (
	"net/url"
	"runtime"
	"testing"

	"aahframework.org/essentials.v0"
	"aahframework.org/test.v0/assert"
)

func TestHTTPNegotiateContentType(t *testing.T) {
	req1 := createRawHTTPRequest(HeaderAccept, "audio/*; q=0.2, audio/basic")
	contentType := NegotiateContentType(req1)
	assert.Equal(t, "audio/basic", contentType.String())
	assert.True(t, contentType.IsEqual("audio/basic"))
	assert.Equal(t, "", contentType.Version())

	req2 := createRawHTTPRequest(HeaderAccept, "application/json;version=2")
	contentType = NegotiateContentType(req2)
	assert.Equal(t, "application/json; version=2", contentType.String())
	assert.True(t, contentType.IsEqual("application/json"))
	assert.Equal(t, "2", contentType.Version())

	req3 := createRawHTTPRequest(HeaderAccept, "text/plain; q=0.5, text/html, text/x-dvi; q=0.8, text/x-c")
	contentType = NegotiateContentType(req3)
	assert.Equal(t, "text/html", contentType.String())
	assert.True(t, contentType.IsEqual("text/html"))
	assert.Equal(t, "", contentType.Version())

	req4 := createRawHTTPRequest(HeaderAccept, "")
	contentType = NegotiateContentType(req4)
	assert.Equal(t, "text/html; charset=utf-8", contentType.String())
	assert.True(t, contentType.IsEqual("text/html"))
	assert.True(t, ess.IsSliceContainsString(contentType.Exts, ".html"))
	assert.Equal(t, "", contentType.Version())

	req := createRawHTTPRequest(HeaderAccept, "application/json")
	req.URL, _ = url.Parse("http://localhost:8080/testpath.json")
	contentType = NegotiateContentType(req)
	if runtime.GOOS != "windows" { // due to mime types not exists
		assert.NotNil(t, contentType)
		assert.True(t, contentType.IsEqual("application/json"))
		assert.Equal(t, ".json", contentType.Exts[0])
	}

	req = createRawHTTPRequest(HeaderAccept, "application/json")
	req.URL, _ = url.Parse("http://localhost:8080/testpath.html")
	contentType = NegotiateContentType(req)
	assert.Equal(t, "text/html", contentType.Mime)
	assert.True(t, ess.IsSliceContainsString(contentType.Exts, ".html"))

	req = createRawHTTPRequest(HeaderAccept, "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	contentType = NegotiateContentType(req)
	assert.Equal(t, "text/html", contentType.String())
	assert.True(t, contentType.IsEqual("text/html"))
	assert.Equal(t, "", contentType.Version())

	// ParseAccept
	req = createRawHTTPRequest(HeaderAccept, "application/json; version=2")
	spec := ParseAccept(req, HeaderAccept).MostQualified()
	assert.Equal(t, "2", spec.GetParam("version", "1"))

	req = createRawHTTPRequest(HeaderAccept, "application/json")
	spec = ParseAccept(req, HeaderAccept).MostQualified()
	assert.Equal(t, "1", spec.GetParam("version", "1"))

	req = createRawHTTPRequest(HeaderAccept, "application/json; version")
	spec = ParseAccept(req, HeaderAccept).MostQualified()
	assert.Equal(t, "", spec.GetParam("version", "1"))
}

func TestHTTPParseContentType(t *testing.T) {
	req1 := createRawHTTPRequest(HeaderContentType, "text/html; charset=utf-8")
	contentType := ParseContentType(req1)
	assert.Equal(t, "text/html", contentType.Mime)
	assert.Equal(t, "text/html; charset=utf-8", contentType.String())
	assert.Equal(t, "utf-8", contentType.Charset("iso-8859-1"))

	req2 := createRawHTTPRequest(HeaderContentType, "text/html")
	contentType = ParseContentType(req2)
	assert.Equal(t, "text/html", contentType.Mime)
	assert.Equal(t, "text/html", contentType.String())
	assert.Equal(t, "iso-8859-1", contentType.Charset("iso-8859-1"))

	req3 := createRawHTTPRequest(HeaderContentType, "application/json")
	contentType = ParseContentType(req3)
	assert.Equal(t, "application/json", contentType.Mime)
	assert.Equal(t, "application/json", contentType.String())
	assert.Equal(t, "", contentType.Charset(""))

	req4 := createRawHTTPRequest(HeaderContentType, "")
	contentType = ParseContentType(req4)
	assert.Equal(t, "text/html", contentType.Mime)
	assert.Equal(t, "text/html; charset=utf-8", contentType.String())
	assert.True(t, ess.IsSliceContainsString(contentType.Exts, ".html"))

	req5 := createRawHTTPRequest(HeaderContentType, "text/html;charset")
	contentType = ParseContentType(req5)
	assert.True(t, (contentType.Mime == "" || contentType.Mime == "text/html"))
	assert.True(t, (contentType.String() == "" || contentType.String() == "text/html"))
	assert.Equal(t, "iso-8859-1", contentType.Charset("iso-8859-1"))
}
