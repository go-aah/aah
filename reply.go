// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"bytes"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"aahframework.org/ahttp"
)

var (
	bufPool = &sync.Pool{New: func() interface{} { return new(bytes.Buffer) }}
)

// Reply gives you control and convenient way to write a response effectively.
type Reply struct {
	Rdr      Render
	Code     int
	ContType string

	redirect bool
	done     bool
	gzip     bool
	path     string
	ctx      *Context
	body     *bytes.Buffer
	cookies  []*http.Cookie
	err      *Error
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Reply - HTTP Status Code
//______________________________________________________________________________

// Status method sets the HTTP Code code for the response.
// Also Reply instance provides easy to use method for very frequently used
// HTTP Status Codes reference: http://www.restapitutorial.com/httpCodecodes.html
func (r *Reply) Status(code int) *Reply {
	r.Code = code
	return r
}

// Ok method sets the HTTP Code as 200 RFC 7231, 6.3.1.
func (r *Reply) Ok() *Reply {
	return r.Status(http.StatusOK)
}

// Created method sets the HTTP Code as 201 RFC 7231, 6.3.2.
func (r *Reply) Created() *Reply {
	return r.Status(http.StatusCreated)
}

// Accepted method sets the HTTP Code as 202 RFC 7231, 6.3.3.
func (r *Reply) Accepted() *Reply {
	return r.Status(http.StatusAccepted)
}

// NoContent method sets the HTTP Code as 204 RFC 7231, 6.3.5.
func (r *Reply) NoContent() *Reply {
	return r.Status(http.StatusNoContent)
}

// MovedPermanently method sets the HTTP Code as 301 RFC 7231, 6.4.2.
func (r *Reply) MovedPermanently() *Reply {
	return r.Status(http.StatusMovedPermanently)
}

// Found method sets the HTTP Code as 302 RFC 7231, 6.4.3.
func (r *Reply) Found() *Reply {
	return r.Status(http.StatusFound)
}

// TemporaryRedirect method sets the HTTP Code as 307 RFC 7231, 6.4.7.
func (r *Reply) TemporaryRedirect() *Reply {
	return r.Status(http.StatusTemporaryRedirect)
}

// BadRequest method sets the HTTP Code as 400 RFC 7231, 6.5.1.
func (r *Reply) BadRequest() *Reply {
	return r.Status(http.StatusBadRequest)
}

// Unauthorized method sets the HTTP Code as 401 RFC 7235, 3.1.
func (r *Reply) Unauthorized() *Reply {
	return r.Status(http.StatusUnauthorized)
}

// Forbidden method sets the HTTP Code as 403 RFC 7231, 6.5.3.
func (r *Reply) Forbidden() *Reply {
	return r.Status(http.StatusForbidden)
}

// NotFound method sets the HTTP Code as 404 RFC 7231, 6.5.4.
func (r *Reply) NotFound() *Reply {
	return r.Status(http.StatusNotFound)
}

// MethodNotAllowed method sets the HTTP Code as 405 RFC 7231, 6.5.5.
func (r *Reply) MethodNotAllowed() *Reply {
	return r.Status(http.StatusMethodNotAllowed)
}

// NotAcceptable method sets the HTTP Code as 406 RFC 7231, 6.5.6
func (r *Reply) NotAcceptable() *Reply {
	return r.Status(http.StatusNotAcceptable)
}

// Conflict method sets the HTTP Code as 409 RFC 7231, 6.5.8.
func (r *Reply) Conflict() *Reply {
	return r.Status(http.StatusConflict)
}

// UnsupportedMediaType method sets the HTTP Code as 415 RFC 7231, 6.5.13
func (r *Reply) UnsupportedMediaType() *Reply {
	return r.Status(http.StatusUnsupportedMediaType)
}

// InternalServerError method sets the HTTP Code as 500 RFC 7231, 6.6.1.
func (r *Reply) InternalServerError() *Reply {
	return r.Status(http.StatusInternalServerError)
}

// ServiceUnavailable method sets the HTTP Code as 503 RFC 7231, 6.6.4.
func (r *Reply) ServiceUnavailable() *Reply {
	return r.Status(http.StatusServiceUnavailable)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Reply - Content Types
//______________________________________________________________________________

// ContentType method sets given Content-Type string for the response.
// Also Reply instance provides easy to use method for very frequently used
// Content-Type(s).
//
// By default aah framework try to determine response 'Content-Type' from
// 'ahttp.Request.AcceptContentType()'.
func (r *Reply) ContentType(contentType string) *Reply {
	if len(r.ContType) == 0 {
		r.ContType = strings.ToLower(contentType)
	}
	return r
}

// JSON method renders given data as JSON response
// and it sets HTTP 'Content-Type' as 'application/json; charset=utf-8'.
func (r *Reply) JSON(data interface{}) *Reply {
	r.ContentType(ahttp.ContentTypeJSON.String())
	r.Render(&jsonRender{Data: data})
	return r
}

// JSONSecure method renders given data as Secure JSON into response.
// and it sets HTTP 'Content-Type' as 'application/json; charset=utf-8'.
//
// See config `render.secure_json.prefix`.
func (r *Reply) JSONSecure(data interface{}) *Reply {
	r.ContentType(ahttp.ContentTypeJSON.String())
	r.Render(&secureJSONRender{Data: data, Prefix: r.ctx.a.secureJSONPrefix})
	return r
}

// JSONP method renders given data as JSONP response with callback
// and it sets HTTP 'Content-Type' as 'application/javascript; charset=utf-8'.
func (r *Reply) JSONP(data interface{}, callback string) *Reply {
	r.ContentType(ahttp.ContentTypeJavascript.String())
	r.Render(&jsonpRender{Data: data, Callback: callback})
	return r
}

// XML method renders given data as XML response and it sets
// HTTP Content-Type as 'application/xml; charset=utf-8'.
func (r *Reply) XML(data interface{}) *Reply {
	r.ContentType(ahttp.ContentTypeXML.String())
	r.Render(&xmlRender{Data: data})
	return r
}

// Text method renders given data as Plain Text response with given values
// and it sets HTTP Content-Type as 'text/plain; charset=utf-8'.
func (r *Reply) Text(format string, values ...interface{}) *Reply {
	r.ContentType(ahttp.ContentTypePlainText.String())
	r.Render(&textRender{Format: format, Values: values})
	return r
}

// Binary method writes given bytes into response. It auto-detects the
// content type of the given bytes if header `Content-Type` is not set.
func (r *Reply) Binary(b []byte) *Reply {
	return r.FromReader(bytes.NewReader(b))
}

// FromReader method reads the data from given reader and writes into response.
// It auto-detects the content type of the file if `Content-Type` is not set.
//
// Note: Method will close the reader after serving if it's satisfies the `io.Closer`.
func (r *Reply) FromReader(reader io.Reader) *Reply {
	r.Render(&binaryRender{Reader: reader})
	return r
}

// File method send the given as file to client. It auto-detects the content type
// of the file if `Content-Type` is not set.
//
// Note: If give filepath is relative path then application base directory is used
// as prefix.
func (r *Reply) File(file string) *Reply {
	if !filepath.IsAbs(file) {
		file = filepath.Join(r.ctx.a.BaseDir(), file)
	}
	r.Render(&binaryRender{Path: file})
	return r
}

// FileDownload method send the given as file to client as a download.
// It sets the `Content-Disposition` as `attachment` with given target name and
// auto-detects the content type of the file if `Content-Type` is not set.
func (r *Reply) FileDownload(file, targetName string) *Reply {
	r.Header(ahttp.HeaderContentDisposition, "attachment; filename="+targetName)
	return r.File(file)
}

// FileInline method send the given as file to client to display.
// For e.g.: display within the browser. It sets the `Content-Disposition` as
// `inline` with given target name and auto-detects the content type of
// the file if `Content-Type` is not set.
func (r *Reply) FileInline(file, targetName string) *Reply {
	r.Header(ahttp.HeaderContentDisposition, "inline; filename="+targetName)
	return r.File(file)
}

// HTML method renders given data with auto mapped template name and layout
// by framework. Also it sets HTTP 'Content-Type' as 'text/html; charset=utf-8'.
//
// aah renders the view template based on -
//
// 1) path 'Namespace/Sub-package' of Controller,
//
// 2) path 'Controller.Action',
//
// 3) view extension 'view.ext' and
//
// 4) case sensitive 'view.case_sensitive' from aah.conf
//
// 5) default layout is 'master.html'
//
//    For e.g.:
//      Namespace/Sub-package: frontend
//      Controller: App
//      Action: Login
//      view.ext: html
//
//      Outcome view template path => /views/pages/frontend/app/login.html
func (r *Reply) HTML(data Data) *Reply {
	return r.HTMLlf("", "", data)
}

// HTMLl method renders based on given layout and data. Refer `Reply.HTML(...)`
// method.
func (r *Reply) HTMLl(layout string, data Data) *Reply {
	return r.HTMLlf(layout, "", data)
}

// HTMLf method renders based on given filename and data. Refer `Reply.HTML(...)`
// method.
func (r *Reply) HTMLf(filename string, data Data) *Reply {
	return r.HTMLlf("", filename, data)
}

// HTMLlf method renders based on given layout, filename and data. Refer `Reply.HTML(...)`
// method.
func (r *Reply) HTMLlf(layout, filename string, data Data) *Reply {
	r.ContentType(ahttp.ContentTypeHTML.String())
	r.Render(&htmlRender{Layout: layout, Filename: filename, ViewArgs: data})
	return r
}

// Redirect method redirects to given redirect URL with status 302.
func (r *Reply) Redirect(redirectURL string) *Reply {
	return r.RedirectWithStatus(redirectURL, http.StatusFound)
}

// RedirectWithStatus method redirects to given redirect URL and status code.
func (r *Reply) RedirectWithStatus(redirectURL string, code int) *Reply {
	r.redirect = true
	r.Status(code)
	r.path = redirectURL
	return r
}

// Error method is used send an error reply, which is handled by aah error handling
// mechanism.
//
// More Info: https://docs.aahframework.org/error-handling.html
func (r *Reply) Error(err *Error) *Reply {
	r.err = err
	return r
}

// Render method is used render custom implementation using interface `aah.Render`.
func (r *Reply) Render(rdr Render) *Reply {
	r.Rdr = rdr
	return r
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Reply methods
//______________________________________________________________________________

// Header method sets the given header and value for the response.
// If value == "", then this method deletes the header.
//
// Note: It overwrites existing header value if it's present.
func (r *Reply) Header(key, value string) *Reply {
	if len(value) == 0 {
		if key == ahttp.HeaderContentType {
			return r.ContentType("")
		}

		r.ctx.Res.Header().Del(key)
	} else {
		if key == ahttp.HeaderContentType {
			return r.ContentType(value)
		}

		r.ctx.Res.Header().Set(key, value)
	}

	return r
}

// HeaderAppend method appends the given header and value for the response.
//
// Note: It just appends to it. It does not overwrite existing header.
func (r *Reply) HeaderAppend(key, value string) *Reply {
	if key == ahttp.HeaderContentType {
		return r.ContentType(value)
	}

	r.ctx.Res.Header().Add(key, value)
	return r
}

// Done method is used to indicate response has already been written using
// `aah.Context.Res` so no further action is needed from framework.
//
// Note: Framework doesn't intervene with response if this method called
// by aah user.
func (r *Reply) Done() *Reply {
	r.done = true
	return r
}

// Cookie method adds the give HTTP cookie into response.
func (r *Reply) Cookie(cookie *http.Cookie) *Reply {
	if r.cookies == nil {
		r.cookies = make([]*http.Cookie, 0)
	}

	r.cookies = append(r.cookies, cookie)
	return r
}

// DisableGzip method allows you disable Gzip for the reply. By default every
// response is gzip compressed if the client supports it and gzip enabled in
// app config.
func (r *Reply) DisableGzip() *Reply {
	r.gzip = false
	return r
}

// IsContentTypeSet method returns true if Content-Type is set otherwise
// false.
func (r *Reply) IsContentTypeSet() bool {
	return len(r.ContType) > 0
}

// Body method returns the response body buffer.
//
//    It might be nil if the -
//
//      1) Response was written successfully on the wire
//
//      2) Response is not yet rendered
//
//      3) Static files, since response is written via `http.ServeContent`
func (r *Reply) Body() *bytes.Buffer {
	return r.body
}

func (r *Reply) isHTML() bool {
	return ahttp.ContentTypeHTML.IsEqual(r.ContType)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//______________________________________________________________________________

// newReply method returns the new instance on reply builder.
func newReply(ctx *Context) *Reply {
	return &Reply{
		Code: http.StatusOK,
		gzip: true,
		ctx:  ctx,
	}
}

func acquireBuffer() *bytes.Buffer {
	return bufPool.Get().(*bytes.Buffer)
}

func releaseBuffer(b *bytes.Buffer) {
	if b != nil {
		b.Reset()
		bufPool.Put(b)
	}
}
