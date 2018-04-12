// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"aahframework.org/ahttp.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
	"aahframework.org/router.v0"
	"aahframework.org/security.v0"
	"aahframework.org/security.v0/session"
)

var (
	ctxPtrType = reflect.TypeOf((*Context)(nil))

	errTargetNotFound = errors.New("target not found")
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Context
//______________________________________________________________________________

// Context type for aah framework, gets embedded in application controller.
//
// Note: this is not standard package `context.Context`.
type Context struct {
	// Req is HTTP request instance
	Req *ahttp.Request

	// Res is HTTP response writer compliant.
	//
	// Note 1: It is highly recommended to use `Reply()` builder for
	// composing your response.
	//
	// Note 2: If you're using `cxt.Res` directly, don't forget to call
	// `Reply().Done()`; so that aah will not intervene with your
	// response.
	Res ahttp.ResponseWriter

	a          *app
	e          *engine
	controller *controllerInfo
	action     *MethodInfo
	actionrv   reflect.Value
	target     interface{}
	targetrv   reflect.Value
	domain     *router.Domain
	route      *router.Route
	subject    *security.Subject
	reply      *Reply
	viewArgs   map[string]interface{}
	values     map[string]interface{}
	abort      bool
	decorated  bool
	logger     log.Loggerer
}

// Reply method gives you control and convenient way to write
// a response effectively.
func (ctx *Context) Reply() *Reply {
	if ctx.reply == nil {
		ctx.reply = newReply(ctx)
	}
	return ctx.reply
}

// ViewArgs method returns aah framework and request related info that can be
// used in template or view rendering, etc.
func (ctx *Context) ViewArgs() map[string]interface{} {
	return ctx.viewArgs
}

// AddViewArg method adds given key and value into `viewArgs`. These view args
// values accessible on templates. Chained call is possible.
func (ctx *Context) AddViewArg(key string, value interface{}) *Context {
	if ctx.viewArgs == nil {
		ctx.viewArgs = make(map[string]interface{})
	}
	ctx.viewArgs[key] = value
	return ctx
}

// ReverseURL method returns the URL for given route name and args.
// See `Domain.ReverseURL` for more information.
func (ctx *Context) ReverseURL(routeName string, args ...interface{}) string {
	domain, rn := ctx.a.findReverseURLDomain(ctx.Req.Host, routeName)
	return createReverseURL(ctx.Log(), domain, rn, nil, args...)
}

// ReverseURLm method returns the URL for given route name and key-value paris.
// See `Domain.ReverseURLm` for more information.
func (ctx *Context) ReverseURLm(routeName string, args map[string]interface{}) string {
	domain, rn := ctx.a.findReverseURLDomain(ctx.Req.Host, routeName)
	return createReverseURL(ctx.Log(), domain, rn, args)
}

// Msg method returns the i18n value for given key otherwise empty string returned.
func (ctx *Context) Msg(key string, args ...interface{}) string {
	return ctx.Msgl(ctx.Req.Locale(), key, args...)
}

// Msgl method returns the i18n value for given local and key otherwise
// empty string returned.
func (ctx *Context) Msgl(locale *ahttp.Locale, key string, args ...interface{}) string {
	return ctx.a.I18n().Lookup(locale, key, args...)
}

// Subdomain method returns the subdomain from the incoming request if available
// as per routes.conf. Otherwise empty string.
func (ctx *Context) Subdomain() string {
	if ctx.domain.IsSubDomain {
		if idx := strings.IndexByte(ctx.Req.Host, '.'); idx > 0 {
			return ctx.Req.Host[:idx]
		}
	}
	return ""
}

// Subject method the subject (aka application user) of current request.
func (ctx *Context) Subject() *security.Subject {
	if ctx.subject == nil {
		ctx.subject = security.AcquireSubject()
	}
	return ctx.subject
}

// Session method always returns `session.Session` object. Use `Session.IsNew`
// to identify whether sesison is newly created or restored from the request
// which was already created.
func (ctx *Context) Session() *session.Session {
	if ctx.Subject().Session == nil {
		ctx.subject.Session = ctx.a.SessionManager().NewSession()
	}
	return ctx.subject.Session
}

// Abort method sets the abort to true. It means framework will not proceed with
// next middleware, next interceptor or action based on context it being used.
// Contexts:
//    1) If it's called in the middleware, then middleware chain stops;
// framework starts processing response.
//    2) If it's called in Before interceptor then Before<Action> interceptor,
// mapped <Action>, After<Action> interceptor and After interceptor will not
// execute; framework starts processing response.
//    3) If it's called in Mapped <Action> then After<Action> interceptor and
// After interceptor will not execute; framework starts processing response.
func (ctx *Context) Abort() {
	ctx.abort = true
}

// IsStaticRoute method returns true if it's static route otherwise false.
func (ctx *Context) IsStaticRoute() bool {
	if ctx.route != nil {
		return ctx.route.IsStatic
	}
	return false
}

// SetURL method is to set the request URL to change the behaviour of request
// routing. Ideal for URL rewrting. URL can be relative or absolute URL.
//
// Note: This method only takes effect on `OnRequest` server event.
func (ctx *Context) SetURL(pathURL string) {
	if !ctx.decorated {
		return
	}

	u, err := url.Parse(pathURL)
	if err != nil {
		ctx.Log().Errorf("invalid URL provided: %s", err)
		return
	}

	rawReq := ctx.Req.Unwrap()
	if !ess.IsStrEmpty(u.Host) {
		ctx.Log().Debugf("Host have been updated from '%s' to '%s'", ctx.Req.Host, u.Host)
		rawReq.Host = u.Host
		rawReq.URL.Host = u.Host
	}

	ctx.Log().Debugf("URL path have been updated from '%s' to '%s'", ctx.Req.Path, u.Path)
	rawReq.URL.Path = u.Path

	// Update the context
	ctx.Req.Host = rawReq.Host
	ctx.Req.Path = rawReq.URL.Path
}

// SetMethod method is to set the request `Method` to change the behaviour
// of request routing. Ideal for URL rewrting.
//
// Note: This method only takes effect on `OnRequest` server event.
func (ctx *Context) SetMethod(method string) {
	if !ctx.decorated {
		return
	}

	method = strings.ToUpper(method)
	if _, found := router.HTTPMethodActionMap[method]; !found {
		ctx.Log().Errorf("given method '%s' is not valid", method)
		return
	}

	ctx.Log().Debugf("Request method have been updated from '%s' to '%s'", ctx.Req.Method, method)
	ctx.Req.Unwrap().Method = method
	ctx.Req.Method = method
}

// Reset method resets context instance for reuse.
func (ctx *Context) reset() {
	ctx.Req = nil
	ctx.Res = nil
	ctx.controller = nil
	ctx.action = nil
	ctx.actionrv = reflect.Value{}
	ctx.target = nil
	ctx.targetrv = reflect.Value{}
	ctx.domain = nil
	ctx.route = nil
	ctx.subject = nil
	ctx.reply = nil
	ctx.viewArgs = nil
	ctx.values = nil
	ctx.abort = false
	ctx.decorated = false
	ctx.logger = nil
}

// Set method is used to set value for the given key in the current request flow.
func (ctx *Context) Set(key string, value interface{}) {
	if ctx.values == nil {
		ctx.values = make(map[string]interface{})
	}
	ctx.values[key] = value
}

// Get method returns the value for the given key, otherwise it returns nil.
func (ctx *Context) Get(key string) interface{} {
	return ctx.values[key]
}

// Log method addeds field `Request ID` into current log context and returns
// the logger.
func (ctx *Context) Log() log.Loggerer {
	if ctx.logger == nil {
		ctx.logger = ctx.a.Log().WithFields(log.Fields{
			"reqid": ctx.Req.Header.Get(ctx.a.requestIDHeaderKey),
		})
	}
	return ctx.logger
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Context Unexported methods
//______________________________________________________________________________

func (ctx *Context) setRequestID() {
	reqID := ctx.Req.Header.Get(ctx.a.requestIDHeaderKey)
	if reqID == "" {
		guid := ess.NewGUID()
		ctx.Req.Header.Set(ctx.a.requestIDHeaderKey, guid)
		ctx.Reply().Header(ctx.a.requestIDHeaderKey, guid)
		return
	}
	ctx.Log().Debugf("Request already has traceability ID: %v", reqID)
}

// setTarget method sets contoller, action, embedded context into
// controller.
func (ctx *Context) setTarget(route *router.Route) error {
	if ctx.controller = ctx.e.cregistry.Lookup(route); ctx.controller == nil {
		return errTargetNotFound
	}

	if ctx.action = ctx.controller.Lookup(route.Action); ctx.action == nil {
		return errTargetNotFound
	}

	target := reflect.New(ctx.controller.Type)

	// check action method exists or not
	ctx.actionrv = reflect.ValueOf(target.Interface()).MethodByName(ctx.action.Name)
	if !ctx.actionrv.IsValid() {
		return errTargetNotFound
	}

	targetElem := target.Elem()
	ctxrv := reflect.ValueOf(ctx)
	for _, index := range ctx.controller.EmbeddedIndexes {
		targetElem.FieldByIndex(index).Set(ctxrv)
	}

	ctx.target = target.Interface()
	ctx.targetrv = reflect.ValueOf(ctx.target)
	return nil
}

func (ctx *Context) detectContentType() *ahttp.ContentType {
	// based on HTTP Header 'Accept'
	acceptContType := ctx.Req.AcceptContentType()
	if acceptContType.Mime == "" || acceptContType.Mime == "*/*" {
		// as per 'render.default' from aah.conf
		return ctx.a.defaultContentType
	}
	return acceptContType
}

// wrapGzipWriter method writes respective header for gzip and wraps write into
// gzip writer.
func (ctx *Context) wrapGzipWriter() {
	ctx.Res.Header().Add(ahttp.HeaderVary, ahttp.HeaderAcceptEncoding)
	ctx.Res.Header().Add(ahttp.HeaderContentEncoding, gzipContentEncoding)
	ctx.Res.Header().Del(ahttp.HeaderContentLength)
	ctx.Res = ahttp.WrapGzipWriter(ctx.Res)
}

// writeCookies method writes the user provided cookies and session cookie; also
// saves the session data into session store if its stateful.
func (ctx *Context) writeCookies() {
	for _, c := range ctx.Reply().cookies {
		http.SetCookie(ctx.Res, c)
	}

	if ctx.a.SessionManager().IsStateful() && ctx.subject != nil && ctx.subject.Session != nil {
		if err := ctx.a.SessionManager().SaveSession(ctx.Res, ctx.subject.Session); err != nil {
			ctx.Log().Error(err)
		}
	}
}

func (ctx *Context) writeHeaders() {
	if ctx.a.serverHeaderEnabled {
		ctx.Res.Header().Set(ahttp.HeaderServer, ctx.a.serverHeader)
	}

	// Write application security headers with many safe defaults and
	// configured header values.
	if ctx.a.secureHeadersEnabled {
		secureHeaders := ctx.a.SecurityManager().SecureHeaders
		// Write common secure headers for all request
		for header, value := range secureHeaders.Common {
			ctx.Res.Header().Set(header, value)
		}

		// Applied to all HTML Content-Type
		if ctx.Reply().isHTML() {
			// X-XSS-Protection
			ctx.Res.Header().Set(ahttp.HeaderXXSSProtection, secureHeaders.XSSFilter)

			// Content-Security-Policy (CSP) and applied only to environment `prod`
			if ctx.a.IsProfileProd() && !ess.IsStrEmpty(secureHeaders.CSP) {
				if secureHeaders.CSPReportOnly {
					ctx.Res.Header().Set(ahttp.HeaderContentSecurityPolicy+"-Report-Only", secureHeaders.CSP)
				} else {
					ctx.Res.Header().Set(ahttp.HeaderContentSecurityPolicy, secureHeaders.CSP)
				}
			}
		}

		// Apply only if HTTPS (SSL)
		if ctx.a.IsSSLEnabled() {
			// Strict-Transport-Security (STS, aka HSTS)
			ctx.Res.Header().Set(ahttp.HeaderStrictTransportSecurity, secureHeaders.STS)

			// Public-Key-Pins PKP (aka HPKP) and applied only to environment `prod`
			if ctx.a.IsProfileProd() && !ess.IsStrEmpty(secureHeaders.PKP) {
				if secureHeaders.PKPReportOnly {
					ctx.Res.Header().Set(ahttp.HeaderPublicKeyPins+"-Report-Only", secureHeaders.PKP)
				} else {
					ctx.Res.Header().Set(ahttp.HeaderPublicKeyPins, secureHeaders.PKP)
				}
			}
		}
	}
}

// Render method renders and detects the errors earlier. Writes the
// error info if any.
func (ctx *Context) render() {
	r := ctx.Reply()
	if r.Rdr == nil {
		return
	}

	r.body = acquireBuffer()
	if err := r.Rdr.Render(r.body); err != nil {
		ctx.Log().Error("Render response body error: ", err)

		// panic would be appropriate here, since it handle by aah error
		// handling mechanism. This is second spot in entire
		// aah framework the `panic` used.
		panic(ErrRenderResponse)
	}
}

// callAction method calls targed action method on the controller.
func (ctx *Context) callAction() {
	// Parse Action Parameters
	actionArgs, err := ctx.parseParameters()
	if err != nil { // Any error of parameter parsing result in 400 Bad Request
		ctx.Reply().Error(err)
		return
	}

	ctx.Log().Debugf("Calling controller: %s.%s", ctx.controller.FqName, ctx.action.Name)
	if ctx.actionrv.Type().IsVariadic() {
		ctx.actionrv.CallSlice(actionArgs)
	} else {
		ctx.actionrv.Call(actionArgs)
	}
}
