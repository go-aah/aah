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
)

type (
	// ContentType is represents request and response content type values
	ContentType struct {
		Mime   string
		Exts   []string
		Params map[string]string
	}
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Content-Type methods
//___________________________________

// IsEqual method compares give Content-Type string with current instance.
//    E.g.:
//      contentType.IsEqual("application/json")
func (c *ContentType) IsEqual(contentType string) bool {
	return strings.HasPrefix(c.String(), strings.ToLower(contentType))
}

// Charset method returns charset of content-type
// otherwise `defaultCharset` is returned
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
	raw := c.Mime
	for k, v := range c.Params {
		raw += fmt.Sprintf("; %s=%s", k, v)
	}
	return raw
}

// String is stringer interface
func (c *ContentType) String() string {
	return c.Raw()
}
