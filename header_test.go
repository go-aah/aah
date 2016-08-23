// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ahttp

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/go-aah/test/assert"
)

func TestParseAcceptHeaderLanguage(t *testing.T) {
	req1 := createRequest(HeaderAcceptLanguage, "en-us;q=0.0,en;q=0.7, da, en-gb;q=0.8")
	specs := ParseAccept(req1, HeaderAcceptLanguage)
	assert.Equal(t, "da", specs.MostQualified().Value)
	assert.Equal(t, 4, len(specs))
	assert.Equal(t, specs[1].Value, "en-gb")
	assert.Equal(t, specs[1].Q, float32(0.8))
	assert.Equal(t, specs[2].Value, "en")
	assert.Equal(t, specs[2].Q, float32(0.7))

	req2 := createRequest(HeaderAcceptLanguage, "en-gb;leve=1;q=0.8, da, en;level=2;q=0.7, en-us;q=gg")
	specs = ParseAccept(req2, HeaderAcceptLanguage)
	assert.Equal(t, "da", specs.MostQualified().Value)
	assert.Equal(t, 4, len(specs))
	assert.Equal(t, specs[1].Value, "en-gb")
	assert.Equal(t, specs[1].Q, float32(0.8))
	assert.Equal(t, specs[2].Value, "en")
	assert.Equal(t, specs[2].Q, float32(0.7))

	req3 := createRequest(HeaderAcceptLanguage, "zh, en-us; q=0.8, en; q=0.6")
	specs = ParseAccept(req3, HeaderAcceptLanguage)
	assert.Equal(t, "zh", specs.MostQualified().Value)
	assert.Equal(t, 3, len(specs))
	assert.Equal(t, specs[1].Value, "en-us")
	assert.Equal(t, specs[1].Q, float32(0.8))
	assert.Equal(t, specs[2].Value, "en")
	assert.Equal(t, specs[2].Q, float32(0.6))

	req4 := createRequest(HeaderAcceptLanguage, "en-gb;q=0.8, da, en;level=2;q=0.7, en-us;leve=1;q=gg")
	specs = ParseAccept(req4, HeaderAcceptLanguage)
	assert.Equal(t, "da", specs.MostQualified().Value)
	assert.Equal(t, 4, len(specs))
	assert.Equal(t, specs[1].Value, "en-gb")
	assert.Equal(t, specs[1].Q, float32(0.8))
	assert.Equal(t, specs[2].Value, "en")
	assert.Equal(t, specs[2].Q, float32(0.7))

	req5 := createRequest(HeaderAcceptLanguage, "zh, en-us; q=wrong, en; q=0.6")
	specs = ParseAccept(req5, HeaderAcceptLanguage)
	assert.Equal(t, "zh", specs.MostQualified().Value)
	assert.Equal(t, 3, len(specs))
	assert.Equal(t, specs[1].Value, "en")
	assert.Equal(t, specs[1].Q, float32(0.6))

	req6 := createRequest(HeaderAcceptLanguage, "es-mx, es, en")
	specs = ParseAccept(req6, HeaderAcceptLanguage)
	assert.Equal(t, "es-mx", specs.MostQualified().Value)
	assert.Equal(t, 3, len(specs))
	assert.Equal(t, specs[1].Value, "es")
	assert.Equal(t, specs[1].Q, float32(1.0))
	assert.Equal(t, specs[2].Value, "en")
	assert.Equal(t, specs[2].Q, float32(1.0))
}

func TestNegotiateLocale(t *testing.T) {
	locale := NegotiateLocale(createRequest(HeaderAcceptLanguage, "es-mx, es, en"))
	assert.Equal(t, "es-mx", locale.Raw)
	assert.Equal(t, "es", locale.Language)
	assert.Equal(t, "mx", locale.Region)
	assert.Equal(t, "es-mx", locale.String())

	locale = NegotiateLocale(createRequest(HeaderAcceptLanguage, "es"))
	assert.Equal(t, "es", locale.Raw)
	assert.Equal(t, "es", locale.Language)
	assert.Equal(t, "", locale.Region)
	assert.Equal(t, "es", locale.String())

	locale = NegotiateLocale(createRequest(HeaderAcceptLanguage, ""))
	assert.Equal(t, true, locale == nil)
}

func TestNegotiateContentType(t *testing.T) {
	contentType := NegotiateContentType(createRequest(HeaderAccept, "audio/*; q=0.2, audio/basic"))
	assert.Equal(t, "audio/basic", contentType.String())
	assert.Equal(t, "audio/basic", contentType.Mime)
	assert.Equal(t, "", contentType.Version())

	contentType = NegotiateContentType(createRequest(HeaderAccept, "application/json;version=2"))
	assert.Equal(t, "application/json;version=2", contentType.String())
	assert.Equal(t, "application/json", contentType.Mime)
	assert.Equal(t, "2", contentType.Version())

	contentType = NegotiateContentType(createRequest(HeaderAccept, "text/plain; q=0.5, text/html, text/x-dvi; q=0.8, text/x-c"))
	assert.Equal(t, "text/html", contentType.String())
	assert.Equal(t, "text/html", contentType.Mime)
	assert.Equal(t, "", contentType.Version())

	contentType = NegotiateContentType(createRequest(HeaderAccept, ""))
	assert.Equal(t, "text/html; charset=utf-8", contentType.String())
	assert.Equal(t, "text/html", contentType.Mime)
	assert.Equal(t, ".html", contentType.Ext)
	assert.Equal(t, "", contentType.Version())

	req := createRequest(HeaderAccept, "application/json")
	req.URL, _ = url.Parse("http://localhost:8080/testpath.json")
	contentType = NegotiateContentType(req)
	assert.Equal(t, "", contentType.Mime)
	assert.Equal(t, ".json", contentType.Ext)

	req = createRequest(HeaderAccept, "application/json; version=2")
	spec := ParseAccept(req, HeaderAccept).MostQualified()
	assert.Equal(t, "2", spec.GetParam("version", "1"))

	req = createRequest(HeaderAccept, "application/json")
	spec = ParseAccept(req, HeaderAccept).MostQualified()
	assert.Equal(t, "1", spec.GetParam("version", "1"))

	req = createRequest(HeaderAccept, "application/json; version")
	spec = ParseAccept(req, HeaderAccept).MostQualified()
	assert.Equal(t, "", spec.GetParam("version", "1"))
}

func TestNegotiateEncoding(t *testing.T) {
	encoding := NegotiateEncoding(createRequest(HeaderAcceptEncoding, "compress;q=0.5, gzip;q=1.0"))
	assert.Equal(t, "gzip", encoding.Value)
	assert.Equal(t, "gzip;q=1.0", encoding.Raw)

	encoding = NegotiateEncoding(createRequest(HeaderAcceptEncoding, "gzip;q=1.0, identity; q=0.5, *;q=0"))
	assert.Equal(t, "gzip", encoding.Value)
	assert.Equal(t, "gzip;q=1.0", encoding.Raw)

	encoding = NegotiateEncoding(createRequest(HeaderAcceptEncoding, ""))
	assert.Equal(t, true, encoding == nil)
}

func TestParseContentType(t *testing.T) {
	contentType := ParseContentType(createRequest(HeaderContentType, "text/html; charset=utf-8"))
	assert.Equal(t, "text/html", contentType.Mime)
	assert.Equal(t, "text/html; charset=utf-8", contentType.String())
	assert.Equal(t, "utf-8", contentType.Charset("iso-8859-1"))

	contentType = ParseContentType(createRequest(HeaderContentType, "text/html"))
	assert.Equal(t, "text/html", contentType.Mime)
	assert.Equal(t, "text/html", contentType.String())
	assert.Equal(t, "iso-8859-1", contentType.Charset("iso-8859-1"))

	contentType = ParseContentType(createRequest(HeaderContentType, "application/json"))
	assert.Equal(t, "application/json", contentType.Mime)
	assert.Equal(t, "application/json", contentType.String())
	assert.Equal(t, "", contentType.Charset(""))

	contentType = ParseContentType(createRequest(HeaderContentType, ""))
	assert.Equal(t, "text/html", contentType.Mime)
	assert.Equal(t, "text/html; charset=utf-8", contentType.String())
	assert.Equal(t, ".html", contentType.Ext)

	contentType = ParseContentType(createRequest(HeaderContentType, "text/html;charset"))
	assert.Equal(t, "text/html", contentType.Mime)
	assert.Equal(t, "text/html;charset", contentType.String())
	assert.Equal(t, "", contentType.Charset("iso-8859-1"))
}

func createRequest(hdrKey, value string) *http.Request {
	hdr := http.Header{}
	hdr.Set(hdrKey, value)
	url, _ := url.Parse("http://localhost:8080/testpath")
	return &http.Request{
		URL:    url,
		Header: hdr,
	}
}
