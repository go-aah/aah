// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ahttp

import (
	"bufio"
	"errors"
	"io"
	"net"
	"net/http"

	"aahframework.org/log"
)

const (
	defaultStatus = http.StatusOK
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

		// WriteHeaderNow method write status and header on the wire
		WriteHeaderNow()
	}

	// Response implements multiple interface (ReaderFrom, CloseNotifier, Flusher,
	// Hijacker) and handy methods for aah framework.
	Response struct {
		w                 http.ResponseWriter
		status            int
		wroteStatus       bool
		wroteStatusHeader bool
		bytesWritten      int
	}
)

// interface compilance
var _ http.CloseNotifier = &Response{}
var _ http.Flusher = &Response{}
var _ http.Hijacker = &Response{}
var _ io.ReaderFrom = &Response{}
var _ ResponseWriter = &Response{}

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
	if code > 0 {
		if r.wroteStatus && r.status != code {
			log.Warnf("Status already written, overriding status code %d with %d", r.status, code)
		}
		r.status = code
		r.wroteStatus = true
	}
}

// WriteHeaderNow method writes the status code on the wire.
func (r *Response) WriteHeaderNow() {
	if r.wroteStatusHeader {
		return
	}

	if r.status == 0 {
		r.status = defaultStatus
	}

	r.w.WriteHeader(r.status)
	r.wroteStatusHeader = true
}

// Header method returns response header map.
func (r *Response) Header() http.Header {
	return r.w.Header()
}

// Write method writes bytes into Response.
func (r *Response) Write(buf []byte) (int, error) {
	r.WriteHeaderNow()
	size, err := r.w.Write(buf)
	r.bytesWritten += size
	return size, err
}

// ReadFrom method calls underlying ReadFrom method with given reader if it's
// compatiable and writes HTTP status OK (200) if it's not written yet.
func (r *Response) ReadFrom(rdr io.Reader) (int64, error) {
	if rf, ok := r.w.(io.ReaderFrom); ok {
		r.WriteHeaderNow()
		size, err := rf.ReadFrom(rdr)
		r.bytesWritten += int(size) // might lose size info
		return size, err
	}
	return 0, errors.New("io.ReaderFrom interface is not implemented")
}

// BytesWritten method returns no. of bytes already written into HTTP response.
func (r *Response) BytesWritten() int {
	return r.bytesWritten
}

// Unwrap method returns the underlying `ResponseWriter`
func (r *Response) Unwrap() http.ResponseWriter {
	return r.w
}

// CloseNotify method calls underlying CloseNotify method if it's compatiable
func (r *Response) CloseNotify() <-chan bool {
	n := r.w.(http.CloseNotifier)
	return n.CloseNotify()
}

// Flush method calls underlying Flush method if it's compatiable
func (r *Response) Flush() {
	if f, ok := r.w.(http.Flusher); ok {
		f.Flush()
	}
}

// Hijack method calls underlying Hijack method if it's compatiable otherwise
// returns an error. It becomes the caller's responsibility to manage
// and close the connection.
func (r *Response) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := r.w.(http.Hijacker); ok {
		return h.Hijack()
	}

	return nil, nil, errors.New("http.Hijacker interface is not compatiable")
}
