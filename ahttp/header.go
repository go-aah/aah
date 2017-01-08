// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ahttp

import (
	"mime"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"aahframework.org/essentials"
)

// HTTP Header names
const (
	HeaderAccept              = "Accept"
	HeaderAcceptEncoding      = "Accept-Encoding"
	HeaderAcceptLanguage      = "Accept-Language"
	HeaderAllow               = "Allow"
	HeaderAuthorization       = "Authorization"
	HeaderCacheControl        = "Cache-Control"
	HeaderConnection          = "Connection"
	HeaderContentDisposition  = "Content-Disposition"
	HeaderContentLength       = "Content-Length"
	HeaderContentType         = "Content-Type"
	HeaderCookie              = "Cookie"
	HeaderHost                = "Host"
	HeaderIfModifiedSince     = "If-Modified-Since"
	HeaderLocation            = "Location"
	HeaderLastModified        = "Last-Modified"
	HeaderMethod              = "Method"
	HeaderReferer             = "Referer"
	HeaderServer              = "Server"
	HeaderSetCookie           = "Set-Cookie"
	HeaderStatus              = "Status"
	HeaderOrigin              = "Origin"
	HeaderTransferEncoding    = "Transfer-Encoding"
	HeaderUpgrade             = "Upgrade"
	HeaderUserAgent           = "User-Agent"
	HeaderVary                = "Vary"
	HeaderWWWAuthenticate     = "WWW-Authenticate"
	HeaderXContentTypeOptions = "X-Content-Type-Options"
	HeaderXForwardedFor       = "X-Forwarded-For"
	HeaderXForwardedHost      = "X-Forwarded-Host"
	HeaderXForwardedPort      = "X-Forwarded-Port"
	HeaderXForwardedProto     = "X-Forwarded-Proto"
	HeaderXForwardedServer    = "X-Forwarded-Server"
	HeaderXFrameOptions       = "X-Frame-Options"
	HeaderXHTTPMethodOverride = "X-HTTP-Method-Override"
	HeaderXRealIP             = "X-Real-Ip"
	HeaderXXSSProtection      = "X-XSS-Protection"
)

type (
	// AcceptSpec used for HTTP Accept, Accept-Language, Accept-Encoding header
	// value and it's quality. Implementation follows the specification of RFC7231
	// https://tools.ietf.org/html/rfc7231#section-5.3
	AcceptSpec struct {
		Raw    string
		Value  string
		Q      float32
		Params map[string]string
	}

	// AcceptSpecs is list of values parsed from header and sorted by
	// quality factor.
	AcceptSpecs []AcceptSpec
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// HTTP header methods
//___________________________________

// NegotiateContentType negotiates the response `Content-Type` from the given HTTP
// `Accept` header. The resolve order is- 1) URL extension 2) Accept header
// Most quailfied one based quality factor otherwise default is HTML.
func NegotiateContentType(req *http.Request) *ContentType {
	// 1) URL extension
	ext := filepath.Ext(req.URL.Path)
	switch ext {
	case ".html", ".htm", ".json", ".js", ".xml", ".txt":
		mimeType := mime.TypeByExtension(ext)
		raw := mimeType
		return &ContentType{
			Raw:    raw,
			Mime:   mimeType,
			Exts:   []string{ext},
			Params: make(map[string]string),
		}
	}

	// 2) From Accept header
	spec := ParseAccept(req, HeaderAccept).MostQualified()
	if spec == nil {
		return htmlContentType()
	}

	exts, _ := mime.ExtensionsByType(spec.Value)

	return &ContentType{
		Raw:    spec.Raw,
		Mime:   spec.Value,
		Exts:   exts,
		Params: spec.Params,
	}
}

// NegotiateLocale negotiates the `Accept-Language` from the given HTTP
// request. Most quailfied one based on quality factor.
func NegotiateLocale(req *http.Request) *Locale {
	return ToLocale(ParseAccept(req, HeaderAcceptLanguage).MostQualified())
}

// NegotiateEncoding negotiates the `Accept-Encoding` from the given HTTP
// request. Most quailfied one based on quality factor.
func NegotiateEncoding(req *http.Request) *AcceptSpec {
	return ParseAccept(req, HeaderAcceptEncoding).MostQualified()
}

// ParseContentType resolves the request `Content-Type` from the given HTTP
// request via header `Content-Type`. Partial implementation of
// https://tools.ietf.org/html/rfc1521#section-4 i.e. parsing only
// type, subtype, parameters based on RFC.
func ParseContentType(req *http.Request) *ContentType {
	contentType := req.Header.Get(HeaderContentType)

	if ess.IsStrEmpty(contentType) {
		return htmlContentType()
	}

	values := strings.Split(strings.ToLower(contentType), ";")
	ctype := values[0]
	params := map[string]string{}
	for _, v := range values[1:] {
		pv := strings.Split(v, "=")
		if len(pv) == 2 {
			params[strings.TrimSpace(pv[0])] = strings.TrimSpace(pv[1])
		} else {
			params[strings.TrimSpace(pv[0])] = ""
		}
	}

	exts, _ := mime.ExtensionsByType(ctype)

	return &ContentType{
		Raw:    contentType,
		Mime:   ctype,
		Exts:   exts,
		Params: params,
	}
}

// ParseAccept parses the HTTP Accept* headers from `http.Request`
// returns the specification with quality factor as per RFC7231
// https://tools.ietf.org/html/rfc7231#section-5.3. Level value is not honored.
//
// Good read - http://stackoverflow.com/a/5331486/1343356 and
// http://stackoverflow.com/questions/13890996/http-accept-level
//
// Known issues with WebKit and IE
// http://www.newmediacampaigns.com/blog/browser-rest-http-accept-headers
func ParseAccept(req *http.Request, hdrKey string) AcceptSpecs {
	hdrValue := req.Header.Get(hdrKey)
	var specs AcceptSpecs

	for _, hv := range strings.Split(hdrValue, ",") {
		if ess.IsStrEmpty(hv) {
			continue
		}

		hv = strings.TrimSpace(hv)
		parts := strings.Split(hv, ";")
		if len(parts) == 1 {
			specs = append(specs, AcceptSpec{
				Raw:   hv,
				Value: parts[0],
				Q:     float32(1.0),
			})
			continue
		}

		q := float32(1.0)
		params := map[string]string{}
		for _, pv := range parts[1:] {
			paramParts := strings.Split(strings.TrimSpace(pv), "=")
			if len(paramParts) == 1 {
				params[paramParts[0]] = ""
				continue
			}

			if paramParts[0] == "q" {
				qv, err := strconv.ParseFloat(paramParts[1], 32)
				if err != nil {
					q = float32(0.0)
					params[paramParts[0]] = "0.0"
					continue
				}
				q = float32(qv)
			}

			params[paramParts[0]] = paramParts[1]
		}

		specs = append(specs, AcceptSpec{
			Raw:    hv,
			Value:  parts[0],
			Q:      q,
			Params: params,
		})
	}

	sort.Sort(specs)

	return specs
}

// ToLocale creates a locale instance from `AcceptSpec`
func ToLocale(a *AcceptSpec) *Locale {
	if a == nil {
		return nil
	}

	values := strings.SplitN(a.Value, "-", 2)
	if len(values) == 2 {
		return &Locale{
			Raw:      a.Raw,
			Language: values[0],
			Region:   values[1],
		}
	}

	return &Locale{
		Raw:      a.Raw,
		Language: values[0],
	}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Locale methods
//___________________________________

// String is stringer interface.
func (l *Locale) String() string {
	return l.Raw
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Content-Type methods
//___________________________________

// Charset returns charset of content-type otherwise `defaultCharset` is returned
// 	For e.g.:
// 		Content-Type: application/json; charset=utf-8
//
// 		Method returns `utf-8`
func (c *ContentType) Charset(defaultCharset string) string {
	if v, ok := c.Params["charset"]; ok {
		return v
	}
	return defaultCharset
}

// Version returns Accept header version paramater value if present otherwise
// empty string
// 	For e.g.:
// 		Accept: application/json; version=2
//
// 		Method returns `2`
func (c *ContentType) Version() string {
	if v, ok := c.Params["version"]; ok {
		return v
	}
	return ""
}

// String is stringer interface
func (c *ContentType) String() string {
	return c.Raw
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// AcceptSpecs methods
//___________________________________

// GetParam returns the Accept* header param value otherwise returns default
// value.
// 	For e.g.:
// 		Accept: application/json; version=2
//
// 		Method returns `2` for key `version`
func (a AcceptSpec) GetParam(key string, defaultValue string) string {
	if v, ok := a.Params[key]; ok {
		return v
	}
	return defaultValue
}

// MostQualified returns the most quailfied accept spec, since `AcceptSpec` is
// sorted by quaity factor. First position is the most quailfied otherwise `nil`.
func (specs AcceptSpecs) MostQualified() *AcceptSpec {
	if len(specs) > 0 {
		return &specs[0]
	}
	return nil
}

// sort.Interface methods for accept spec

func (specs AcceptSpecs) Len() int {
	return len(specs)
}

func (specs AcceptSpecs) Swap(i, j int) {
	specs[i], specs[j] = specs[j], specs[i]
}

func (specs AcceptSpecs) Less(i, j int) bool {
	return specs[i].Q > specs[j].Q
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

func htmlContentType() *ContentType {
	return &ContentType{
		Raw:  "text/html; charset=utf-8",
		Mime: "text/html",
		Exts: []string{".html"},
		Params: map[string]string{
			"charset": "utf-8",
		},
	}
}
