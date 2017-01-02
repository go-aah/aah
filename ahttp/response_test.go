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
	"testing"

	"aahframework.org/test/assert"
)

func TestResponseWriter(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		writer := WrapResponseWriter(w)

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

	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Error Get: %v", err)
	}

	bytes, _ := ioutil.ReadAll(resp.Body)
	assert.Equal(t, "aah framework response writer test", string(bytes))
}
