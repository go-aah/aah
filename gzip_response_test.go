// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/ahttp source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ahttp

import (
	"compress/gzip"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"aahframework.org/essentials.v0"
	"aahframework.org/test.v0/assert"
)

func TestHTTPGzipWriter(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		GzipLevel = gzip.BestSpeed
		gw := GetGzipResponseWriter(GetResponseWriter(w))
		defer PutGzipResponseWiriter(gw)

		gw.Header().Set(HeaderVary, HeaderAcceptEncoding)
		gw.Header().Set(HeaderContentEncoding, "gzip")
		gw.WriteHeader(http.StatusOK)

		_, _ = gw.Write([]byte(`aah framework - testing gzip response writer

      Package ahttp is to cater HTTP helper methods for aah framework.
      Like parse HTTP headers, ResponseWriter, content type, etc.

      Dir method returns a http.Filesystem that can be directly used by http.FileServer().
      It works the same as http.Dir() also provides ability to disable directory listing
      with http.FileServer
      `))
		assert.Equal(t, 407, gw.BytesWritten())
		assert.Equal(t, 200, gw.Status())
		assert.NotNil(t, gw.Unwrap())

		gw.(http.Flusher).Flush()

		_ = gw.(http.Pusher).Push("/test/sample.txt", nil)

		ch := gw.(http.CloseNotifier).CloseNotify()
		assert.NotNil(t, ch)
	}

	resp := gzipCallAndValidate(t, handler)
	assert.Equal(t, 397, len(resp))
	assert.True(t, strings.HasPrefix(string(resp), "aah framework - testing gzip response writer"))
}

func TestHTTPGzipWriter2(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		GzipLevel = gzip.BestSpeed
		gw := WrapGzipWriter(AcquireResponseWriter(w))
		defer ReleaseResponseWriter(gw)

		gw.Header().Set(HeaderVary, HeaderAcceptEncoding)
		gw.Header().Set(HeaderContentEncoding, "gzip")
		gw.WriteHeader(http.StatusOK)

		_, _ = gw.Write([]byte(`aah framework - testing gzip response writer

      Package ahttp is to cater HTTP helper methods for aah framework.
      Like parse HTTP headers, ResponseWriter, content type, etc.

      Dir method returns a http.Filesystem that can be directly used by http.FileServer().
      It works the same as http.Dir() also provides ability to disable directory listing
      with http.FileServer

			Streamlined pool and methods.
      `))
		assert.Equal(t, 441, gw.BytesWritten())
		assert.Equal(t, 200, gw.Status())
		assert.NotNil(t, gw.Unwrap())
	}

	resp := gzipCallAndValidate(t, handler)
	assert.Equal(t, 431, len(resp))
	assert.True(t, strings.HasPrefix(string(resp), "aah framework - testing gzip response writer"))
}

func TestHTTPGzipHijack(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		GzipLevel = gzip.BestSpeed
		for i := 0; i < 5; i++ {
			ngw, _ := gzip.NewWriterLevel(w, GzipLevel)
			gwPool.Put(ngw)
		}
		gw := WrapGzipWriter(GetResponseWriter(w))

		con, rw, err := gw.(http.Hijacker).Hijack()
		assert.FailOnError(t, err, "")
		defer ess.CloseQuietly(con)

		bytes := []byte("aah framework calling gzip hijack")
		_, _ = rw.WriteString("HTTP/1.1 200 OK\r\n")
		_, _ = rw.WriteString("Date: " + time.Now().Format(http.TimeFormat) + "\r\n")
		_, _ = rw.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
		_, _ = rw.WriteString("Content-Length: " + strconv.Itoa(len(bytes)))
		_, _ = rw.WriteString(string(bytes) + "\r\n")
		_ = rw.Flush()
	}

	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	_, _ = http.Get(server.URL)
}

func gzipCallAndValidate(t *testing.T, handler http.HandlerFunc) []byte {
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Error Get: %v", err)
	}
	defer ess.CloseQuietly(resp.Body)

	bytes, _ := ioutil.ReadAll(resp.Body)
	return bytes
}
