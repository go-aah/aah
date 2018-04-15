// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"html/template"
	"io"
	"mime"
	"net"
	"net/http"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"aahframework.org/ahttp.v0"
	"aahframework.org/essentials.v0"
)

func isValidTimeUnit(str string, units ...string) bool {
	for _, v := range units {
		if strings.HasSuffix(str, v) {
			return true
		}
	}
	return false
}

// bodyAllowedForStatus reports whether a given response status code
// permits a body. See RFC 2616, section 4.4.
//
// This method taken from https://golang.org/src/net/http/transfer.go#bodyAllowedForStatus
func bodyAllowedForStatus(status int) bool {
	switch {
	case status >= 100 && status <= 199:
		return false
	case status == 204: // Status NoContent
		return false
	case status == 304: // Status NotModified
		return false
	}
	return true
}

// TODO this method is candidate for essentials library
// move it when you get a time
func firstNonZeroString(values ...string) string {
	for _, v := range values {
		v = strings.TrimSpace(v)
		if len(v) > 0 {
			return v
		}
	}
	return ""
}

// TODO this method is candidate for essentials library
// move it when you get a time
func firstNonZeroInt64(values ...int64) int64 {
	for _, v := range values {
		if v != 0 {
			return v
		}
	}
	return 0
}

// resolveDefaultContentType method returns the Content-Type based on given
// input.
func resolveDefaultContentType(ct string) *ahttp.ContentType {
	switch ct {
	case "html":
		return ahttp.ContentTypeHTML
	case "json":
		return ahttp.ContentTypeJSON
	case "xml":
		return ahttp.ContentTypeXML
	case "text":
		return ahttp.ContentTypePlainText
	case "js":
		return ahttp.ContentTypeJavascript
	default:
		return nil
	}
}

func parseHost(address, toPort string) string {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return address
	}

	if ess.IsStrEmpty(toPort) {
		return host
	}
	return host + ":" + toPort
}

func reverseSlice(s []string) []string {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}

func kind(t reflect.Type) reflect.Kind {
	if t.Kind() == reflect.Ptr {
		return t.Elem().Kind()
	}
	return t.Kind()
}

func actualType(v interface{}) reflect.Type {
	vt := reflect.TypeOf(v)
	if vt.Kind() == reflect.Ptr {
		vt = vt.Elem()
	}

	return vt
}

// createRegistryKeyAndNamespace method creates the controller registry key.
func createRegistryKeyAndNamespace(cType reflect.Type) (string, string) {
	namespace := cType.PkgPath()
	if idx := strings.Index(namespace, "controllers"); idx > -1 {
		namespace = namespace[idx+11:]
	}

	if ess.IsStrEmpty(namespace) {
		return strings.ToLower(cType.Name()), ""
	}

	namespace = strings.TrimPrefix(namespace, string(filepath.Separator))
	return strings.ToLower(path.Join(namespace, cType.Name())), namespace
}

// findEmbeddedContext method does breadth-first search on struct anonymous
// field to find `aah.Context` index positions.
func findEmbeddedContext(controllerType reflect.Type) [][]int {
	var indexes [][]int
	type nodeType struct {
		val   reflect.Value
		index []int
	}

	queue := []nodeType{{reflect.New(controllerType), []int{}}}

	for len(queue) > 0 {
		var (
			node     = queue[0]
			elem     = node.val
			elemType = elem.Type()
		)

		if elemType.Kind() == reflect.Ptr {
			elem = elem.Elem()
			elemType = elem.Type()
		}

		queue = queue[1:]
		if elemType.Kind() != reflect.Struct {
			continue
		}

		for i := 0; i < elem.NumField(); i++ {
			// skip non-anonymous fields
			field := elemType.Field(i)
			if !field.Anonymous {
				continue
			}

			// If it's a `aah.Context`, record the field indexes
			if field.Type == ctxPtrType {
				indexes = append(indexes, append(node.index, i))
				continue
			}

			fieldValue := elem.Field(i)
			queue = append(queue,
				nodeType{fieldValue, append(append([]int{}, node.index...), i)})
		}
	}

	return indexes
}

func sortHeaderKeys(hdrs http.Header) []string {
	keys := make([]string, 0, len(hdrs))
	for key := range hdrs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func parseCacheBustPart(name, part string) string {
	if strings.Contains(name, part) {
		name = strings.Replace(name, "-"+part, "", 1)
		name = strings.Replace(name, part+"-", "", 1)
	}
	return name
}

// checkGzipRequired method return for static which requires gzip response.
func checkGzipRequired(file string) bool {
	switch filepath.Ext(file) {
	case ".css", ".js", ".html", ".htm", ".json", ".xml",
		".txt", ".csv", ".ttf", ".otf", ".eot":
		return true
	default:
		return false
	}
}

// detectFileContentType method to identify the static file content-type.
func detectFileContentType(file string, content io.ReadSeeker) (string, error) {
	ctype := mime.TypeByExtension(filepath.Ext(file))
	if ctype == "" {
		// read a chunk to decide between utf-8 text and binary
		// only 512 bytes expected by `http.DetectContentType`
		var buf [512]byte
		n, _ := io.ReadFull(content, buf[:])
		ctype = http.DetectContentType(buf[:n])

		// rewind to output whole file
		if _, err := content.Seek(0, io.SeekStart); err != nil {
			return "", errSeeker
		}
	}
	return ctype, nil
}

// sanatizeValue method sanatizes string type value, rest we can't do any.
// It's a user responbility.
func sanatizeValue(value interface{}) interface{} {
	switch v := value.(type) {
	case string:
		return template.HTMLEscapeString(v)
	default:
		return v
	}
}

// funcEqual method to compare to function callback interface data. In effect
// comparing the pointers of the indirect layer. Read more about the
// representation of functions here: http://golang.org/s/go11func
func funcEqual(a, b interface{}) bool {
	av := reflect.ValueOf(&a).Elem()
	bv := reflect.ValueOf(&b).Elem()
	return av.InterfaceData() == bv.InterfaceData()
}

// funcName method to get callback function name.
func funcName(f interface{}) string {
	fi := ess.GetFunctionInfo(f)
	return fi.Name
}

func parsePriority(priority ...int) int {
	pr := 1 // default priority is 1
	if len(priority) > 0 && priority[0] > 0 {
		pr = priority[0]
	}
	return pr
}

func stripCharset(ct string) string {
	if idx := strings.IndexByte(ct, ';'); idx > 0 {
		return ct[:idx]
	}
	return ct
}

// wrapGzipWriter method writes respective header for gzip and wraps write into
// gzip writer.
func wrapGzipWriter(res ahttp.ResponseWriter) ahttp.ResponseWriter {
	res.Header().Add(ahttp.HeaderVary, ahttp.HeaderAcceptEncoding)
	res.Header().Add(ahttp.HeaderContentEncoding, gzipContentEncoding)
	res.Header().Del(ahttp.HeaderContentLength)
	return ahttp.WrapGzipWriter(res)
}
