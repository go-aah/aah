// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/ahttp source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ahttp

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"aahframework.org/essentials.v0"
	"aahframework.org/test.v0/assert"
)

func TestHTTPResponseWriter(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		writer := GetResponseWriter(w)
		defer PutResponseWriter(writer)

		writer.WriteHeader(http.StatusOK)
		assert.Equal(t, http.StatusOK, writer.Status())

		_, _ = writer.Write([]byte("aah framework response writer"))
		assert.Equal(t, 29, writer.BytesWritten())
		_ = writer.Header()
		_ = writer.Unwrap()
	}

	callAndValidate(t, handler, "aah framework response writer")
}

func TestHTTPNoStatusWritten(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		writer := GetResponseWriter(w)
		defer PutResponseWriter(writer)

		_, _ = writer.Write([]byte("aah framework no status written"))
		assert.Equal(t, 31, writer.BytesWritten())
	}

	callAndValidate(t, handler, "aah framework no status written")
}

func TestHTTPMultipleStatusWritten(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		writer := GetResponseWriter(w)
		defer PutResponseWriter(writer)

		writer.WriteHeader(http.StatusOK)
		writer.WriteHeader(http.StatusAccepted)

		_, _ = writer.Write([]byte("aah framework mutiple status written"))
		assert.Equal(t, 36, writer.BytesWritten())
	}

	callAndValidate(t, handler, "aah framework mutiple status written")
}

func TestHTTPHijackCall(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		writer := GetResponseWriter(w)

		con, rw, err := writer.(http.Hijacker).Hijack()
		assert.FailOnError(t, err, "")
		defer ess.CloseQuietly(con)

		bytes := []byte("aah framework calling hijack")
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

func TestHTTPCallCloseNotifyAndFlush(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		writer := GetResponseWriter(w)
		defer PutResponseWriter(writer)

		_, _ = writer.Write([]byte("aah framework calling close notify and flush"))
		assert.Equal(t, 44, writer.BytesWritten())

		writer.(http.Flusher).Flush()
		ch := writer.(http.CloseNotifier).CloseNotify()
		assert.NotNil(t, ch)
	}

	callAndValidate(t, handler, "aah framework calling close notify and flush")
}

func callAndValidate(t *testing.T, handler http.HandlerFunc, response string) {
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Error Get: %v", err)
	}
	defer ess.CloseQuietly(resp.Body)

	bytes, _ := ioutil.ReadAll(resp.Body)
	assert.Equal(t, response, string(bytes))
}
