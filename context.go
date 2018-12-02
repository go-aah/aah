// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"aahframe.work/ahttp"
	"aahframe.work/ainsp"
	"aahframe.work/essentials"
	"aahframe.work/log"
	"aahframe.work/router"
	"aahframe.work/security"
	"aahframe.work/security/authz"
	"aahframe.work/security/session"
)

var (
	_ ess.Valuer = (*Context)(nil)

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

	a          *Application
	e          *HTTPEngine
	controller *ainsp.Target
	action     *ainsp.Method
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

// RouteURL method returns the URL for given route name and args.
// See `router.Domain.RouteURL` for more information.
func (ctx *Context) RouteURL(routeName string, args ...interface{}) string {
	return ctx.a.Router().CreateRouteURL(ctx.Req.Host, routeName, nil, args...)
}

// RouteURLNamedArgs method returns the URL for given route name and key-value paris.
// See `router.Domain.RouteURLNamedArgs` for more information.
func (ctx *Context) RouteURLNamedArgs(routeName string, args map[string]interface{}) string {
	return ctx.a.Router().CreateRouteURL(ctx.Req.Host, routeName, args)
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
// 	framework starts processing response.
//    2) If it's called in Before interceptor then Before<Action> interceptor,
// 	mapped <Action>, After<Action> interceptor and After interceptor will not
// 	execute; framework starts processing response.
//    3) If it's called in Mapped <Action> then After<Action> interceptor and
// 	After interceptor will not execute; framework starts processing response.
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
// Note: This method only takes effect on `OnRequest` HTTP server event.
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
// Note: This method only takes effect on `OnRequest` HTTP server event.
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

// Log method adds field `Request ID` into current log context and returns
// the logger.
func (ctx *Context) Log() log.Loggerer {
	if ctx.logger == nil {
		if h := ctx.Req.Header[ctx.a.settings.RequestIDHeaderKey]; len(h) > 0 {
			ctx.logger = ctx.a.Log().WithFields(log.Fields{
				"reqid": h[0],
			})
		} else {
			ctx.logger = ctx.a.Log()
		}
	}
	return ctx.logger
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Context Unexported methods
//______________________________________________________________________________

func (ctx *Context) setRequestID() {
	h := ctx.Req.Header[ctx.a.settings.RequestIDHeaderKey]
	if len(h) == 0 {
		guid := ess.NewGUID()
		ctx.Req.Header.Set(ctx.a.settings.RequestIDHeaderKey, guid)
		ctx.Reply().Header(ctx.a.settings.RequestIDHeaderKey, guid)
		return
	}
	ctx.Log().Debugf("Request already has traceability ID: %v", h[0])
}

// setTarget method sets contoller, action, embedded context into
// controller.
func (ctx *Context) setTarget(route *router.Route) error {
	if ctx.route == nil || ctx.target != nil {
		return nil
	}

	if ctx.controller = ctx.e.registry.Lookup(route.Target); ctx.controller == nil {
		return errTargetNotFound
	}

	if ctx.action = ctx.controller.Lookup(route.Action); ctx.action == nil {
		return errTargetNotFound
	}

	target := reflect.New(ctx.controller.Type)
	ctx.target = target.Interface()

	// check action method exists or not
	ctx.actionrv = reflect.ValueOf(ctx.target).MethodByName(ctx.action.Name)
	if !ctx.actionrv.IsValid() {
		return errTargetNotFound
	}

	targetElem := target.Elem()
	ctxrv := reflect.ValueOf(ctx)
	for _, index := range ctx.controller.EmbeddedIndexes {
		targetElem.FieldByIndex(index).Set(ctxrv)
	}

	ctx.targetrv = reflect.ValueOf(ctx.target)
	return nil
}

func (ctx *Context) detectContentType() string {
	// based on HTTP Header 'Accept'
	acceptContType := ctx.Req.AcceptContentType()
	if acceptContType.Mime == "" || acceptContType.Mime == "*/*" {
		// as per 'render.default' from aah.conf
		return ctx.a.settings.DefaultContentType
	}
	return acceptContType.String()
}

// writeCookies method writes the user provided cookies and session cookie; also
// saves the session data into session store if its stateful.
func (ctx *Context) writeCookies() {
	for _, c := range ctx.Reply().cookies {
		http.SetCookie(ctx.Res, c)
	}

	if ctx.a.SessionManager().IsStateful() && ctx.a.SessionManager().IsPath(ctx.Req.Path) {
		if ctx.subject != nil && ctx.subject.Session != nil {
			if err := ctx.a.SessionManager().SaveSession(ctx.Res, ctx.subject.Session); err != nil {
				ctx.Log().Error(err)
			}
		}
	}
}

func (ctx *Context) writeHeaders() {
	if ctx.a.settings.ServerHeaderEnabled {
		ctx.Res.Header().Set(ahttp.HeaderServer, ctx.a.settings.ServerHeader)
	}

	// Write application security headers with many safe defaults and
	// configured header values.
	if ctx.a.settings.SecureHeadersEnabled {
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
			if ctx.a.IsEnvProfile("prod") && len(secureHeaders.CSP) > 0 {
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
			if ctx.a.IsEnvProfile("prod") && len(secureHeaders.PKP) > 0 {
				if secureHeaders.PKPReportOnly {
					ctx.Res.Header().Set(ahttp.HeaderPublicKeyPins+"-Report-Only", secureHeaders.PKP)
				} else {
					ctx.Res.Header().Set(ahttp.HeaderPublicKeyPins, secureHeaders.PKP)
				}
			}
		}
	}
}

// hasAccess method checks the subject's access by defined access rule in the
// route.
func (ctx *Context) hasAccess() (bool, []*authz.Reason) {
	return ctx.route.HasAccess(ctx.Subject())
}
