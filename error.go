// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"strings"

	"aahframework.org/ahttp.v0"
	"aahframework.org/essentials.v0"
)

// writeError method writes the server error response based content type.
// It's handy internal method.
func writeErrorInfo(ctx *Context, code int, msg string) {
	ct := ctx.Reply().ContType
	if ess.IsStrEmpty(ct) {
		if ict := identifyContentType(ctx); ict != nil {
			ct = ict.Mime
		}
	} else if idx := strings.IndexByte(ct, ';'); idx > 0 {
		ct = ct[:idx]
	}

	switch ct {
	case ahttp.ContentTypeJSON.Mime, ahttp.ContentTypeJSONText.Mime:
		ctx.Reply().Status(code).JSON(Data{
			"code":    code,
			"message": msg,
		})
	case ahttp.ContentTypeXML.Mime, ahttp.ContentTypeXMLText.Mime:
		ctx.Reply().Status(code).XML(Data{
			"code":    code,
			"message": msg,
		})
	default:
		ctx.Reply().Status(code).Text("%d %s", code, msg)
	}
}
