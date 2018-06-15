// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"

	"aahframework.org/ahttp.v0"
	"aahframework.org/essentials.v0"
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
	ErrNotAuthenticated           = errors.New("aah: not authenticated")
	ErrAccessDenied               = errors.New("aah: access denied")
	ErrAuthenticationFailed       = errors.New("aah: authentication failed")
	ErrAuthorizationFailed        = errors.New("aah: authorization failed")
	ErrSessionAuthenticationInfo  = errors.New("aah: session authentication info")
	ErrUnableToGetPrincipal       = errors.New("aah: unable to get principal")
	ErrGeneric                    = errors.New("aah: generic error")
	ErrValidation                 = errors.New("aah: validation error")
	ErrRenderResponse             = errors.New("aah: render response error")
	ErrWriteResponse              = errors.New("aah: write response error")
)

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

// ErrorHandlerFunc is function type, it used to define centralized error handler
// for your application.
//
//  - Return `true`, if you have handled your errors, aah just writes the reply on the wire.
//
//  - Return `false`, you may or may not handled the error, aah would propagate
// 	the error further to default error handler.
type ErrorHandlerFunc func(ctx *Context, err *Error) bool

// ErrorHandler is interface for implement controller level error handling
type ErrorHandler interface {
	// HandleError method is to handle error on your controller
	//
	//  - Return `true`, if you have handled your errors, aah just writes the reply on the wire.
	//
	//  - Return `false`, aah would propagate the error further to centralized
	// 	  error handler, if not handled and then finally default error handler would take place.
	HandleError(err *Error) bool
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// app Unexported methods
//______________________________________________________________________________

func (a *app) initError() error {
	a.errorMgr = &errorManager{
		a: a,
	}
	return nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Error Manager
//______________________________________________________________________________

type errorManager struct {
	a           *app
	handlerFunc ErrorHandlerFunc
}

func (er *errorManager) SetHandler(handlerFn ErrorHandlerFunc) {
	if handlerFn != nil {
		er.handlerFunc = handlerFn
		er.a.Log().Infof("Custom centralized application error handler is registered with: %v", funcName(handlerFn))
	}
}

func (er *errorManager) Handle(ctx *Context) {
	// GitHub #132 Call Controller error handler if exists
	if ceh, ok := ctx.target.(ErrorHandler); ok {
		ctx.Log().Tracef("Calling controller error handler: %s.HandleError", ctx.controller.FqName)
		if ceh.HandleError(ctx.Reply().err) {
			return
		}
	}

	// Call Centralized error handler if registered
	if er.handlerFunc != nil {
		ctx.Log().Trace("Calling centralized error handler")
		if er.handlerFunc(ctx, ctx.Reply().err) {
			return
		}
	}

	// Call Default error handler
	ctx.Log().Trace("Calling default error handler")
	er.DefaultHandler(ctx, ctx.Reply().err)
}

// DefaultHandler method is used when custom error handler is not register
// in the aah. It writes the response based on HTTP Content-Type.
func (er *errorManager) DefaultHandler(ctx *Context, err *Error) bool {
	ct := ctx.Reply().ContType
	if ess.IsStrEmpty(ct) {
		ct = ctx.detectContentType().Mime
		if ctx.a.viewMgr == nil && ct == ahttp.ContentTypeHTML.Mime {
			ct = ahttp.ContentTypePlainText.Mime
		}
	}

	ct = stripCharset(ct)

	// Set HTTP response code
	ctx.Reply().Status(err.Code)

	// Set it to nil do not expose any app internal info
	err.Data = nil

	switch ct {
	case ahttp.ContentTypeJSON.Mime, ahttp.ContentTypeJSONText.Mime:
		ctx.Reply().JSON(err)
	case ahttp.ContentTypeXML.Mime, ahttp.ContentTypeXMLText.Mime:
		ctx.Reply().XML(err)
	case ahttp.ContentTypeHTML.Mime:
		html := &htmlRender{
			Template: defaultErrorHTMLTemplate,
			Filename: fmt.Sprintf("%d%s", err.Code, ctx.a.viewMgr.fileExt),
			ViewArgs: Data{"Error": err},
		}

		if ctx.a.viewMgr != nil {
			tmpl, terr := ctx.a.ViewEngine().Get("", "errors", html.Filename)
			if tmpl != nil || terr == nil {
				html.Template = tmpl
			}
		}

		ctx.Reply().Rdr = html
		ctx.a.viewMgr.addFrameworkValuesIntoViewArgs(ctx)
	default:
		ctx.Reply().Text("%d - %s", err.Code, err.Message)
	}
	return true
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Error type
//______________________________________________________________________________

// Error structure used to represent the error details in the aah framework.
type Error struct {
	Reason  error       `json:"-" xml:"-"`
	Code    int         `json:"code,omitempty" xml:"code,omitempty"`
	Message string      `json:"message,omitempty" xml:"message,omitempty"`
	Data    interface{} `json:"data,omitempty" xml:"data,omitempty"`
}

// Error method is to comply error interface.
func (e *Error) Error() string {
	return fmt.Sprintf("%v, code '%v', message '%s'", e.Reason, e.Code, e.Message)
}

func newError(err error, code int) *Error {
	return &Error{Reason: err, Code: code, Message: http.StatusText(code)}
}

func newErrorWithData(err error, code int, data interface{}) *Error {
	return &Error{Reason: err, Code: code, Message: http.StatusText(code), Data: data}
}
