// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org/ahttp source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ahttp

import (
	"mime"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"aahframe.work/aah/essentials"
	"aahframe.work/aah/log"
)

const vendorTreePrefix = "vnd."

// HTTP Header names
const (
	HeaderAccept                          = "Accept"
	HeaderAcceptEncoding                  = "Accept-Encoding"
	HeaderAcceptLanguage                  = "Accept-Language"
	HeaderAcceptRanges                    = "Accept-Ranges"
	HeaderAccessControlAllowCredentials   = "Access-Control-Allow-Credentials"
	HeaderAccessControlAllowHeaders       = "Access-Control-Allow-Headers"
	HeaderAccessControlAllowMethods       = "Access-Control-Allow-Methods"
	HeaderAccessControlAllowOrigin        = "Access-Control-Allow-Origin"
	HeaderAccessControlExposeHeaders      = "Access-Control-Expose-Headers"
	HeaderAccessControlMaxAge             = "Access-Control-Max-Age"
	HeaderAccessControlRequestHeaders     = "Access-Control-Request-Headers"
	HeaderAccessControlRequestMethod      = "Access-Control-Request-Method"
	HeaderAge                             = "Age"
	HeaderAllow                           = "Allow"
	HeaderAuthorization                   = "Authorization"
	HeaderCacheControl                    = "Cache-Control"
	HeaderConnection                      = "Connection"
	HeaderContentDisposition              = "Content-Disposition"
	HeaderContentEncoding                 = "Content-Encoding"
	HeaderContentLength                   = "Content-Length"
	HeaderContentType                     = "Content-Type"
	HeaderContentSecurityPolicy           = "Content-Security-Policy"
	HeaderContentSecurityPolicyReportOnly = "Content-Security-Policy-Report-Only"
	HeaderCookie                          = "Cookie"
	HeaderDate                            = "Date"
	HeaderETag                            = "ETag"
	HeaderExpires                         = "Expires"
	HeaderHost                            = "Host"
	HeaderIfMatch                         = "If-Match"
	HeaderIfModifiedSince                 = "If-Modified-Since"
	HeaderIfNoneMatch                     = "If-None-Match"
	HeaderIfRange                         = "If-Range"
	HeaderIfUnmodifiedSince               = "If-Unmodified-Since"
	HeaderKeepAlive                       = "Keep-Alive"
	HeaderLastModified                    = "Last-Modified"
	HeaderLocation                        = "Location"
	HeaderOrigin                          = "Origin"
	HeaderMethod                          = "Method"
	HeaderPublicKeyPins                   = "Public-Key-Pins"
	HeaderRange                           = "Range"
	HeaderReferer                         = "Referer"
	HeaderReferrerPolicy                  = "Referrer-Policy"
	HeaderRetryAfter                      = "Retry-After"
	HeaderServer                          = "Server"
	HeaderSetCookie                       = "Set-Cookie"
	HeaderStatus                          = "Status"
	HeaderStrictTransportSecurity         = "Strict-Transport-Security"
	HeaderTransferEncoding                = "Transfer-Encoding"
	HeaderUpgrade                         = "Upgrade"
	HeaderUserAgent                       = "User-Agent"
	HeaderVary                            = "Vary"
	HeaderWWWAuthenticate                 = "WWW-Authenticate"
	HeaderXContentTypeOptions             = "X-Content-Type-Options"
	HeaderXDNSPrefetchControl             = "X-DNS-Prefetch-Control"
	HeaderXCSRFToken                      = "X-CSRF-Token"
	HeaderXForwardedFor                   = "X-Forwarded-For"
	HeaderXForwardedHost                  = "X-Forwarded-Host"
	HeaderXForwardedPort                  = "X-Forwarded-Port"
	HeaderXForwardedProto                 = "X-Forwarded-Proto"
	HeaderXForwardedProtocol              = "X-Forwarded-Protocol"
	HeaderXForwardedSsl                   = "X-Forwarded-Ssl"
	HeaderXUrlScheme                      = "X-Url-Scheme"
	HeaderXForwardedServer                = "X-Forwarded-Server"
	HeaderXFrameOptions                   = "X-Frame-Options"
	HeaderXHTTPMethodOverride             = "X-HTTP-Method-Override"
	HeaderXPermittedCrossDomainPolicies   = "X-Permitted-Cross-Domain-Policies"
	HeaderXRealIP                         = "X-Real-Ip"
	HeaderXRequestedWith                  = "X-Requested-With"
	HeaderXRequestID                      = "X-Request-Id"
	HeaderXXSSProtection                  = "X-XSS-Protection"
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
// Global HTTP header methods
//___________________________________

// NegotiateContentType method negotiates the response `Content-Type` from the given HTTP
// `Accept` header. The resolve order is- 1) URL extension 2) Accept header
// Most quailfied one based quality factor otherwise default is HTML.
func NegotiateContentType(req *http.Request) *ContentType {
	// 1) URL extension
	ext := filepath.Ext(req.URL.Path)
	switch ext {
	case ".html", ".htm", ".json", ".js", ".xml", ".txt":
		return parseMediaType(mime.TypeByExtension(ext))
	}

	// 2) From Accept header
	spec := ParseAccept(req, HeaderAccept).MostQualified()
	if spec == nil {
		// if parsed spec is nil return content type as HTML.
		return ContentTypeHTML
	}

	// 3) Accept Header Vendor Types
	// RFC4288 https://tools.ietf.org/html/rfc4288#section-3.2
	if parts, yes := isVendorType(spec.Value); yes {
		subparts := strings.Split(parts[1], "+")
		if strings.Contains(subparts[0], "-v") {
			verparts := strings.Split(subparts[0], "-v")
			spec.Params["vendor"] = strings.TrimPrefix(verparts[0], vendorTreePrefix)
			spec.Params["version"] = verparts[1]
		} else {
			spec.Params["vendor"] = strings.TrimPrefix(subparts[0], vendorTreePrefix)
		}

		// Rewrite the Content-Type
		spec.Value = parts[0] + "/" + subparts[1]
	}

	exts, _ := mime.ExtensionsByType(spec.Value)
	return newContentType(spec.Value, exts, spec.Params)
}

// NegotiateLocale method negotiates the `Accept-Language` from the given HTTP
// request. Most quailfied one based on quality factor.
func NegotiateLocale(req *http.Request) *Locale {
	return ToLocale(ParseAccept(req, HeaderAcceptLanguage).MostQualified())
}

// NegotiateEncoding negotiates the `Accept-Encoding` from the given HTTP
// request. Most quailfied one based on quality factor.
func NegotiateEncoding(req *http.Request) *AcceptSpec {
	return ParseAcceptEncoding(req).MostQualified()
}

// ParseContentType method parses the request header `Content-Type` as per RFC1521.
func ParseContentType(req *http.Request) *ContentType {
	contentType := req.Header.Get(HeaderContentType)
	if contentType == "" {
		return ContentTypeHTML
	}
	return parseMediaType(contentType)
}

// ParseAcceptEncoding method parses the request HTTP header `Accept-Encoding`
// as per RFC7231 https://tools.ietf.org/html/rfc7231#section-5.3.4. It returns
// `AcceptSpecs`.
func ParseAcceptEncoding(req *http.Request) AcceptSpecs {
	return ParseAccept(req, HeaderAcceptEncoding)
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
				Raw:    hv,
				Value:  parts[0],
				Q:      float32(1.0),
				Params: make(map[string]string),
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

// ToLocale method creates a locale instance from `AcceptSpec`
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

// NewLocale method returns locale instance for given locale string.
func NewLocale(value string) *Locale {
	return ToLocale(
		&AcceptSpec{
			Raw:   value,
			Value: value,
		})
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Locale methods
//___________________________________

// String is stringer interface.
func (l Locale) String() string {
	return l.Raw
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// AcceptSpecs methods
//___________________________________

// GetParam method returns the Accept* header param value otherwise returns default
// value.
// 	For e.g.:
// 		Accept: application/json; version=2
//
// 		Method returns `2` for key `version`
func (a AcceptSpec) GetParam(key string, defaultValue string) string {
	if v, found := a.Params[key]; found {
		return v
	}
	return defaultValue
}

// MostQualified method returns the most quailfied accept spec, since `AcceptSpec` is
// sorted by quaity factor. First position is the most quailfied otherwise `nil`.
func (specs AcceptSpecs) MostQualified() *AcceptSpec {
	if len(specs) > 0 {
		return &specs[0]
	}
	return nil
}

// sort.Interface methods for accept spec
func (specs AcceptSpecs) Len() int           { return len(specs) }
func (specs AcceptSpecs) Swap(i, j int)      { specs[i], specs[j] = specs[j], specs[i] }
func (specs AcceptSpecs) Less(i, j int) bool { return specs[i].Q > specs[j].Q }

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

// isVendorType method check the mime type is vendor type as per
// RFC4288 https://tools.ietf.org/html/rfc4288#section-3.2 - Vendor Tree
// i.e. `vnd.` prefix.
func isVendorType(mime string) ([]string, bool) {
	parts := strings.Split(mime, "/")
	return parts, strings.HasPrefix(parts[1], vendorTreePrefix)
}

// parseMediaType method parses a media type value and any optional
// parameters, per RFC 1521. the values in Content-Type and
// Content-Disposition headers (RFC 2183).
func parseMediaType(value string) *ContentType {
	ctype, params, err := mime.ParseMediaType(value)
	if err != nil {
		log.Errorf("%v for value: %v", err, value)
	}

	exts, _ := mime.ExtensionsByType(ctype)
	return newContentType(ctype, exts, params)
}
