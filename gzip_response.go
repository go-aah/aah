// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/ahttp source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ahttp

import (
	"bufio"
	"compress/gzip"
	"io"
	"net"
	"net/http"
	"sync"
)

var (
	// GzipLevel holds value from app config.
	GzipLevel int

	grPool = &sync.Pool{New: func() interface{} { return &GzipResponse{} }}
	gwPool = &sync.Pool{}

	// interface compliance
	_ http.CloseNotifier = (*GzipResponse)(nil)
	_ http.Flusher       = (*GzipResponse)(nil)
	_ http.Hijacker      = (*GzipResponse)(nil)
	_ http.Pusher        = (*GzipResponse)(nil)
	_ io.Closer          = (*GzipResponse)(nil)
	_ ResponseWriter     = (*GzipResponse)(nil)
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package methods
//___________________________________

// TODO for old method cleanup

// GetGzipResponseWriter wraps `http.ResponseWriter`, returns aah framework response
// writer that allows to advantage of response process.
// Deprecated use `WrapGzipWriter` instead.
func GetGzipResponseWriter(w ResponseWriter) ResponseWriter {
	gr := grPool.Get().(*GzipResponse)
	gr.gw = acquireGzipWriter(w)
	gr.r = w.(*Response)
	return gr
}

// PutGzipResponseWiriter method resets and puts the gzip writer into pool.
// Deprecated use `ReleaseResponseWriter` instead.
func PutGzipResponseWiriter(rw ResponseWriter) {
	releaseGzipResponse(rw.(*GzipResponse))
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// GzipResponse
//___________________________________

// GzipResponse extends `ahttp.Response` to provides gzip compression for response
// bytes to the underlying response.
type GzipResponse struct {
	r  *Response
	gw *gzip.Writer
}

// Status method returns HTTP response status code. If status is not yet written
// it reurns 0.
func (g *GzipResponse) Status() int {
	return g.r.Status()
}

// WriteHeader method writes given status code into Response.
func (g *GzipResponse) WriteHeader(code int) {
	g.r.WriteHeader(code)
}

// Header method returns response header map.
func (g *GzipResponse) Header() http.Header {
	return g.r.Header()
}

// Write method writes bytes into Response.
func (g *GzipResponse) Write(b []byte) (int, error) {
	g.r.WriteHeader(http.StatusOK)
	size, err := g.gw.Write(b)
	g.r.bytesWritten += size
	return size, err
}

// BytesWritten method returns no. of bytes already written into HTTP response.
func (g *GzipResponse) BytesWritten() int {
	return g.r.BytesWritten()
}

// Close method closes the writer if possible.
func (g *GzipResponse) Close() error {
	if err := g.gw.Close(); err != nil {
		return err
	}
	return g.r.Close()
}

// Unwrap method returns the underlying `http.ResponseWriter`
func (g *GzipResponse) Unwrap() http.ResponseWriter {
	return g.r.Unwrap()
}

// CloseNotify method calls underlying CloseNotify method if it's compatible
func (g *GzipResponse) CloseNotify() <-chan bool {
	return g.r.CloseNotify()
}

// Flush method calls underlying Flush method if it's compatible
func (g *GzipResponse) Flush() {
	if g.gw != nil {
		_ = g.gw.Flush()
	}

	g.r.Flush()
}

// Hijack method calls underlying Hijack method if it's compatible otherwise
// returns an error. It becomes the caller's responsibility to manage
// and close the connection.
func (g *GzipResponse) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return g.r.Hijack()
}

// Push method calls underlying Push method HTTP/2 if compatible otherwise
// returns nil
func (g *GzipResponse) Push(target string, opts *http.PushOptions) error {
	return g.r.Push(target, opts)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// GzipResponse Unexported methods
//___________________________________

// releaseGzipResponse method resets and puts the gzip response into pool.
func releaseGzipResponse(gw *GzipResponse) {
	_ = gw.Close()
	gwPool.Put(gw.gw)
	releaseResponse(gw.r)
	grPool.Put(gw)
}

func acquireGzipWriter(w io.Writer) *gzip.Writer {
	gw := gwPool.Get()
	if gw == nil {
		if ngw, err := gzip.NewWriterLevel(w, GzipLevel); err == nil {
			return ngw
		}
		return nil
	}
	ngw := gw.(*gzip.Writer)
	ngw.Reset(w)
	return ngw
}
