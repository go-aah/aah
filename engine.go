// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"aahframework.org/ahttp.v0"
	"aahframework.org/aruntime.v0"
	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
	"aahframework.org/pool.v0"
)

const (
	flowCont flowResult = iota
	flowStop
)

const (
	aahServerName         = "aah-go-server"
	gzipContentEncoding   = "gzip"
	hstsHeaderValue       = "max-age=31536000; includeSubDomains"
	defaultGlobalPoolSize = 500
	defaultBufPoolSize    = 200
)

var (
	minifier                       MinifierFunc
	errFileNotFound                = errors.New("file not found")
	noGzipStatusCodes              = []int{http.StatusNotModified, http.StatusNoContent}
	defaultRequestAccessLogPattern = "%clientip %reqid %reqtime %restime %resstatus %ressize %reqmeth    od %requrl"
)

type (
	// MinifierFunc is to minify the HTML buffer and write the response into writer.
	MinifierFunc func(contentType string, w io.Writer, r io.Reader) error

	// flowResult is result of engine activities flow.
	// For e.g.: route, authentication, authorization, etc.
	flowResult uint8

	// Engine is the aah framework application server handler for request and response.
	// Implements `http.Handler` interface.
	engine struct {
		isRequestIDEnabled bool
		requestIDHeader    string
		isGzipEnabled      bool
		ctxPool            *pool.Pool
		reqPool            *pool.Pool
		replyPool          *pool.Pool
		bufPool            *pool.Pool
		cfg                *config.Config
	}

	byName []os.FileInfo
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Engine methods
//___________________________________

// ServeHTTP method implementation of http.Handler interface.
func (e *engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	now := time.Now()

	var ral requestAccessLog

	ralChan := newRequestAccessLogChan()

	isAccessLogEnabled := e.cfg.BoolDefault("request.access_log.enable", false)

	if isAccessLogEnabled {
		ral = requestAccessLog{
			startTime: now,
		}
	}

	ctx := e.prepareContext(w, r)
	defer e.putContext(ctx)

	// Recovery handling, capture every possible panic(s)
	defer e.handleRecovery(ctx)

	if e.isRequestIDEnabled {
		e.setRequestID(ctx)
	}

	// 'OnRequest' server extension point
	publishOnRequestEvent(ctx)

	// Handling route

	if e.handleRoute(ctx) == flowStop {
		return
	}

	// Load session
	e.loadSession(ctx)

	// Parsing request params
	e.parseRequestParams(ctx)

	// Set defaults when actual value not found
	e.setDefaults(ctx)

	// Middlewares, interceptors, targeted controller
	e.executeMiddlewares(ctx)

	// Write Reply on the wire
	e.writeReply(ctx)

	if isAccessLogEnabled {

		ral.ctx = ctx
		ral.requestID = ctx.Req.Header.Get(e.requestIDHeader)
		ral.logPattern = e.cfg.StringDefault("request.access_log.pattern", defaultRequestAccessLogPattern)
		ral.elapsedTime = time.Now().Sub(ral.startTime)

		ralChan <- ral
	}
}

// handleRecovery method handles application panics and recovers from it.
// Panic gets translated into HTTP Internal Server Error (Status 500).
func (e *engine) handleRecovery(ctx *Context) {
	if r := recover(); r != nil {
		log.Errorf("Internal Server Error on %s", ctx.Req.Path)

		st := aruntime.NewStacktrace(r, AppConfig())
		buf := e.getBuffer()
		defer e.putBuffer(buf)

		st.Print(buf)
		log.Error(buf.String())

		ctx.Reply().InternalServerError()
		e.negotiateContentType(ctx)
		if ahttp.ContentTypeJSON.IsEqual(ctx.Reply().ContType) {
			ctx.Reply().JSON(Data{"code": "500", "message": "Internal Server Error"})
		} else if ahttp.ContentTypeXML.IsEqual(ctx.Reply().ContType) {
			ctx.Reply().XML(Data{"code": "500", "message": "Internal Server Error"})
		} else {
			ctx.Reply().Text("500 Internal Server Error")
		}

		e.writeReply(ctx)
	}
}

// setRequestID method sets the unique request id in the request header.
// It won't set new request id header already present.
func (e *engine) setRequestID(ctx *Context) {
	if ess.IsStrEmpty(ctx.Req.Header.Get(e.requestIDHeader)) {
		ctx.Req.Header.Set(e.requestIDHeader, ess.NewGUID())
	} else {
		log.Debugf("Request already has ID: %v", ctx.Req.Header.Get(e.requestIDHeader))
	}
	ctx.Reply().Header(e.requestIDHeader, ctx.Req.Header.Get(e.requestIDHeader))
}

// prepareContext method gets controller, request from pool, set the targeted
// controller, parses the request and returns the controller.
func (e *engine) prepareContext(w http.ResponseWriter, req *http.Request) *Context {
	ctx, r := e.getContext(), e.getRequest()
	ctx.Req = ahttp.ParseRequest(req, r)
	ctx.Res = ahttp.GetResponseWriter(w)
	ctx.reply = e.getReply()
	ctx.viewArgs = make(map[string]interface{})

	return ctx
}

// handleRoute method handle route processing for the incoming request.
// It does-
//  - finding domain
//  - finding route
//  - handling static route
//  - handling redirect trailing slash
//  - auto options
//  - route not found
//  - if route found then it sets targeted controller into context
//  - adds the pathParams into context if present
//
// Returns status as-
//  - flowCont
//  - flowStop
func (e *engine) handleRoute(ctx *Context) flowResult {
	domain := AppRouter().FindDomain(ctx.Req)
	if domain == nil {
		ctx.Reply().NotFound().Text("404 Not Found")
		e.writeReply(ctx)
		return flowStop
	}

	route, pathParams, rts := domain.Lookup(ctx.Req)
	if route == nil { // route not found
		if err := handleRtsOptionsMna(ctx, domain, rts); err == nil {
			e.writeReply(ctx)
			return flowStop
		}

		ctx.route = domain.NotFoundRoute
		handleRouteNotFound(ctx, domain, domain.NotFoundRoute)
		e.writeReply(ctx)
		return flowStop
	}

	ctx.route = route
	ctx.domain = domain

	// Path parameters
	if pathParams.Len() > 0 {
		ctx.Req.Params.Path = make(map[string]string, pathParams.Len())
		for _, v := range *pathParams {
			ctx.Req.Params.Path[v.Key] = v.Value
		}
	}

	// Serving static file
	if route.IsStatic {
		if err := e.serveStatic(ctx); err == errFileNotFound {
			handleRouteNotFound(ctx, domain, route)
			e.writeReply(ctx)
		}
		return flowStop
	}

	// No controller or action found for the route
	if err := ctx.setTarget(route); err == errTargetNotFound {
		handleRouteNotFound(ctx, domain, route)
		e.writeReply(ctx)
		return flowStop
	}

	return flowCont
}

// loadSession method loads session from request for `stateful` session.
func (e *engine) loadSession(ctx *Context) {
	if AppSessionManager().IsStateful() {
		ctx.session = AppSessionManager().GetSession(ctx.Req.Raw)
	}
}

// setDefaults method sets default value based on aah app configuration
// when actual value is not found.
func (e *engine) setDefaults(ctx *Context) {
	if ctx.Req.Locale == nil {
		ctx.Req.Locale = ahttp.NewLocale(AppConfig().StringDefault("i18n.default", "en"))
	}
}

// executeMiddlewares method executes the configured middlewares.
func (e *engine) executeMiddlewares(ctx *Context) {
	mwChain[0].Next(ctx)
}

// writeReply method writes the response on the wire based on `Reply` instance.
func (e *engine) writeReply(ctx *Context) {
	reply := ctx.Reply()

	// Response already written on the wire, don't go forward.
	// refer `ctx.Abort()` method.
	if reply.done {
		return
	}

	// HTTP headers
	e.writeHeaders(ctx)

	// Set Cookies
	e.setCookies(ctx)

	if reply.redirect { // handle redirects
		log.Debugf("Redirecting to '%s' with status '%d'", reply.path, reply.Code)
		http.Redirect(ctx.Res, ctx.Req.Raw, reply.path, reply.Code)
		return
	}

	// ContentType
	e.negotiateContentType(ctx)

	// resolving view template
	e.resolveView(ctx)

	// Render it and detect the errors earlier. So that framework can write
	// error info without messing with response on the wire.
	e.doRender(ctx)

	// Gzip
	if !isNoGzipStatusCode(reply.Code) && reply.body.Len() != 0 {
		e.wrapGzipWriter(ctx)
	}

	// ContentType, if it's not set then auto detect later in the writer
	if ctx.Reply().IsContentTypeSet() {
		ctx.Res.Header().Set(ahttp.HeaderContentType, reply.ContType)
	}

	// 'OnPreReply' server extension point
	publishOnPreReplyEvent(ctx)

	// HTTP status
	ctx.Res.WriteHeader(reply.Code)

	// Write response buffer on the wire
	if minifier == nil || !appIsProfileProd ||
		isNoGzipStatusCode(reply.Code) ||
		!ahttp.ContentTypeHTML.IsEqual(reply.ContType) {
		_, _ = reply.body.WriteTo(ctx.Res)
	} else if err := minifier(reply.ContType, ctx.Res, reply.body); err != nil {
		log.Errorf("Minifier error: %v", err)
	}

	// 'OnAfterReply' server extension point
	publishOnAfterReplyEvent(ctx)
}

// negotiateContentType method tries to identify if reply.ContType is empty.
// Not necessarily it will set one.
func (e *engine) negotiateContentType(ctx *Context) {
	if !ctx.Reply().IsContentTypeSet() {
		if !ess.IsStrEmpty(ctx.Req.AcceptContentType.Mime) &&
			ctx.Req.AcceptContentType.Mime != "*/*" { // based on 'Accept' Header
			ctx.Reply().ContentType(ctx.Req.AcceptContentType.String())
		} else if ct := defaultContentType(); ct != nil { // as per 'render.default' in aah.conf
			ctx.Reply().ContentType(ct.String())
		}
	}
}

// wrapGzipWriter method writes respective header for gzip and wraps write into
// gzip writer.
func (e *engine) wrapGzipWriter(ctx *Context) {
	if ctx.Req.IsGzipAccepted && e.isGzipEnabled && ctx.Reply().gzip {
		ctx.Res.Header().Add(ahttp.HeaderVary, ahttp.HeaderAcceptEncoding)
		ctx.Res.Header().Add(ahttp.HeaderContentEncoding, gzipContentEncoding)
		ctx.Res.Header().Del(ahttp.HeaderContentLength)
		ctx.Res = ahttp.GetGzipResponseWriter(ctx.Res)
	}
}

// writeHeaders method writes the headers on the wire.
func (e *engine) writeHeaders(ctx *Context) {
	for k, v := range ctx.Reply().Hdr {
		for _, vv := range v {
			ctx.Res.Header().Add(k, vv)
		}
	}

	ctx.Res.Header().Set(ahttp.HeaderServer, aahServerName)

	// Set the HSTS if SSL is enabled on aah server
	// Know more: https://www.owasp.org/index.php/HTTP_Strict_Transport_Security_Cheat_Sheet
	if AppIsSSLEnabled() {
		ctx.Res.Header().Set(ahttp.HeaderStrictTransportSecurity, hstsHeaderValue)
	}
}

// setCookies method sets the user cookies, session cookie and saves session
// into session store is session mode is stateful.
func (e *engine) setCookies(ctx *Context) {
	for _, c := range ctx.Reply().cookies {
		http.SetCookie(ctx.Res, c)
	}

	if AppSessionManager().IsStateful() && ctx.session != nil {
		// Pass it to view args before saving cookie
		session := *ctx.session
		ctx.AddViewArg(keySessionValues, &session)
		if err := AppSessionManager().SaveSession(ctx.Res, ctx.session); err != nil {
			log.Error(err)
		}
	}
}

// getContext method gets context instance from the pool
func (e *engine) getContext() *Context {
	return e.ctxPool.Get().(*Context)
}

// getRequest method gets request instance from the pool
func (e *engine) getRequest() *ahttp.Request {
	return e.reqPool.Get().(*ahttp.Request)
}

// getReply method gets reply instance from the pool
func (e *engine) getReply() *Reply {
	return e.replyPool.Get().(*Reply)
}

// putContext method puts context back to pool
func (e *engine) putContext(ctx *Context) {
	// Close the writer and Put back to pool
	if ctx.Res != nil {
		if _, ok := ctx.Res.(*ahttp.GzipResponse); ok {
			ahttp.PutGzipResponseWiriter(ctx.Res)
		} else {
			ahttp.PutResponseWriter(ctx.Res)
		}
	}

	// clear and put `ahttp.Request` into pool
	if ctx.Req != nil {
		ctx.Req.Reset()
		e.reqPool.Put(ctx.Req)
	}

	// clear and put `Reply` into pool
	if ctx.reply != nil {
		e.putBuffer(ctx.reply.body)
		ctx.reply.Reset()
		e.replyPool.Put(ctx.reply)
	}

	// clear and put `aah.Context` into pool
	ctx.Reset()
	e.ctxPool.Put(ctx)
}

// getBuffer method gets buffer from pool
func (e *engine) getBuffer() *bytes.Buffer {
	return e.bufPool.Get().(*bytes.Buffer)
}

// putBPool puts buffer into pool
func (e *engine) putBuffer(b *bytes.Buffer) {
	if b == nil {
		return
	}
	b.Reset()
	e.bufPool.Put(b)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

func newEngine(cfg *config.Config) *engine {
	ahttp.GzipLevel = cfg.IntDefault("render.gzip.level", 5)

	if !(ahttp.GzipLevel >= 1 && ahttp.GzipLevel <= 9) {
		logAsFatal(fmt.Errorf("'render.gzip.level' is not a valid level value: %v", ahttp.GzipLevel))
	}

	return &engine{
		isRequestIDEnabled: cfg.BoolDefault("request.id.enable", true),
		requestIDHeader:    cfg.StringDefault("request.id.header", ahttp.HeaderXRequestID),
		isGzipEnabled:      cfg.BoolDefault("render.gzip.enable", true),
		ctxPool: pool.NewPool(
			cfg.IntDefault("runtime.pooling.global", defaultGlobalPoolSize),
			func() interface{} {
				return &Context{}
			},
		),
		reqPool: pool.NewPool(
			cfg.IntDefault("runtime.pooling.global", defaultGlobalPoolSize),
			func() interface{} {
				return &ahttp.Request{}
			},
		),
		replyPool: pool.NewPool(
			cfg.IntDefault("runtime.pooling.global", defaultGlobalPoolSize),
			func() interface{} {
				return NewReply()
			},
		),
		bufPool: pool.NewPool(
			cfg.IntDefault("runtime.pooling.buffer", defaultBufPoolSize),
			func() interface{} {
				return &bytes.Buffer{}
			},
		),
		cfg: cfg,
	}
}
