// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"io"
	"net/http"

	"aahframework.org/ahttp.v0"
	"aahframework.org/essentials.v0"
)

// Reply gives you control and convenient way to write a response effectively.
type Reply struct {
	Code        int
	ContType    string
	Hdr         http.Header
	Rdr         Render
	redirect    bool
	redirectURL string
	done        bool
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Global methods
//___________________________________

// NewReply method returns the new instance on reply builder.
func NewReply() *Reply {
	return &Reply{
		Hdr: http.Header{},
	}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Reply methods - Code Codes
//___________________________________

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

// Conflict method sets the HTTP Code as 409  RFC 7231, 6.5.8.
func (r *Reply) Conflict() *Reply {
	return r.Status(http.StatusConflict)
}

// InternalServerError method sets the HTTP Code as 500 RFC 7231, 6.6.1.
func (r *Reply) InternalServerError() *Reply {
	return r.Status(http.StatusInternalServerError)
}

// ServiceUnavailable method sets the HTTP Code as 503 RFC 7231, 6.6.4.
func (r *Reply) ServiceUnavailable() *Reply {
	return r.Status(http.StatusServiceUnavailable)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Reply methods - Content Types
//___________________________________

// ContentType method sets given Content-Type string for the response.
// Also Reply instance provides easy to use method for very frequently used
// Content-Type(s).
//
// By default aah framework try to determine response 'Content-Type' from
// 'ahttp.Request.AcceptContentType'.
func (r *Reply) ContentType(contentType string) *Reply {
	r.ContType = contentType
	return r
}

// JSON method renders given data as JSON response.
// Also it sets HTTP 'Content-Type' as 'application/json; charset=utf-8'.
// Response rendered pretty if 'render.pretty' is true.
func (r *Reply) JSON(data interface{}) *Reply {
	r.Rdr = &JSON{Data: data}
	r.ContentType(ahttp.ContentTypeJSON.Raw())
	return r
}

// JSONP method renders given data as JSONP response with callback.
// Also it sets HTTP 'Content-Type' as 'application/json; charset=utf-8'.
// Response rendered pretty if 'render.pretty' is true.
func (r *Reply) JSONP(data interface{}, callback string) *Reply {
	r.Rdr = &JSON{Data: data, IsJSONP: true, Callback: callback}
	r.ContentType(ahttp.ContentTypeJSON.Raw())
	return r
}

// XML method renders given data as XML response. Also it sets
// HTTP Content-Type as 'application/xml; charset=utf-8'.
// Response rendered pretty if 'render.pretty' is true.
func (r *Reply) XML(data interface{}) *Reply {
	r.Rdr = &XML{Data: data}
	r.ContentType(ahttp.ContentTypeXML.Raw())
	return r
}

// Text method renders given data as Plain Text response with given values.
// Also it sets HTTP Content-Type as 'text/plain; charset=utf-8'.
func (r *Reply) Text(format string, values ...interface{}) *Reply {
	r.Rdr = &Text{Format: format, Values: values}
	r.ContentType(ahttp.ContentTypePlainText.Raw())
	return r
}

// Bytes method writes the given bytes into response with given 'Content-Type'.
func (r *Reply) Bytes(contentType string, data []byte) *Reply {
	r.Rdr = &Bytes{Data: data}
	r.ContentType(contentType)
	return r
}

// File method writes the given file into response and close the file
// after write. Also it sets HTTP 'Content-Type' as 'application/octet-stream'
// and adds the header 'Content-Disposition' as 'attachment' with given filename.
// Note: Method does close the given 'io.ReadCloser' after writing a response.
func (r *Reply) File(filename string, file io.ReadCloser) *Reply {
	r.Header(ahttp.HeaderContentDisposition, "attachment; filename="+filename)
	return r.fileReply(filename, file)
}

// FileInline method writes the given file into response and close the file
// after write. Also it sets HTTP 'Content-Type' as 'application/octet-stream'
// and adds the header 'Content-Disposition' as 'inline' with given filename.
// Note: Method does close the given 'io.ReadCloser' after writing a response.
func (r *Reply) FileInline(filename string, file io.ReadCloser) *Reply {
	r.Header(ahttp.HeaderContentDisposition, "inline; filename="+filename)
	return r.fileReply(filename, file)
}

// HTML method renders given data with auto mapped template name and layout
// by framework. Also it sets HTTP 'Content-Type' as 'text/html; charset=utf-8'.
// By default aah framework renders the template based on
// 1) path 'Controller.Action',
// 2) template extension 'template.ext' and
// 3) case sensitive 'template.case_sensitive' from aah.conf
// 4) default layout is 'master'
//    E.g.:
//      Controller: App
//      Action: Login
//      template.ext: html
//
//      template => /views/pages/app/login.html
//               => /views/pages/App/Login.html
//
func (r *Reply) HTML(data Data) *Reply {
	r.Rdr = &HTML{
		ViewArgs: data,
	}
	r.ContentType(ahttp.ContentTypeHTML.Raw())
	return r
}

// HTMLl method renders based on given layout and data. Refer `Reply.HTML(...)`
// method.
func (r *Reply) HTMLl(layout string, data Data) *Reply {
	r.Rdr = &HTML{
		Layout:   layout,
		ViewArgs: data,
	}
	r.ContentType(ahttp.ContentTypeHTML.Raw())
	return r
}

// HTMLlf method renders based on given layout, filename and data. Refer `Reply.HTML(...)`
// method.
func (r *Reply) HTMLlf(layout, filename string, data Data) *Reply {
	r.Rdr = &HTML{
		Layout:   layout,
		Filename: filename,
		ViewArgs: data,
	}
	r.ContentType(ahttp.ContentTypeHTML.Raw())
	return r
}

// Redirect method redirect the to given redirect URL with status 302 and it does
// not provide option for chain call.
func (r *Reply) Redirect(redirectURL string) {
	r.redirect = true
	r.Found()
	r.redirectURL = redirectURL
}

// Redirects method redirect the to given redirect URL and status code. It does
// not provide option for chain call.
func (r *Reply) Redirects(redirectURL string, code int) {
	r.redirect = true
	r.Status(code)
	r.redirectURL = redirectURL
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Reply methods
//___________________________________

// Header method sets the given header and value for the response.
// If value == "", then this method deletes the header.
// Note: It overwrites existing header value if it's present.
func (r *Reply) Header(key, value string) *Reply {
	if ess.IsStrEmpty(value) {
		if key == ahttp.HeaderContentType {
			return r.ContentType("")
		}

		r.Hdr.Del(key)
	} else {
		if key == ahttp.HeaderContentType {
			return r.ContentType(value)
		}

		r.Hdr.Set(key, value)
	}

	return r
}

// HeaderAppend method appends the given header and value for the response.
// Note: It does not overwrite existing header, it just appends to it.
func (r *Reply) HeaderAppend(key, value string) *Reply {
	if key == ahttp.HeaderContentType {
		return r.ContentType(value)
	}

	r.Hdr.Add(key, value)
	return r
}

// Done method concludes middleware flow, action flow by returning control over
// to framework and informing that reply has already been sent via `aahContext.Res`
// and that no further action is needed.
func (r *Reply) Done() *Reply {
	r.done = true
	return r
}

// IsContentTypeSet method returns true if Content-Type is set otherwise
// false.
func (r *Reply) IsContentTypeSet() bool {
	return !ess.IsStrEmpty(r.ContType)
}

// IsStatusSet method returns true if HTTP Code is set for the 'Reply'
// otherwise false.
func (r *Reply) IsStatusSet() bool {
	return r.Code != 0
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported Reply methods
//___________________________________

func (r *Reply) fileReply(filename string, file io.ReadCloser) *Reply {
	r.Rdr = &File{Data: file}
	r.ContentType(ahttp.ContentTypeOctetStream.Raw())
	return r
}
