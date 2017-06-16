// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"io/ioutil"
	"net/http"

	"aahframework.org/ahttp.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
)

const (
	keyRequestParams  = "RequestParams"
	keyOverrideLocale = "lang"
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Params method
//___________________________________

// parseRequestParams method parses the incoming HTTP request to collects request
// parameters (Payload, Form, Query, Multi-part) stores into context. Request
// params are made available in View via template functions.
func (e *engine) parseRequestParams(ctx *Context) {
	req := ctx.Req.Raw

	if ctx.Req.Method != ahttp.MethodGet {
		contentType := ctx.Req.ContentType.Mime
		log.Debugf("Request content type: %s", contentType)

		// TODO add support for content-type restriction and return 415 status for that
		// TODO HTML sanitizer for Form and Multipart Form

		switch contentType {
		case ahttp.ContentTypeJSON.Mime, ahttp.ContentTypeXML.Mime, ahttp.ContentTypeXMLText.Mime:
			if payloadBytes, err := ioutil.ReadAll(req.Body); err == nil {
				ctx.Req.Payload = payloadBytes
			} else {
				log.Errorf("unable to read request body for '%s': %s", contentType, err)
			}
		case ahttp.ContentTypeForm.Mime:
			if err := req.ParseForm(); err == nil {
				ctx.Req.Params.Form = req.Form
			} else {
				log.Errorf("unable to parse form: %s", err)
			}
		case ahttp.ContentTypeMultipartForm.Mime:
			if err := req.ParseMultipartForm(appMultipartMaxMemory); err == nil {
				ctx.Req.Params.Form = req.MultipartForm.Value
				ctx.Req.Params.File = req.MultipartForm.File
			} else {
				log.Errorf("unable to parse multipart form: %s", err)
			}
		} // switch end

		// clean up
		defer func(r *http.Request) {
			if r.MultipartForm != nil {
				log.Debug("multipart form file clean up")
				if err := r.MultipartForm.RemoveAll(); err != nil {
					log.Error(err)
				}
			}
		}(req)
	}

	// i18n option via the config value "i18n.url_param_name".
	// If that value is missing, we default to the `lang` query parameter
	queryParam := AppConfig().StringDefault("i18n.param_name.query", keyOverrideLocale)
	if lang := ctx.Req.QueryValue(queryParam); !ess.IsStrEmpty(lang) {
		ctx.Req.Locale = ahttp.NewLocale(lang)
	}

	// All the request parameters made available to templates via funcs.
	ctx.AddViewArg(keyRequestParams, ctx.Req.Params)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Template methods
//___________________________________

// tmplPathParam method returns Request Path Param value for the given key.
func tmplPathParam(viewArgs map[string]interface{}, key string) interface{} {
	params := viewArgs[keyRequestParams].(*ahttp.Params)
	return sanatizeValue(params.PathValue(key))
}

// tmplFormParam method returns Request Form value for the given key.
func tmplFormParam(viewArgs map[string]interface{}, key string) interface{} {
	params := viewArgs[keyRequestParams].(*ahttp.Params)
	return sanatizeValue(params.FormValue(key))
}

// tmplQueryParam method returns Request Query String value for the given key.
func tmplQueryParam(viewArgs map[string]interface{}, key string) interface{} {
	params := viewArgs[keyRequestParams].(*ahttp.Params)
	return sanatizeValue(params.QueryValue(key))
}
