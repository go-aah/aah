// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strings"

	"aahframework.org/ahttp.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/valpar.v0"
	"gopkg.in/go-playground/validator.v9"
)

const (
	// KeyViewArgRequest key name is used to store HTTP Request instance
	// into `ViewArgs`.
	KeyViewArgRequest = "_aahRequest"

	keyOverrideI18nName = "lang"
	allContentTypes     = "*/*"
)

var (
	emptyArg = make([]reflect.Value, 0)
)

type requestParser func(ctx *Context) flowResult

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package method
//______________________________________________________________________________

// AddValueParser method adds given custom value parser for the `reflect.Type`
func AddValueParser(typ reflect.Type, parser valpar.Parser) error {
	return valpar.AddValueParser(typ, parser)
}

// Validator method return the default validator of aah framework.
//
// Refer to https://godoc.org/gopkg.in/go-playground/validator.v9 for detailed
// documentation.
func Validator() *validator.Validate {
	return valpar.Validator()
}

// Validate method is to validate struct via underneath validator.
//
// Returns:
//
//  - For validation errors: returns `validator.ValidationErrors` and nil
//
//  - For invalid input: returns nil, error (invalid input such as nil, non-struct, etc.)
//
//  - For no validation errors: nil, nil
func Validate(s interface{}) (validator.ValidationErrors, error) {
	return valpar.Validate(s)
}

// ValidateValue method is to validate individual value on demand.
//
// Returns -
//
//  - true: validation passed
//
//  - false: validation failed
//
// For example:
//
// 	i := 15
// 	result := valpar.ValidateValue(i, "gt=1,lt=10")
//
// 	emailAddress := "sample@sample"
// 	result := valpar.ValidateValue(emailAddress, "email")
//
// 	numbers := []int{23, 67, 87, 23, 90}
// 	result := valpar.ValidateValue(numbers, "unique")
func ValidateValue(v interface{}, rules string) bool {
	return valpar.ValidateValue(v, rules)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Bind Middleware
//______________________________________________________________________________

// BindMiddleware method parses the incoming HTTP request to collects request
// parameters (Path, Form, Query, Multipart) stores into context. Request
// params are made available in View via template functions.
func BindMiddleware(ctx *Context, m *Middleware) {
	if ctx.a.I18n() != nil {
		// i18n locale HTTP header `Accept-Language` value override via
		// Path Variable and URL Query Param (config i18n { param_name { ... } }).
		// Note: Query parameter takes precedence of all.
		if locale := firstNonZeroString(
			ctx.Req.QueryValue(ctx.a.bindMgr.keyQueryParamName),
			ctx.Req.PathValue(ctx.a.bindMgr.keyPathParamName)); !ess.IsStrEmpty(locale) {
			ctx.Req.SetLocale(ahttp.NewLocale(locale))
		}
	}

	// Per https://tools.ietf.org/html/rfc7231#section-8.1.3
	if ctx.a.bindMgr.payloadSupported.MatchString(ctx.Req.Method) {
		ctx.Log().Debugf("Request Content-Type mime: %s", ctx.Req.ContentType())

		// Content Negotitaion - Accepted & Offered, refer to GitHub #75
		if ctx.a.bindMgr.contentNegotiationEnabled {
			if len(ctx.a.bindMgr.acceptedContentTypes) > 0 &&
				!ess.IsSliceContainsString(ctx.a.bindMgr.acceptedContentTypes, ctx.Req.ContentType().Mime) {
				ctx.Log().Warnf("Content type '%v' not accepted by server", ctx.Req.ContentType())
				ctx.Reply().Error(newError(ErrContentTypeNotAccepted, http.StatusUnsupportedMediaType))
				return
			}

			if len(ctx.a.bindMgr.offeredContentTypes) > 0 &&
				!ess.IsSliceContainsString(ctx.a.bindMgr.offeredContentTypes, ctx.Req.AcceptContentType().Mime) {
				ctx.Log().Warnf("Content type '%v' not offered by server", ctx.Req.AcceptContentType())
				ctx.Reply().Error(newError(ErrContentTypeNotOffered, http.StatusNotAcceptable))
				return
			}
		}

		// Prevent DDoS attacks by large HTTP request bodies by enforcing
		// configured hard limit for non-multipart/form-data Content-Type GitHub #83.
		if !ahttp.ContentTypeMultipartForm.IsEqual(ctx.Req.ContentType().Mime) {
			ctx.Req.Unwrap().Body = http.MaxBytesReader(ctx.Res, ctx.Req.Unwrap().Body,
				firstNonZeroInt64(ctx.route.MaxBodySize, ctx.a.maxBodyBytes))
		}

		// Set the tee reader if dump log enabled with request body enabled
		if ctx.a.dumpLogEnabled && ctx.a.dumpLog.logRequestBody {
			reqBuf := acquireBuffer()
			ctx.Req.Unwrap().Body = ioutil.NopCloser(io.TeeReader(ctx.Req.Unwrap().Body, reqBuf))
			ctx.Set(keyAahRequestBodyBuf, reqBuf)
		}

		// Parse request content by Content-Type
		if parser, found := ctx.a.bindMgr.requestParsers[ctx.Req.ContentType().Mime]; found {
			if res := parser(ctx); res == flowAbort {
				return
			}
		}
	}

	m.Next(ctx)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// app Unexported methods
//______________________________________________________________________________

func (a *app) initBind() error {
	cfg := a.Config()

	bindMgr := &bindManager{
		keyPathParamName:          cfg.StringDefault("i18n.param_name.path", keyOverrideI18nName),
		keyQueryParamName:         cfg.StringDefault("i18n.param_name.query", keyOverrideI18nName),
		contentNegotiationEnabled: cfg.BoolDefault("request.content_negotiation.enable", false),
		requestParsers:            make(map[string]requestParser),
		payloadSupported:          regexp.MustCompile(`(POST|PUT|DELETE)`),
	}

	// Content Negotitaion, GitHub #75
	bindMgr.acceptedContentTypes, _ = cfg.StringList("request.content_negotiation.accepted")
	for idx, v := range bindMgr.acceptedContentTypes {
		bindMgr.acceptedContentTypes[idx] = strings.ToLower(v)
		if v == allContentTypes {
			// when `*/*` is mentioned, don't check the condition
			// because it means every content type is allowed
			bindMgr.acceptedContentTypes = make([]string, 0)
			break
		}
	}

	bindMgr.offeredContentTypes, _ = cfg.StringList("request.content_negotiation.offered")
	for idx, v := range bindMgr.offeredContentTypes {
		bindMgr.offeredContentTypes[idx] = strings.ToLower(v)
		if v == allContentTypes {
			// when `*/*` is mentioned, don't check the condition
			// because it means every content type is allowed
			bindMgr.offeredContentTypes = make([]string, 0)
			break
		}
	}

	// Auto Parse and Bind, GitHub #26
	bindMgr.requestParsers[ahttp.ContentTypeMultipartForm.Mime] = multipartFormParser
	bindMgr.requestParsers[ahttp.ContentTypeForm.Mime] = formParser

	bindMgr.autobindPriority = reverseSlice(strings.Split(cfg.StringDefault("request.auto_bind.priority", "PFQ"), ""))
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

	a.bindMgr = bindMgr
	return nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Bind Manager
//______________________________________________________________________________

type bindManager struct {
	contentNegotiationEnabled bool
	keyQueryParamName         string
	keyPathParamName          string
	acceptedContentTypes      []string
	offeredContentTypes       []string
	autobindPriority          []string
	requestParsers            map[string]requestParser
	payloadSupported          *regexp.Regexp
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Content Parser methods
//______________________________________________________________________________

func multipartFormParser(ctx *Context) flowResult {
	if err := ctx.Req.Unwrap().ParseMultipartForm(ctx.a.multipartMaxMemory); err != nil {
		ctx.Log().Errorf("Unable to parse multipart form: %s", err)
	}
	return flowCont
}

func formParser(ctx *Context) flowResult {
	if err := ctx.Req.Unwrap().ParseForm(); err != nil {
		ctx.Log().Errorf("Unable to parse form: %s", err)
	}
	return flowCont
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Context - Action Parameters Auto Parse
//______________________________________________________________________________

func (ctx *Context) parseParameters() ([]reflect.Value, *Error) {
	paramCnt := len(ctx.action.Parameters)

	// If parameters not exists, return here
	if paramCnt == 0 {
		return emptyArg, nil
	}

	// Parse and Bind parameters
	params := ctx.createParams()
	var err error
	actionArgs := make([]reflect.Value, paramCnt)
	for idx, val := range ctx.action.Parameters {
		var result reflect.Value
		if vpFn, found := valpar.ValueParser(val.Type); found {
			result, err = vpFn(val.Name, val.Type, params)

			// GitHub #132 Validation implementation
			if rule, found := ctx.route.ValidationRule(val.Name); found {
				if !ValidateValue(result.Interface(), rule) {
					errMsg := fmt.Sprintf("Path param validation failed [name: %s, rule: %s, value: %v]",
						val.Name, rule, result.Interface())
					ctx.Log().Error(errMsg)
					return nil, newErrorWithData(ErrValidation, http.StatusBadRequest, errMsg)
				}
			}
		} else if val.Kind == reflect.Struct {
			ct := ctx.Req.ContentType().Mime
			if ct == ahttp.ContentTypeJSON.Mime || ct == ahttp.ContentTypeXML.Mime ||
				ct == ahttp.ContentTypeJSONText.Mime || ct == ahttp.ContentTypeXMLText.Mime {
				result, err = valpar.Body(ct, ctx.Req.Body(), val.Type)
			} else {
				result, err = valpar.Struct("", val.Type, params)
			}
		}

		// check error
		if err != nil {
			if !result.IsValid() {
				ctx.Log().Errorf("Parsed parameter value is invalid or value parser not found [param: %s, type: %s]",
					val.Name, val.Type)
			}
			return nil, newErrorWithData(ErrInvalidRequestParameter, http.StatusBadRequest, err)
		}

		// Apply Validation for type `struct`
		if val.Kind == reflect.Struct {
			if errs, _ := Validate(result.Interface()); errs != nil {
				ctx.Log().Errorf("Param validation failed [name: %s, type: %s], Validation Errors:\n%v",
					val.Name, val.Type, errs.Error())
				return nil, newErrorWithData(ErrValidation, http.StatusBadRequest, errs)
			}
		}

		// set action parameter value
		actionArgs[idx] = result
	}

	return actionArgs, nil
}

// Create param values based on autobind priority
func (ctx *Context) createParams() url.Values {
	params := make(url.Values)
	for _, priority := range ctx.a.bindMgr.autobindPriority {
		switch priority {
		case "P": // Path Values
			for k, v := range ctx.Req.PathParams {
				params.Set(k, v)
			}
		case "F": // Form Values
			for k, v := range ctx.Req.Unwrap().Form {
				params[k] = v
			}
		case "Q": // Query Values
			for k, v := range ctx.Req.URL().Query() {
				params[k] = v
			}
		}
	}
	return params
}
