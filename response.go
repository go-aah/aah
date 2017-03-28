// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/ahttp source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ahttp

import (
	"bufio"
	"errors"
	"io"
	"net"
	"net/http"

	"aahframework.org/essentials.v0"
)

type (
	// ResponseWriter extends the `http.ResponseWriter` interface to implements
	// aah framework response.
	ResponseWriter interface {
		http.ResponseWriter

		// Status returns the HTTP status of the request otherwise 0
		Status() int

		// BytesWritten returns the total number of bytes written
		BytesWritten() int

		// Unwrap returns the original `ResponseWriter`
		Unwrap() http.ResponseWriter
	}

	// Response implements multiple interface (ReaderFrom, CloseNotifier, Flusher,
	// Hijacker) and handy methods for aah framework.
	Response struct {
		w            http.ResponseWriter
		status       int
		wroteStatus  bool
		bytesWritten int
	}
)

// interface compliance
var (
	_ http.CloseNotifier = &Response{}
	_ http.Flusher       = &Response{}
	_ http.Hijacker      = &Response{}
	_ io.Closer          = &Response{}
	_ ResponseWriter     = &Response{}
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Global methods
//___________________________________

// WrapResponseWriter wraps `http.ResponseWriter`, returns aah framework response
// writer that allows to advantage of response process.
func WrapResponseWriter(w http.ResponseWriter) ResponseWriter {
	return &Response{w: w}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Response methods
//___________________________________

// Status method returns HTTP response status code. If status is not yet written
// it reurns 0.
func (r *Response) Status() int {
	return r.status
}

// WriteHeader method writes given status code into Response.
func (r *Response) WriteHeader(code int) {
	if code > 0 && !r.wroteStatus {
		r.status = code
		r.wroteStatus = true
		r.w.WriteHeader(code)
	}
}

// Header method returns response header map.
func (r *Response) Header() http.Header {
	return r.w.Header()
}

// Write method writes bytes into Response.
func (r *Response) Write(b []byte) (int, error) {
	r.setContentTypeIfNotSet(b)
	r.WriteHeader(http.StatusOK)

	size, err := r.w.Write(b)
	r.bytesWritten += size
	return size, err
}

// BytesWritten method returns no. of bytes already written into HTTP response.
func (r *Response) BytesWritten() int {
	return r.bytesWritten
}

// Close method closes the writer if possible.
func (r *Response) Close() error {
	ess.CloseQuietly(r.w)
	return nil
}

// Unwrap method returns the underlying `http.ResponseWriter`
func (r *Response) Unwrap() http.ResponseWriter {
	return r.w
}

// CloseNotify method calls underlying CloseNotify method if it's compatible
func (r *Response) CloseNotify() <-chan bool {
	n := r.w.(http.CloseNotifier)
	return n.CloseNotify()
}

// Flush method calls underlying Flush method if it's compatible
func (r *Response) Flush() {
	if f, ok := r.w.(http.Flusher); ok {
		f.Flush()
	}
}

// Hijack method calls underlying Hijack method if it's compatible otherwise
// returns an error. It becomes the caller's responsibility to manage
// and close the connection.
func (r *Response) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := r.w.(http.Hijacker); ok {
		return h.Hijack()
	}

	return nil, nil, errors.New("http.Hijacker interface is not compatible")
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Response Unexported methods
//___________________________________

func (r *Response) setContentTypeIfNotSet(b []byte) {
	if _, found := r.Header()[HeaderContentType]; !found {
		r.Header().Set(HeaderContentType, http.DetectContentType(b))
	}
}
