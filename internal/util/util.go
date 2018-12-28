// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package util

import (
	"html/template"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"reflect"
	"strings"

	"aahframe.work/ahttp"
)

// IsValidTimeUnit method to check supported time unit suffixes.
// If supported returns true otherwise false.
func IsValidTimeUnit(str string, units ...string) bool {
	for _, v := range units {
		if strings.HasSuffix(str, v) {
			return true
		}
	}
	return false
}

// DetectFileContentType method to identify the static file content-type.
// It's similar to http.DetectContentType but has windows OS and corner case
// covered.
func DetectFileContentType(file string, content io.ReadSeeker) (string, error) {
	ctype := MimeTypeByExtension(filepath.Ext(file))
	if ctype == "" {
		// read a chunk to decide between utf-8 text and binary
		// only 512 bytes expected by `http.DetectContentType`
		var buf [512]byte
		n, _ := io.ReadFull(content, buf[:]) // #nosec
		ctype = http.DetectContentType(buf[:n])

		// rewind to output whole file
		if _, err := content.Seek(0, io.SeekStart); err != nil {
			return "", err
		}
	}
	return ctype, nil
}

// MimeTypeByExtension method to get MIME info by file extension with corner case
// covered since mime.TypeByExtension behaves wired on windows for `.js` and `.css`,
// it better to have some basic measure
func MimeTypeByExtension(ext string) string {
	switch ext {
	case ".html", ".htm":
		return ahttp.ContentTypeHTML.String()
	case ".css":
		return ahttp.ContentTypeCSSText.String()
	case ".js":
		return ahttp.ContentTypeJavascript.String()
	case ".json":
		return ahttp.ContentTypeJSON.String()
	case ".xml":
		return ahttp.ContentTypeXML.String()
	case ".txt", ".text":
		return ahttp.ContentTypePlainText.String()
	default:
		return mime.TypeByExtension(ext)
	}
}

// FuncEqual method to compare to function callback interface data. In effect
// comparing the pointers of the indirect layer. Read more about the
// representation of functions here: http://golang.org/s/go11func
func FuncEqual(a, b interface{}) bool {
	av := reflect.ValueOf(&a).Elem()
	bv := reflect.ValueOf(&b).Elem()
	return av.InterfaceData() == bv.InterfaceData()
}

// OnlyMIME method to strip everything after `;` from the content type value.
func OnlyMIME(ct string) string {
	if idx := strings.IndexByte(ct, ';'); idx > 0 {
		return ct[:idx]
	}
	return ct
}

// AddQueryString method to add the given query string key value pair appropriately
// to the given URL string.
func AddQueryString(u, k, v string) string {
	v = url.QueryEscape(v)
	if len(u) == 0 {
		return "?" + k + "=" + v
	}
	if idx := strings.IndexByte(u, '?'); idx == -1 {
		return u + "?" + k + "=" + v
	}
	return u + "&" + k + "=" + v
}

// SanitizeValue method to sanatizes type `string`, rest we can't do any.
// It's a user responbility.
func SanitizeValue(value interface{}) interface{} {
	switch v := value.(type) {
	case string:
		return template.HTMLEscapeString(v)
	default:
		return v
	}
}

// IsGzipWorthForFile method to decide whether gzipping file content it worth to do by
// checking file extension. If its worth to do it returns true otherwise false.
func IsGzipWorthForFile(name string) bool {
	switch filepath.Ext(name) {
	case ".css", ".js", ".html", ".htm", ".json", ".ico", ".svg",
		".eot", ".ttf", ".otf", ".xml", ".rss", ".txt", ".csv":
		return true
	default:
		return false
	}
}

// FirstNonEmpty method returns the first non-empty string from given var args
// otherwise empty string.
func FirstNonEmpty(values ...string) string {
	for _, v := range values {
		v = strings.TrimSpace(v)
		if len(v) > 0 {
			return v
		}
	}
	return ""
}
