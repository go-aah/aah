// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ahttp

import (
	"net/http"
	"net/url"
	"testing"

	"aahframework.org/test/assert"
)

func TestParseAcceptHeaderLanguage(t *testing.T) {
	req1 := createRawHTTPRequest(HeaderAcceptLanguage, "en-us;q=0.0,en;q=0.7, da, en-gb;q=0.8")
	specs := ParseAccept(req1, HeaderAcceptLanguage)
	assert.Equal(t, "da", specs.MostQualified().Value)
	assert.Equal(t, 4, len(specs))
	assert.Equal(t, specs[1].Value, "en-gb")
	assert.Equal(t, specs[1].Q, float32(0.8))
	assert.Equal(t, specs[2].Value, "en")
	assert.Equal(t, specs[2].Q, float32(0.7))

	req2 := createRawHTTPRequest(HeaderAcceptLanguage, "en-gb;leve=1;q=0.8, da, en;level=2;q=0.7, en-us;q=gg")
	specs = ParseAccept(req2, HeaderAcceptLanguage)
	assert.Equal(t, "da", specs.MostQualified().Value)
	assert.Equal(t, 4, len(specs))
	assert.Equal(t, specs[1].Value, "en-gb")
	assert.Equal(t, specs[1].Q, float32(0.8))
	assert.Equal(t, specs[2].Value, "en")
	assert.Equal(t, specs[2].Q, float32(0.7))

	req3 := createRawHTTPRequest(HeaderAcceptLanguage, "zh, en-us; q=0.8, en; q=0.6")
	specs = ParseAccept(req3, HeaderAcceptLanguage)
	assert.Equal(t, "zh", specs.MostQualified().Value)
	assert.Equal(t, 3, len(specs))
	assert.Equal(t, specs[1].Value, "en-us")
	assert.Equal(t, specs[1].Q, float32(0.8))
	assert.Equal(t, specs[2].Value, "en")
	assert.Equal(t, specs[2].Q, float32(0.6))

	req4 := createRawHTTPRequest(HeaderAcceptLanguage, "en-gb;q=0.8, da, en;level=2;q=0.7, en-us;leve=1;q=gg")
	specs = ParseAccept(req4, HeaderAcceptLanguage)
	assert.Equal(t, "da", specs.MostQualified().Value)
	assert.Equal(t, 4, len(specs))
	assert.Equal(t, specs[1].Value, "en-gb")
	assert.Equal(t, specs[1].Q, float32(0.8))
	assert.Equal(t, specs[2].Value, "en")
	assert.Equal(t, specs[2].Q, float32(0.7))

	req5 := createRawHTTPRequest(HeaderAcceptLanguage, "zh, en-us; q=wrong, en; q=0.6")
	specs = ParseAccept(req5, HeaderAcceptLanguage)
	assert.Equal(t, "zh", specs.MostQualified().Value)
	assert.Equal(t, 3, len(specs))
	assert.Equal(t, specs[1].Value, "en")
	assert.Equal(t, specs[1].Q, float32(0.6))

	req6 := createRawHTTPRequest(HeaderAcceptLanguage, "es-mx, es, en")
	specs = ParseAccept(req6, HeaderAcceptLanguage)
	assert.Equal(t, "es-mx", specs.MostQualified().Value)
	assert.Equal(t, 3, len(specs))
	assert.Equal(t, specs[1].Value, "es")
	assert.Equal(t, specs[1].Q, float32(1.0))
	assert.Equal(t, specs[2].Value, "en")
	assert.Equal(t, specs[2].Q, float32(1.0))
}

func TestNegotiateLocale(t *testing.T) {
	req1 := createRawHTTPRequest(HeaderAcceptLanguage, "es-mx, es, en")
	locale := NegotiateLocale(req1)
	assert.Equal(t, "es-mx", locale.Raw)
	assert.Equal(t, "es", locale.Language)
	assert.Equal(t, "mx", locale.Region)
	assert.Equal(t, "es-mx", locale.String())

	req2 := createRawHTTPRequest(HeaderAcceptLanguage, "es")
	locale = NegotiateLocale(req2)
	assert.Equal(t, "es", locale.Raw)
	assert.Equal(t, "es", locale.Language)
	assert.Equal(t, "", locale.Region)
	assert.Equal(t, "es", locale.String())

	req3 := createRawHTTPRequest(HeaderAcceptLanguage, "")
	locale = NegotiateLocale(req3)
	assert.Equal(t, true, locale == nil)
}

func TestNegotiateEncoding(t *testing.T) {
	req1 := createRawHTTPRequest(HeaderAcceptEncoding, "compress;q=0.5, gzip;q=1.0")
	encoding := NegotiateEncoding(req1)
	assert.Equal(t, "gzip", encoding.Value)
	assert.Equal(t, "gzip;q=1.0", encoding.Raw)

	req2 := createRawHTTPRequest(HeaderAcceptEncoding, "gzip;q=1.0, identity; q=0.5, *;q=0")
	encoding = NegotiateEncoding(req2)
	assert.Equal(t, "gzip", encoding.Value)
	assert.Equal(t, "gzip;q=1.0", encoding.Raw)

	req3 := createRawHTTPRequest(HeaderAcceptEncoding, "")
	encoding = NegotiateEncoding(req3)
	assert.Equal(t, true, encoding == nil)
}

// func createRequest(hdrKey, value string) *Request {
// 	req := &Request{Raw: createRawHTTPRequest(hdrKey, value)}
// 	req.Path = req.Raw.URL.Path
// 	req.Header = req.Raw.Header
// 	return req
// }

func createRawHTTPRequest(hdrKey, value string) *http.Request {
	hdr := http.Header{}
	hdr.Set(hdrKey, value)
	url, _ := url.Parse("http://localhost:8080/testpath")
	return &http.Request{
		URL:    url,
		Header: hdr,
	}
}
