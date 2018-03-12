// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"aahframework.org/ahttp.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/valpar.v0"
)

const (
	// KeyViewArgRequestParams key name is used to store HTTP Request Params instance
	// into `ViewArgs`.
	KeyViewArgRequestParams = "_aahRequestParams"

	keyOverrideI18nName = "lang"
	allContentTypes     = "*/*"
)

var (
	keyQueryParamName           = keyOverrideI18nName
	keyPathParamName            = keyOverrideI18nName
	requestParsers              = make(map[string]requestParser)
	isContentNegotiationEnabled bool
	acceptedContentTypes        []string
	offeredContentTypes         []string
	autobindPriority            []string

	errInvalidParsedValue = errors.New("aah: parsed value is invalid")
)

type requestParser func(ctx *Context) flowResult

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package method
//___________________________________

// AddValueParser method adds given custom value parser for the `reflect.Type`
func AddValueParser(typ reflect.Type, parser valpar.Parser) error {
	return valpar.AddValueParser(typ, parser)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Bind Middleware
//___________________________________

// BindMiddleware method parses the incoming HTTP request to collects request
// parameters (Path, Form, Query, Multipart) stores into context. Request
// params are made available in View via template functions.
func BindMiddleware(ctx *Context, m *Middleware) {
	if AppI18n() != nil {
		// i18n locale HTTP header `Accept-Language` value override via
		// Path Variable and URL Query Param (config i18n { param_name { ... } }).
		// Note: Query parameter takes precedence of all.
		if locale := firstNonZeroString(
			ctx.Req.QueryValue(keyQueryParamName),
			ctx.Req.PathValue(keyPathParamName)); !ess.IsStrEmpty(locale) {
			ctx.Req.Locale = ahttp.NewLocale(locale)
		}
	}

	if ctx.Req.Method == ahttp.MethodGet {
		goto PCont
	}

	ctx.Log().Debugf("Request Content-Type mime: %s", ctx.Req.ContentType.Mime)

	// Content Negotitaion - Accepted & Offered, refer to GitHub #75
	if isContentNegotiationEnabled {
		if len(acceptedContentTypes) > 0 &&
			!ess.IsSliceContainsString(acceptedContentTypes, ctx.Req.ContentType.Mime) {
			ctx.Log().Warnf("Content type '%v' not accepted by server", ctx.Req.ContentType.Mime)
			ctx.Reply().Error(&Error{
				Reason:  ErrContentTypeNotAccepted,
				Code:    http.StatusUnsupportedMediaType,
				Message: http.StatusText(http.StatusUnsupportedMediaType),
			})
			return
		}

		if len(offeredContentTypes) > 0 &&
			!ess.IsSliceContainsString(offeredContentTypes, ctx.Req.AcceptContentType.Mime) {
			ctx.Reply().Error(&Error{
				Reason:  ErrContentTypeNotOffered,
				Code:    http.StatusNotAcceptable,
				Message: http.StatusText(http.StatusNotAcceptable),
			})
			ctx.Log().Warnf("Content type '%v' not offered by server", ctx.Req.AcceptContentType.Mime)
			return
		}
	}

	// Prevent DDoS attacks by large HTTP request bodies by enforcing
	// configured hard limit, GitHub #83.
	if ctx.Req.ContentType.Mime != ahttp.ContentTypeMultipartForm.Mime {
		ctx.Req.Unwrap().Body = http.MaxBytesReader(ctx.Res, ctx.Req.Unwrap().Body,
			firstNonZeroInt64(ctx.route.MaxBodySize, appMaxBodyBytesSize))
	}

	// Parse request content by Content-Type
	if parser, found := requestParsers[ctx.Req.ContentType.Mime]; found {
		if res := parser(ctx); res == flowStop {
			return
		}
	}

PCont:
	// Compose request details, we can log at the end of the request.
	if isDumpLogEnabled {
		ctx.Set(keyAahRequestDump, composeRequestDump(ctx))
	}

	m.Next(ctx)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Content Parser methods
//___________________________________

func multipartFormParser(ctx *Context) flowResult {
	if err := ctx.Req.Unwrap().ParseMultipartForm(appMultipartMaxMemory); err != nil {
		ctx.Log().Errorf("Unable to parse multipart form: %s", err)
	} else {
		ctx.Req.Params.Form = ctx.Req.Unwrap().MultipartForm.Value
		ctx.Req.Params.File = ctx.Req.Unwrap().MultipartForm.File
	}
	return flowCont
}

func formParser(ctx *Context) flowResult {
	if err := ctx.Req.Unwrap().ParseForm(); err != nil {
		ctx.Log().Errorf("Unable to parse form: %s", err)
	} else {
		ctx.Req.Params.Form = ctx.Req.Unwrap().Form
	}
	return flowCont
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Action Parameters Auto Parse
//___________________________________

func parseParameters(ctx *Context) ([]reflect.Value, *Error) {
	paramCnt := len(ctx.action.Parameters)

	// If parameters not exists, return here
	if paramCnt == 0 {
		return emptyArg, nil
	}

	// Parse and Bind parameters
	params := createParams(ctx)
	var err error
	actionArgs := make([]reflect.Value, paramCnt)
	for idx, val := range ctx.action.Parameters {
		var result reflect.Value
		if vpFn, found := valpar.ValueParser(val.Type); found {
			result, err = vpFn(val.Name, val.Type, params)

			// GitHub #132 Validation implementation
			if rule, found := ctx.route.ValidationRule(val.Name); found {
				if !valpar.ValidateValue(result.Interface(), rule) {
					errMsg := fmt.Sprintf("Path param validation failed [name: %s, rule: %s, value: %v]",
						val.Name, rule, result.Interface())
					ctx.Log().Error(errMsg)
					return nil, &Error{
						Reason:  ErrValidation,
						Code:    http.StatusBadRequest,
						Message: http.StatusText(http.StatusBadRequest),
						Data:    errMsg,
					}
				}
			}
		} else if val.kind == reflect.Struct {
			ct := ctx.Req.ContentType.Mime
			if ct == ahttp.ContentTypeJSON.Mime || ct == ahttp.ContentTypeJSONText.Mime ||
				ct == ahttp.ContentTypeXML.Mime || ct == ahttp.ContentTypeXMLText.Mime {
				result, err = valpar.Body(ct, ctx.Req.Body(), val.Type)
				if isDumpLogEnabled && dumpRequestBody {
					addReqBodyIntoCtx(ctx, result)
				}
			} else {
				result, err = valpar.Struct("", val.Type, params)
			}
		}

		// check error
		if err != nil {
			if !result.IsValid() {
				ctx.Log().Errorf("Parsed result value is invalid or value parser not found [param: %s, type: %s]",
					val.Name, val.Type)
			}

			return nil, &Error{
				Reason:  ErrInvalidRequestParameter,
				Code:    http.StatusBadRequest,
				Message: http.StatusText(http.StatusBadRequest),
				Data:    err,
			}
		}

		// Apply Validation for type `struct`
		if val.kind == reflect.Struct {
			if errs, _ := valpar.Validate(result.Interface()); errs != nil {
				ctx.Log().Errorf("Param validation failed [name: %s, type: %s], Validation Errors:\n%v",
					val.Name, val.Type, errs.Error())

				return nil, &Error{
					Reason:  ErrValidation,
					Code:    http.StatusBadRequest,
					Message: http.StatusText(http.StatusBadRequest),
					Data:    errs,
				}
			}
		}

		// set action parameter value
		actionArgs[idx] = result
	}

	return actionArgs, nil
}

// Create param values based on autobind priority
func createParams(ctx *Context) url.Values {
	params := make(url.Values)
	for _, priority := range autobindPriority {
		switch priority {
		case "P": // Path Values
			for k, v := range ctx.Req.Params.Path {
				params.Set(k, v)
			}
		case "F": // Form Values
			for k, v := range ctx.Req.Params.Form {
				params[k] = v
			}
		case "Q": // Query Values
			for k, v := range ctx.Req.Params.Query {
				params[k] = v
			}
		}
	}
	return params
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

func bindInitialize(e *Event) {
	cfg := AppConfig()
	keyPathParamName = cfg.StringDefault("i18n.param_name.path", keyOverrideI18nName)
	keyQueryParamName = cfg.StringDefault("i18n.param_name.query", keyOverrideI18nName)

	// Content Negotitaion, GitHub #75
	isContentNegotiationEnabled = cfg.BoolDefault("request.content_negotiation.enable", false)
	acceptedContentTypes, _ = cfg.StringList("request.content_negotiation.accepted")
	for idx, v := range acceptedContentTypes {
		acceptedContentTypes[idx] = strings.ToLower(v)
		if v == allContentTypes {
			// when `*/*` is mentioned, don't check the condition
			// because it means every content type is allowed
			acceptedContentTypes = make([]string, 0)
			break
		}
	}

	offeredContentTypes, _ = cfg.StringList("request.content_negotiation.offered")
	for idx, v := range offeredContentTypes {
		offeredContentTypes[idx] = strings.ToLower(v)
		if v == allContentTypes {
			// when `*/*` is mentioned, don't check the condition
			// because it means every content type is allowed
			offeredContentTypes = make([]string, 0)
			break
		}
	}

	// Auto Parse and Bind, GitHub #26
	requestParsers[ahttp.ContentTypeMultipartForm.Mime] = multipartFormParser
	requestParsers[ahttp.ContentTypeForm.Mime] = formParser

	autobindPriority = reverseSlice(strings.Split(cfg.StringDefault("request.auto_bind.priority", "PFQ"), ""))
	timeFormats, found := cfg.StringList("format.time")
	if !found {
		timeFormats = []string{
			"2006-01-02T15:04:05Z07:00",
			"2006-01-02T15:04:05Z",
			"2006-01-02 15:04:05",
			"2006-01-02"}
	}
	valpar.TimeFormats = timeFormats
	valpar.StructTagName = cfg.StringDefault("request.auto_bind.tag_name", "bind")
}

func init() {
	OnStart(bindInitialize)
}
