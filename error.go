// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"errors"
	"fmt"
	"html/template"
	"reflect"
	"strings"

	"aahframework.org/ahttp.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
)

// aah errors
var (
	ErrPanicRecovery              = errors.New("aah: panic recovery")
	ErrDomainNotFound             = errors.New("aah: domain not found")
	ErrRouteNotFound              = errors.New("aah: route not found")
	ErrStaticFileNotFound         = errors.New("aah: static file not found")
	ErrControllerOrActionNotFound = errors.New("aah: controller or action not found")
	ErrInvalidRequestParameter    = errors.New("aah: invalid request parameter")
	ErrContentTypeNotAccepted     = errors.New("aah: content type not accepted")
	ErrContentTypeNotOffered      = errors.New("aah: content type not offered")
	ErrHTTPMethodNotAllowed       = errors.New("aah: http method not allowed")
	ErrAccessDenied               = errors.New("aah: access denied")
	ErrAuthenticationFailed       = errors.New("aah: authentication failed")
	ErrGeneric                    = errors.New("aah: generic error")
	ErrValidation                 = errors.New("aah: validation error")
)

var errorHandlerFunc ErrorHandlerFunc

var defaultErrorHTMLTemplate = template.Must(template.New("error_template").Parse(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8" />
  <meta http-equiv="X-UA-Compatible" content="IE=edge" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>{{ .Error.Code }} {{ .Error.Message }}</title>
  <link href="//fonts.googleapis.com/css?family=Open+Sans:300,400,700" rel="stylesheet" type="text/css">
  <style>
    html {-ms-text-size-adjust:100%;-webkit-text-size-adjust:100%}
    html, body {
      margin: 0;
      background-color: #fff;
      color: #636b6f;
      font-family: 'Open Sans', sans-serif;
      font-weight: 100;
      height: 80vh;
    }
    .container {
      align-items: center;
      display: flex;
      justify-content: center;
      position: relative;
      height: 80vh;
    }
    .content {
      text-align: center;
    }
    .title {
      font-size: 36px;
      font-weight: bold;
      padding: 20px;
    }
  </style>
  </head>
  <body>
    <div class="container">{{ with .Error }}
      <div class="content">
        <div class="title">
          {{ .Code }} {{ .Message }}
        </div>
      </div>{{ end }}
    </div>
  </body>
</html>
`))

type (
	// Error structure used to represent the error details in the aah framework.
	Error struct {
		Reason  error       `json:"-" xml:"-"`
		Code    int         `json:"code,omitempty" xml:"code,omitempty"`
		Message string      `json:"message,omitempty" xml:"message,omitempty"`
		Data    interface{} `json:"data,omitempty" xml:"data,omitempty"`
	}

	// ErrorHandlerFunc is function type, it used to define centralized error handler
	// for your application.
	//
	//  - Return `true`, if you have handled your errors, aah just writes the reply on the wire.
	//
	//  - Return `false`, you may or may not handled the error, aah would propagate the error further to default
	// error handler.
	ErrorHandlerFunc func(ctx *Context, err *Error) bool

	// ErrorHandler is interface for implement controller level error handling
	ErrorHandler interface {
		// HandleError method is to handle error on your controller
		//
		//  - Return `true`, if you have handled your errors, aah just writes the reply on the wire.
		//
		//  - Return `false`, aah would propagate the error further to centralized
		// error handler, if not handled and then finally default error handler would take place.
		HandleError(err *Error) bool
	}
)

// SetErrorHandler method is used to register centralized application error
// handling. If custom handler is not then default error handler is used.
func SetErrorHandler(handlerFunc ErrorHandlerFunc) {
	if handlerFunc != nil {
		log.Infof("Custom centralized application error handler registered: %v", funcName(handlerFunc))
		errorHandlerFunc = handlerFunc
	}
}

// Error method is to comply error interface.
func (e *Error) Error() string {
	return fmt.Sprintf("%v, code '%v', message '%s'", e.Reason, e.Code, e.Message)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported package methods
//___________________________________

// handleError method is aah centralized error handler.
func handleError(ctx *Context, err *Error) {
	// GitHub #132 Call Controller error handler if exists
	if target := reflect.ValueOf(ctx.target); target.IsValid() {
		if eh, ok := target.Interface().(ErrorHandler); ok {
			ctx.Log().Trace("Calling controller error handler")
			if eh.HandleError(err) {
				return
			}
		}
	}

	// Call Centralized error handler if registered
	if errorHandlerFunc != nil {
		ctx.Log().Trace("Calling centralized error handler")
		if errorHandlerFunc(ctx, err) {
			return
		}
	}

	// Call Default error handler
	ctx.Log().Trace("Calling default error handler")
	defaultErrorHandlerFunc(ctx, err)
}

// defaultErrorHandlerFunc method is used when custom error handler is not register
// in the aah. It writes the response based on HTTP Content-Type.
func defaultErrorHandlerFunc(ctx *Context, err *Error) bool {
	ct := ctx.Reply().ContType
	if ess.IsStrEmpty(ct) {
		if ict := identifyContentType(ctx); ict != nil {
			ct = ict.Mime
		}
	} else if idx := strings.IndexByte(ct, ';'); idx > 0 {
		ct = ct[:idx]
	}

	// Set HTTP response code
	ctx.Reply().Status(err.Code)

	switch ct {
	case ahttp.ContentTypeJSON.Mime, ahttp.ContentTypeJSONText.Mime:
		ctx.Reply().JSON(err)
	case ahttp.ContentTypeXML.Mime, ahttp.ContentTypeXMLText.Mime:
		ctx.Reply().XML(err)
	case ahttp.ContentTypeHTML.Mime:
		html := acquireHTML()
		html.Filename = fmt.Sprintf("%d%s", err.Code, appViewExt)
		if AppViewEngine() != nil {
			tmpl, er := AppViewEngine().Get("", "errors", html.Filename)
			if tmpl == nil || er != nil {
				html.Template = defaultErrorHTMLTemplate
			} else {
				html.Template = tmpl
			}
		} else {
			html.Template = defaultErrorHTMLTemplate
		}
		html.ViewArgs = Data{"Error": err}
		addFrameworkValuesIntoViewArgs(ctx, html)
		ctx.Reply().Rdr = html
	default:
		ctx.Reply().Text("%d - %s", err.Code, err.Message)
	}
	return true
}
