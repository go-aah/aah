// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"aahframework.org/ahttp.v0"
	"aahframework.org/aruntime.v0"
	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
	"aahframework.org/security.v0"
)

const (
	flowCont flowResult = iota
	flowStop
)

const (
	gzipContentEncoding = "gzip"
	hstsHeaderValue     = "max-age=31536000; includeSubDomains"
)

var (
	errFileNotFound = errors.New("file not found")
	ctHTML          = ahttp.ContentTypeHTML

	minifier MinifierFunc
	ctxPool  *sync.Pool
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
		isRequestIDEnabled       bool
		requestIDHeader          string
		isGzipEnabled            bool
		isAccessLogEnabled       bool
		isStaticAccessLogEnabled bool
		isServerHeaderEnabled    bool
		serverHeader             string
	}
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Engine methods
//___________________________________

// ServeHTTP method implementation of http.Handler interface.
func (e *engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Capture the startTime earlier.
	// This value is as accurate as could be.
	startTime := time.Now()

	ctx := e.prepareContext(w, r)
	ctx.Set(appReqStartTimeKey, startTime)
	defer releaseContext(ctx)

	// Recovery handling, capture every possible panic(s)
	defer e.handleRecovery(ctx)

	if e.isRequestIDEnabled {
		e.setRequestID(ctx)
	}

	// 'OnRequest' server extension point
	publishOnRequestEvent(ctx)

	// Handling route
	if e.handleRoute(ctx) == flowStop {
		goto wReply
	}

	// Load session
	e.loadSession(ctx)

	// Authentication and Authorization
	if e.handleAuthcAndAuthz(ctx) == flowStop {
		goto wReply
	}

	// Parsing request params
	if e.parseRequestParams(ctx) == flowStop {
		goto wReply
	}

	// Middlewares, interceptors, targeted controller
	e.executeMiddlewares(ctx)

wReply:
	// Write Reply on the wire
	e.writeReply(ctx)
}

// handleRecovery method handles application panics and recovers from it.
// Panic gets translated into HTTP Internal Server Error (Status 500).
func (e *engine) handleRecovery(ctx *Context) {
	if r := recover(); r != nil {
		log.Errorf("Internal Server Error on %s", ctx.Req.Path)

		st := aruntime.NewStacktrace(r, AppConfig())
		buf := acquireBuffer()
		defer releaseBuffer(buf)

		st.Print(buf)
		log.Error(buf.String())

		ctx.Reply().Error(&Error{
			Code:    http.StatusInternalServerError,
			Message: http.StatusText(http.StatusInternalServerError),
			Data:    r,
		})

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
func (e *engine) prepareContext(w http.ResponseWriter, r *http.Request) *Context {
	ctx := acquireContext()
	ctx.Req = ahttp.AcquireRequest(r)
	ctx.Res = ahttp.AcquireResponseWriter(w)
	ctx.reply = acquireReply()
	ctx.subject = security.AcquireSubject()
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
		log.Warnf("Domain not found, Host: %s, Path: %s", ctx.Req.Host, ctx.Req.Path)
		ctx.Reply().Error(&Error{
			Code:    http.StatusNotFound,
			Message: http.StatusText(http.StatusNotFound),
		})
		return flowStop
	}

	route, pathParams, rts := domain.Lookup(ctx.Req)
	if route == nil { // route not found
		if err := handleRtsOptionsMna(ctx, domain, rts); err == nil {
			return flowStop
		}

		log.Warnf("Route not found, Host: %s, Path: %s", ctx.Req.Host, ctx.Req.Path)
		ctx.Reply().Error(&Error{
			Code:    http.StatusNotFound,
			Message: http.StatusText(http.StatusNotFound),
		})
		return flowStop
	}

	ctx.route = route
	ctx.domain = domain

	// security form auth case
	if isFormAuthLoginRoute(ctx) {
		return flowCont
	}

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
			log.Warnf("Static file not found, Host: %s, Path: %s", ctx.Req.Host, ctx.Req.Path)
			ctx.Reply().done = false
			ctx.Reply().NotFound().body = acquireBuffer()
		}
		return flowStop
	}

	// No controller or action found for the route
	if err := ctx.setTarget(route); err == errTargetNotFound {
		log.Warnf("Target not found, Controller: %s, Action: %s", route.Controller, route.Action)
		ctx.Reply().Error(&Error{
			Code:    http.StatusNotFound,
			Message: http.StatusText(http.StatusNotFound),
		})
		return flowStop
	}

	return flowCont
}

// loadSession method loads session from request for `stateful` session.
func (e *engine) loadSession(ctx *Context) {
	if AppSessionManager().IsStateful() {
		ctx.subject.Session = AppSessionManager().GetSession(ctx.Req.Unwrap())
	}
}

// executeMiddlewares method executes the configured middlewares.
func (e *engine) executeMiddlewares(ctx *Context) {
	mwChain[0].Next(ctx)
}

// writeReply method writes the response on the wire based on `Reply` instance.
func (e *engine) writeReply(ctx *Context) {
	if ctx.Reply().err != nil {
		handleError(ctx, ctx.Reply().err)
	}

	// Response already written on the wire, don't go forward.
	// refer to `Reply().Done()` method.
	if ctx.Reply().done {
		return
	}

	// 'OnPreReply' server extension point
	publishOnPreReplyEvent(ctx)

	// HTTP headers
	e.writeHeaders(ctx)

	// Set Cookies
	e.setCookies(ctx)

	reply := ctx.Reply()
	if reply.redirect { // handle redirects
		log.Debugf("Redirecting to '%s' with status '%d'", reply.path, reply.Code)
		http.Redirect(ctx.Res, ctx.Req.Unwrap(), reply.path, reply.Code)
		return
	}

	// ContentType
	if !reply.IsContentTypeSet() {
		if ct := identifyContentType(ctx); ct != nil {
			reply.ContentType(ct.String())
		}
	}

	// resolving view template
	e.resolveView(ctx)

	// Render it and detect the errors earlier. So that framework can write
	// error info without messing with response on the wire.
	e.doRender(ctx)

	isBodyAllowed := isResponseBodyAllowed(reply.Code)
	// Gzip, 1kb above TODO make it configurable from aah.conf
	if isBodyAllowed && reply.body.Len() > 1024 {
		e.wrapGzipWriter(ctx)
	}

	// ContentType, if it's not set then auto detect later in the writer
	if reply.IsContentTypeSet() {
		ctx.Res.Header().Set(ahttp.HeaderContentType, reply.ContType)
	}

	// HTTP status
	ctx.Res.WriteHeader(reply.Code)

	// Write response on the wire
	if isBodyAllowed {
		e.writeBody(ctx)
	}

	// 'OnAfterReply' server extension point
	publishOnAfterReplyEvent(ctx)

	// Send data to access log channel
	if e.isAccessLogEnabled {
		sendToAccessLog(ctx)
	}
}

// wrapGzipWriter method writes respective header for gzip and wraps write into
// gzip writer.
func (e *engine) wrapGzipWriter(ctx *Context) {
	if ctx.Req.IsGzipAccepted && e.isGzipEnabled && ctx.Reply().gzip {
		ctx.Res.Header().Add(ahttp.HeaderVary, ahttp.HeaderAcceptEncoding)
		ctx.Res.Header().Add(ahttp.HeaderContentEncoding, gzipContentEncoding)
		ctx.Res.Header().Del(ahttp.HeaderContentLength)
		ctx.Res = ahttp.WrapGzipWriter(ctx.Res)
	}
}

// writeHeaders method writes the headers on the wire.
func (e *engine) writeHeaders(ctx *Context) {
	for k, v := range ctx.Reply().Hdr {
		for _, vv := range v {
			ctx.Res.Header().Add(k, vv)
		}
	}

	if e.isServerHeaderEnabled {
		ctx.Res.Header().Set(ahttp.HeaderServer, e.serverHeader)
	}

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

	if AppSessionManager().IsStateful() && ctx.subject.Session != nil {
		if err := AppSessionManager().SaveSession(ctx.Res, ctx.subject.Session); err != nil {
			log.Error(err)
		}
	}
}

func (e *engine) writeBody(ctx *Context) {
	if minifier == nil || !appIsProfileProd || !ctHTML.IsEqual(ctx.Reply().ContType) {
		if _, err := ctx.Reply().body.WriteTo(ctx.Res); err != nil {
			log.Error(err)
		}
	} else if err := minifier(ctx.Reply().ContType, ctx.Res, ctx.Reply().body); err != nil {
		log.Errorf("Minifier error: %s", err.Error())
	}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

func newEngine(cfg *config.Config) *engine {
	ahttp.GzipLevel = cfg.IntDefault("render.gzip.level", 5)
	if !(ahttp.GzipLevel >= 1 && ahttp.GzipLevel <= 9) {
		logAsFatal(fmt.Errorf("'render.gzip.level' is not a valid level value: %v", ahttp.GzipLevel))
	}

	serverHeader := cfg.StringDefault("server.header", "")

	return &engine{
		isRequestIDEnabled:       cfg.BoolDefault("request.id.enable", true),
		requestIDHeader:          cfg.StringDefault("request.id.header", ahttp.HeaderXRequestID),
		isGzipEnabled:            cfg.BoolDefault("render.gzip.enable", true),
		isAccessLogEnabled:       cfg.BoolDefault("server.access_log.enable", false),
		isStaticAccessLogEnabled: cfg.BoolDefault("server.access_log.static_file", true),
		isServerHeaderEnabled:    !ess.IsStrEmpty(serverHeader),
		serverHeader:             serverHeader,
	}
}

func acquireContext() *Context {
	return ctxPool.Get().(*Context)
}

func releaseContext(ctx *Context) {
	ahttp.ReleaseResponseWriter(ctx.Res)
	ahttp.ReleaseRequest(ctx.Req)
	security.ReleaseSubject(ctx.subject)
	releaseReply(ctx.reply)

	ctx.Reset()
	ctxPool.Put(ctx)
}

func init() {
	ctxPool = &sync.Pool{New: func() interface{} {
		return &Context{
			viewArgs: make(map[string]interface{}),
			values:   make(map[string]interface{}),
		}
	}}
}
