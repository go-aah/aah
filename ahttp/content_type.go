// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ahttp

import (
	"fmt"
	"strings"
)

var (
	// ContentTypeHTML HTML content type.
	ContentTypeHTML = &ContentType{
		Mime: "text/html",
		Exts: []string{".html", ".htm"},
		Params: map[string]string{
			"charset": "utf-8",
		},
	}

	// ContentTypeJSON JSON content type.
	ContentTypeJSON = &ContentType{
		Mime: "application/json",
		Exts: []string{".json"},
		Params: map[string]string{
			"charset": "utf-8",
		},
	}

	// ContentTypeXML XML content type.
	ContentTypeXML = &ContentType{
		Mime: "application/xml",
		Exts: []string{".xml"},
		Params: map[string]string{
			"charset": "utf-8",
		},
	}

	// ContentTypePlainText content type.
	ContentTypePlainText = &ContentType{
		Mime: "text/plain",
		Params: map[string]string{
			"charset": "utf-8",
		},
	}

	// ContentTypeOctetStream content type for bytes.
	ContentTypeOctetStream = &ContentType{
		Mime: "application/octet-stream",
	}
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

// IsEqual method compare give Content-Type string wth instance.
//    E.g.:
//      contentType.IsEqual("application/json")
func (c *ContentType) IsEqual(contentType string) bool {
	return strings.HasPrefix(c.Raw(), strings.ToLower(contentType))
}

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
