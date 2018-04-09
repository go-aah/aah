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
	"sync"
)

var (
	responsePool = &sync.Pool{New: func() interface{} { return &Response{} }}

	// interface compliance
	_ http.CloseNotifier = (*Response)(nil)
	_ http.Flusher       = (*Response)(nil)
	_ http.Hijacker      = (*Response)(nil)
	_ http.Pusher        = (*Response)(nil)
	_ io.Closer          = (*Response)(nil)
	_ ResponseWriter     = (*Response)(nil)
)

// ResponseWriter extends the `http.ResponseWriter` interface to implements
// aah framework response.
type ResponseWriter interface {
	http.ResponseWriter

	// Status returns the HTTP status of the request otherwise 0
	Status() int

	// BytesWritten returns the total number of bytes written
	BytesWritten() int

	// Unwrap returns the original `ResponseWriter`
	Unwrap() http.ResponseWriter
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package methods
//___________________________________

// TODO for old method cleanup

// GetResponseWriter method wraps given writer and returns the aah response writer.
// Deprecated use `AcquireResponseWriter` instead.
func GetResponseWriter(w http.ResponseWriter) ResponseWriter {
	rw := responsePool.Get().(*Response)
	rw.w = w
	return rw
}

// PutResponseWriter method puts response writer back to pool.
// Deprecated use `ReleaseResponseWriter` instead.
func PutResponseWriter(aw ResponseWriter) {
	releaseResponse(aw.(*Response))
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Response
//___________________________________

// Response implements multiple interface (CloseNotifier, Flusher,
// Hijacker) and handy methods for aah framework.
type Response struct {
	w            http.ResponseWriter
	status       int
	wroteStatus  bool
	bytesWritten int
}

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
	if w, ok := r.w.(io.Closer); ok {
		return w.Close()
	}
	return nil
}

// Unwrap method returns the underlying `http.ResponseWriter`
func (r *Response) Unwrap() http.ResponseWriter {
	return r.w
}

// CloseNotify method calls underlying CloseNotify method if it's compatible
func (r *Response) CloseNotify() <-chan bool {
	if n, ok := r.w.(http.CloseNotifier); ok {
		return n.CloseNotify()
	}
	return nil
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

// Push method calls underlying Push method HTTP/2 if compatible otherwise
// returns nil
func (r *Response) Push(target string, opts *http.PushOptions) error {
	if p, ok := r.w.(http.Pusher); ok {
		return p.Push(target, opts)
	}
	return nil
}

// Reset method resets the instance value for repurpose.
func (r *Response) Reset() {
	r.w = nil
	r.status = 0
	r.bytesWritten = 0
	r.wroteStatus = false
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Response Unexported methods
//___________________________________

// releaseResponse method puts response back to pool.
func releaseResponse(r *Response) {
	_ = r.Close()
	r.Reset()
	responsePool.Put(r)
}
