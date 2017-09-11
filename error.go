// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"fmt"
	"html/template"
	"strings"

	"aahframework.org/ahttp.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0-unstable"
)

var errorHandler ErrorHandler

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
		Code    int         `json:"code,omitempty" xml:"code,omitempty"`
		Message string      `json:"message,omitempty" xml:"message,omitempty"`
		Data    interface{} `json:"data,omitempty" xml:"data,omitempty"`
	}

	// ErrorHandler is function type used to register centralized error handling
	// in aah framework.
	ErrorHandler func(ctx *Context, err *Error) bool
)

// SetErrorHandler method is used to register centralized application error
// handling. If custom handler is not then default error handler is used.
func SetErrorHandler(handler ErrorHandler) {
	if handler != nil {
		log.Infof("Custom centralized application error handler registered: %v", funcName(handler))
		errorHandler = handler
	}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported package methods
//___________________________________

// handleError method is aah centralized error handler.
func handleError(ctx *Context, err *Error) {
	if errorHandler == nil {
		defaultErrorHandler(ctx, err)
	} else {
		if !errorHandler(ctx, err) {
			defaultErrorHandler(ctx, err)
		}
	}
}

// defaultErrorHandler method is used when custom error handler is not register
// in the aah. It writes the response based on HTTP Content-Type.
func defaultErrorHandler(ctx *Context, err *Error) bool {
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
		if appViewEngine != nil {
			tmpl, er := appViewEngine.Get("", "errors", html.Filename)
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
