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
	// KeyViewArgRequestParams key name is used to store HTTP Request Params instance
	// into `ViewArgs`.
	KeyViewArgRequestParams = "_aahRequestParams"

	keyOverrideI18nName = "lang"
)

var (
	keyQueryParamName    = keyOverrideI18nName
	keyPathParamName     = keyOverrideI18nName
	isAcceptedExists     bool
	acceptedContentTypes []string
	isOfferedExists      bool
	offeredContentTypes  []string
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Params Unexported method
//___________________________________

// parseRequestParams method parses the incoming HTTP request to collects request
// parameters (Payload, Form, Query, Multi-part) stores into context. Request
// params are made available in View via template functions.
func (e *engine) parseRequestParams(ctx *Context) flowResult {
	// Content Negotitaion - Accepted, refer to Github #75
	if isAcceptedExists && !ess.IsSliceContainsString(acceptedContentTypes, ctx.Req.ContentType.Mime) {
		log.Warnf("Content type '%v' not accepted by server", ctx.Req.ContentType.Mime)
		ctx.Reply().Error(&Error{
			Code:    http.StatusUnsupportedMediaType,
			Message: http.StatusText(http.StatusUnsupportedMediaType),
		})
		return flowStop
	}

	// Content Negotitaion - Offered, refer to Github #75
	if isOfferedExists && !ess.IsSliceContainsString(offeredContentTypes, ctx.Req.AcceptContentType.Mime) {
		log.Warnf("Content type '%v' not offered by server", ctx.Req.AcceptContentType.Mime)
		ctx.Reply().Error(&Error{
			Code:    http.StatusNotAcceptable,
			Message: http.StatusText(http.StatusNotAcceptable),
		})
		return flowStop
	}

	req := ctx.Req.Unwrap()
	if ctx.Req.Method != ahttp.MethodGet {
		contentType := ctx.Req.ContentType.Mime
		log.Debugf("Request content type: %s", contentType)

		// Prevent DDoS attacks by large HTTP request bodies by enforcing
		// configured hard limit, Github #83.
		if contentType != ahttp.ContentTypeMultipartForm.Mime {
			req.Body = http.MaxBytesReader(ctx.Res, req.Body,
				firstNonZeroInt64(ctx.route.MaxBodySize, appMaxBodyBytesSize))
		}

		// TODO add support for content-type restriction and return 415 status for that
		// TODO HTML sanitizer for Form and Multipart Form

		switch contentType {
		case ahttp.ContentTypeJSON.Mime, ahttp.ContentTypeJSONText.Mime,
			ahttp.ContentTypeXML.Mime, ahttp.ContentTypeXMLText.Mime:
			if payloadBytes, err := ioutil.ReadAll(req.Body); err == nil {
				ctx.Req.Payload = payloadBytes
			} else {
				log.Errorf("unable to read request body for '%s': %s", contentType, err)
				ctx.Reply().Error(&Error{
					Code:    http.StatusBadRequest,
					Message: http.StatusText(http.StatusBadRequest),
				})
				return flowStop
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
		if req.MultipartForm != nil {
			defer func(r *http.Request) {
				log.Debug("multipart form file clean up")
				if err := r.MultipartForm.RemoveAll(); err != nil {
					log.Error(err)
				}
			}(req)
		}
	}

	if AppI18n() != nil {
		// i18n locale HTTP header `Accept-Language` value override via
		// Path Variable and URL Query Param (config i18n { param_name { ... } }).
		// Note: Query parameter takes precedence of all.
		// Default parameter name is `lang`
		if locale := firstNonZeroString(
			ctx.Req.QueryValue(keyQueryParamName),
			ctx.Req.PathValue(keyPathParamName)); !ess.IsStrEmpty(locale) {
			ctx.Req.Locale = ahttp.NewLocale(locale)
		}

		// All the request parameters made available to templates via funcs.
		ctx.AddViewArg(KeyViewArgRequestParams, ctx.Req.Params)
	}

	return flowCont
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Template methods
//___________________________________

// tmplPathParam method returns Request Path Param value for the given key.
func tmplPathParam(viewArgs map[string]interface{}, key string) interface{} {
	params := viewArgs[KeyViewArgRequestParams].(*ahttp.Params)
	return sanatizeValue(params.PathValue(key))
}

// tmplFormParam method returns Request Form value for the given key.
func tmplFormParam(viewArgs map[string]interface{}, key string) interface{} {
	params := viewArgs[KeyViewArgRequestParams].(*ahttp.Params)
	return sanatizeValue(params.FormValue(key))
}

// tmplQueryParam method returns Request Query String value for the given key.
func tmplQueryParam(viewArgs map[string]interface{}, key string) interface{} {
	params := viewArgs[KeyViewArgRequestParams].(*ahttp.Params)
	return sanatizeValue(params.QueryValue(key))
}

func paramInitialize(e *Event) {
	cfg := AppConfig()
	keyPathParamName = cfg.StringDefault("i18n.param_name.path", keyOverrideI18nName)
	keyQueryParamName = cfg.StringDefault("i18n.param_name.query", keyOverrideI18nName)
	acceptedContentTypes, isAcceptedExists = cfg.StringList("request.content_negotiation.accepted")
	offeredContentTypes, isOfferedExists = cfg.StringList("request.content_negotiation.offered")
}

func init() {
	OnStart(paramInitialize)
}
