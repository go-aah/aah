// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/ahttp source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ahttp

import (
	"fmt"
	"strings"
)

var (
	// ContentTypeHTML HTML content type.
	ContentTypeHTML = parseMediaType("text/html; charset=utf-8")

	// ContentTypeJSON JSON content type.
	ContentTypeJSON = parseMediaType("application/json; charset=utf-8")

	// ContentTypeJSONText JSON text content type.
	ContentTypeJSONText = parseMediaType("text/json; charset=utf-8")

	// ContentTypeXML XML content type.
	ContentTypeXML = parseMediaType("application/xml; charset=utf-8")

	// ContentTypeXMLText XML text content type.
	ContentTypeXMLText = parseMediaType("text/xml; charset=utf-8")

	// ContentTypeMultipartForm form data and File.
	ContentTypeMultipartForm = parseMediaType("multipart/form-data")

	// ContentTypeForm form data type.
	ContentTypeForm = parseMediaType("application/x-www-form-urlencoded")

	// ContentTypePlainText content type.
	ContentTypePlainText = parseMediaType("text/plain; charset=utf-8")

	// ContentTypeOctetStream content type for bytes.
	ContentTypeOctetStream = parseMediaType("application/octet-stream")

	// ContentTypeJavascript content type.
	ContentTypeJavascript = parseMediaType("application/javascript; charset=utf-8")

	// ContentTypeEventStream Server-Sent Events content type.
	ContentTypeEventStream = parseMediaType("text/event-stream")
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Content-Type
//___________________________________

// ContentType is represents request and response content type values
type ContentType struct {
	Mime   string
	raw    string
	Exts   []string
	Params map[string]string
}

// IsEqual method returns true if its equals to current content-type instance
// otherwise false.
//    E.g.:
//      contentType.IsEqual("application/json")
//      contentType.IsEqual("application/json; charset=utf-8")
func (c *ContentType) IsEqual(contentType string) bool {
	return strings.HasPrefix(contentType, c.Mime)
}

// Charset method returns charset of content-type
// otherwise `defaultCharset` is returned
// 	For e.g.:
// 		Content-Type: application/json; charset=utf-8
//
// 		Method returns `utf-8`
func (c *ContentType) Charset(defaultCharset string) string {
	if v, found := c.Params["charset"]; found {
		return v
	}
	return defaultCharset
}

// Version method returns Accept header version paramater value if present
// otherwise empty string
// 	For e.g.:
// 		Accept: application/json; version=2
// 		Accept: application/vnd.mycompany.myapp.customer-v2+json
//
// 		Method returns `2`
func (c *ContentType) Version() string {
	return c.GetParam("version")
}

// Vendor method returns Accept header vendor info if present
// otherwise empty string
// 	For e.g.:
// 		Accept: application/vnd.mycompany.myapp.customer-v2+json
//
// 		Method returns `mycompany.myapp.customer`
func (c *ContentType) Vendor() string {
	return c.GetParam("vendor")
}

// GetParam method returns the media type paramater of Accept Content-Type header
// otherwise returns empty string
// value.
// 	For e.g.:
// 		Accept: application/json; version=2
//
// 		Method returns `2` for key `version`
func (c *ContentType) GetParam(key string) string {
	if v, found := c.Params[key]; found {
		return v
	}
	return ""
}

// Raw method returns complete Content-Type composed.
//    E.g.: application/json; charset=utf-8; version=2
func (c *ContentType) Raw() string {
	return c.raw
}

// String is stringer interface
func (c ContentType) String() string {
	return c.raw
}

func newContentType(ctype string, exts []string, params map[string]string) *ContentType {
	raw := ctype
	for k, v := range params {
		raw += fmt.Sprintf("; %s=%s", k, v)
	}
	return &ContentType{Mime: ctype, Exts: exts, Params: params, raw: raw}
}
