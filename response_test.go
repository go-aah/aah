// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ahttp

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"aahframework.org/test/assert"
)

func TestResponseWriter(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		writer := WrapResponseWriter(w)
		defer writer.(*Response).Close()

		writer.WriteHeader(http.StatusOK)
		assert.Equal(t, http.StatusOK, writer.Status())

		_, _ = writer.Write([]byte("aah framework response writer"))
		assert.Equal(t, 29, writer.BytesWritten())

		buf := bytes.NewBufferString(" test")
		_, _ = writer.(io.ReaderFrom).ReadFrom(buf)

		assert.Equal(t, 34, writer.BytesWritten())
		_ = writer.Header()
		_ = writer.Unwrap()
	}

	callAndValidate(t, handler, "aah framework response writer test")
}

func TestNoStatusWritten(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		writer := WrapResponseWriter(w)

		_, _ = writer.Write([]byte("aah framework no status written"))
		assert.Equal(t, 31, writer.BytesWritten())
	}

	callAndValidate(t, handler, "aah framework no status written")
}

func TestMultipleStatusWritten(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		writer := WrapResponseWriter(w)

		writer.WriteHeader(http.StatusOK)
		writer.WriteHeader(http.StatusAccepted)

		_, _ = writer.Write([]byte("aah framework mutiple status written"))
		assert.Equal(t, 36, writer.BytesWritten())
	}

	callAndValidate(t, handler, "aah framework mutiple status written")
}

func TestHijackCall(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		writer := WrapResponseWriter(w)

		con, rw, err := writer.(http.Hijacker).Hijack()
		assert.FailOnError(t, err, "")
		defer con.Close()

		bytes := []byte("aah framework calling hijack")
		rw.WriteString("HTTP/1.1 200 OK\r\n")
		rw.WriteString("Date: " + time.Now().Format(http.TimeFormat) + "\r\n")
		rw.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
		rw.WriteString("Content-Length: " + strconv.Itoa(len(bytes)))
		rw.WriteString(string(bytes) + "\r\n")
		_ = rw.Flush()
	}

	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	http.Get(server.URL)
}

func TestCallCloseNotifyAndFlush(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		writer := WrapResponseWriter(w)

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
	defer resp.Body.Close()

	bytes, _ := ioutil.ReadAll(resp.Body)
	assert.Equal(t, response, string(bytes))
}
