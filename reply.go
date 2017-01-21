// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"net/http"

	"aahframework.org/aah/ahttp"
	"aahframework.org/aah/render"
	"aahframework.org/essentials"
)

// Reply gives you control and convenient way to write a response effectively.
type Reply struct {
	status      int
	contentType string
	header      http.Header
	render      render.Render
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Reply methods - Status Codes
//___________________________________

// Status method sets the HTTP status code for the response.
// Also Reply instance provides easy to use method for very frequently used
// HTTP Status Codes reference: http://www.restapitutorial.com/httpstatuscodes.html
func (r *Reply) Status(code int) *Reply {
	r.status = code
	return r
}

// Ok method sets the HTTP status as 200 RFC 7231, 6.3.1.
func (r *Reply) Ok() *Reply {
	return r.Status(http.StatusOK)
}

// Created method sets the HTTP status as 201 RFC 7231, 6.3.2.
func (r *Reply) Created() *Reply {
	return r.Status(http.StatusCreated)
}

// Accepted method sets the HTTP status as 202 RFC 7231, 6.3.3.
func (r *Reply) Accepted() *Reply {
	return r.Status(http.StatusAccepted)
}

// NoContent method sets the HTTP status as 204 RFC 7231, 6.3.5.
func (r *Reply) NoContent() *Reply {
	return r.Status(http.StatusNoContent)
}

// MovedPermanently method sets the HTTP status as 301 RFC 7231, 6.4.2.
func (r *Reply) MovedPermanently() *Reply {
	return r.Status(http.StatusMovedPermanently)
}

// TemporaryRedirect method sets the HTTP status as 307 RFC 7231, 6.4.7.
func (r *Reply) TemporaryRedirect() *Reply {
	return r.Status(http.StatusTemporaryRedirect)
}

// BadRequest method sets the HTTP status as 400 RFC 7231, 6.5.1.
func (r *Reply) BadRequest() *Reply {
	return r.Status(http.StatusBadRequest)
}

// Unauthorized method sets the HTTP status as 401 RFC 7235, 3.1.
func (r *Reply) Unauthorized() *Reply {
	return r.Status(http.StatusUnauthorized)
}

// Forbidden method sets the HTTP status as 403 RFC 7231, 6.5.3.
func (r *Reply) Forbidden() *Reply {
	return r.Status(http.StatusForbidden)
}

// NotFound method sets the HTTP status as 404 RFC 7231, 6.5.4.
func (r *Reply) NotFound() *Reply {
	return r.Status(http.StatusNotFound)
}

// MethodNotAllowed method sets the HTTP status as 405 RFC 7231, 6.5.5.
func (r *Reply) MethodNotAllowed() *Reply {
	return r.Status(http.StatusMethodNotAllowed)
}

// Conflict method sets the HTTP status as 409  RFC 7231, 6.5.8.
func (r *Reply) Conflict() *Reply {
	return r.Status(http.StatusConflict)
}

// InternalServerError method sets the HTTP status as 500 RFC 7231, 6.6.1.
func (r *Reply) InternalServerError() *Reply {
	return r.Status(http.StatusInternalServerError)
}

// ServiceUnavailable method sets the HTTP status as 503 RFC 7231, 6.6.4.
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
	r.contentType = contentType
	return r
}

// JSON method renders given data as JSON response.
// Also it sets HTTP 'Content-Type' as 'application/json; charset=utf-8'.
// Response rendered pretty if 'render.pretty' is true.
func (r *Reply) JSON(data interface{}) *Reply {
	r.render = &render.JSON{Data: data}
	r.ContentType(ahttp.ContentTypeJSON.Raw())
	return r
}

// JSONP method renders given data as JSONP response with callback.
// Also it sets HTTP 'Content-Type' as 'application/json; charset=utf-8'.
// Response rendered pretty if 'render.pretty' is true.
func (r *Reply) JSONP(data interface{}, callback string) *Reply {
	r.render = &render.JSON{Data: data, IsJSONP: true, Callback: callback}
	r.ContentType(ahttp.ContentTypeJSON.Raw())
	return r
}

// XML method renders given data as XML response. Also it sets
// HTTP Content-Type as 'application/xml; charset=utf-8'.
// Response rendered pretty if 'render.pretty' is true.
func (r *Reply) XML(data interface{}) *Reply {
	r.render = &render.XML{Data: data}
	r.ContentType(ahttp.ContentTypeXML.Raw())
	return r
}

// Text method renders given data as Plain Text response with given values.
// Also it sets HTTP Content-Type as 'text/plain; charset=utf-8'.
func (r *Reply) Text(format string, values ...interface{}) *Reply {
	r.render = &render.Text{Format: format, Values: values}
	r.ContentType(ahttp.ContentTypePlainText.Raw())
	return r
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Reply methods
//___________________________________

// Header method sets the given header and value for the response.
// If value == "", then this method deletes the header.
// Note: It overwrites existing header value if it's present.
func (r *Reply) Header(key, value string) *Reply {
	if ess.IsStrEmpty(value) {
		r.header.Del(key)
	} else {
		r.header.Set(key, value)
	}

	return r
}

// HeaderAppend method appends the given header and value for the response.
// Note: It does not overwrite existing header, it just appends to it.
func (r *Reply) HeaderAppend(key, value string) *Reply {
	r.header.Add(key, value)
	return r
}

// IsContentTypeSet method returns true if Content-Type is set otherwise
// false.
func (r *Reply) IsContentTypeSet() bool {
	return ess.IsStrEmpty(r.contentType)
}

// IsStatusSet method returns true if HTTP Status is set for the 'Reply'
// otherwise false.
func (r *Reply) IsStatusSet() bool {
	return r.status != 0
}
