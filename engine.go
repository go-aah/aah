// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"errors"
	"io"
	"net/http"
	"sync"
	"time"

	"aahframework.org/ahttp.v0"
	"aahframework.org/aruntime.v0"
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
)

var (
	errFileNotFound = errors.New("file not found")
)

type (
	// MinifierFunc is to minify the HTML buffer and write the response into writer.
	MinifierFunc func(contentType string, w io.Writer, r io.Reader) error

	// flowResult is result of engine activities flow.
	// For e.g.: route, authentication, authorization, etc.
	flowResult uint8
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Engine
//______________________________________________________________________________

// Engine is the aah framework application server handler.
//
// Implements `http.Handler` interface.
type engine struct {
	a         *app
	ctxPool   *sync.Pool
	mwStack   []MiddlewareFunc
	mwChain   []*Middleware
	cregistry controllerRegistry

	// server extensions
	onRequestFunc    EventCallbackFunc
	onPreReplyFunc   EventCallbackFunc
	onAfterReplyFunc EventCallbackFunc
	onPreAuthFunc    EventCallbackFunc
	onPostAuthFunc   EventCallbackFunc
}

func (e *engine) Log() log.Loggerer {
	return e.a.logger
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Engine - Server Extensions
//______________________________________________________________________________

func (e *engine) OnRequest(sef EventCallbackFunc) {
	if e.onRequestFunc != nil {
		e.Log().Warnf("Changing 'OnRequest' server extension from '%s' to '%s'",
			funcName(e.onRequestFunc), funcName(sef))
	}
	e.onRequestFunc = sef
}

func (e *engine) OnPreReply(sef EventCallbackFunc) {
	if e.onPreReplyFunc != nil {
		e.Log().Warnf("Changing 'OnPreReply' server extension from '%s' to '%s'",
			funcName(e.onPreReplyFunc), funcName(sef))
	}
	e.onPreReplyFunc = sef
}

func (e *engine) OnAfterReply(sef EventCallbackFunc) {
	if e.onAfterReplyFunc != nil {
		e.Log().Warnf("Changing 'OnAfterReply' server extension from '%s' to '%s'",
			funcName(e.onAfterReplyFunc), funcName(sef))
	}
	e.onAfterReplyFunc = sef
}

func (e *engine) OnPreAuth(sef EventCallbackFunc) {
	if e.onPreAuthFunc != nil {
		e.Log().Warnf("Changing 'OnPreAuth' server extension from '%s' to '%s'",
			funcName(e.onPreAuthFunc), funcName(sef))
	}
	e.onPreAuthFunc = sef
}

func (e *engine) OnPostAuth(sef EventCallbackFunc) {
	if e.onPostAuthFunc != nil {
		e.Log().Warnf("Changing 'OnPostAuth' server extension from '%s' to '%s'",
			funcName(e.onPostAuthFunc), funcName(sef))
	}
	e.onPostAuthFunc = sef
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Engine - Server Extension Publish
//______________________________________________________________________________

func (e *engine) publishOnRequestEvent(ctx *Context) {
	if e.onRequestFunc != nil {
		ctx.decorated = true
		e.onRequestFunc(&Event{Name: EventOnRequest, Data: ctx})
		ctx.decorated = false
	}
}

func (e *engine) publishOnPreReplyEvent(ctx *Context) {
	if e.onPreReplyFunc != nil {
		e.onPreReplyFunc(&Event{Name: EventOnPreReply, Data: ctx})
	}
}

func (e *engine) publishOnAfterReplyEvent(ctx *Context) {
	if e.onAfterReplyFunc != nil {
		e.onAfterReplyFunc(&Event{Name: EventOnAfterReply, Data: ctx})
	}
}

func (e *engine) publishOnPreAuthEvent(ctx *Context) {
	if e.onPreAuthFunc != nil {
		e.onPreAuthFunc(&Event{Name: EventOnPreAuth, Data: ctx})
	}
}

func (e *engine) publishOnPostAuthEvent(ctx *Context) {
	if e.onPostAuthFunc != nil {
		e.onPostAuthFunc(&Event{Name: EventOnPostAuth, Data: ctx})
	}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Engine - HTTP Handler
//______________________________________________________________________________

// ServeHTTP method implementation of http.Handler interface.
func (e *engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Capture the startTime earlier, so that value is as accurate.
	startTime := time.Now()

	ctx := e.ctxPool.Get().(*Context)
	ctx.Req, ctx.Res = ahttp.AcquireRequest(r), ahttp.AcquireResponseWriter(w)
	ctx.Set(reqStartTimeKey, startTime)
	defer e.releaseContext(ctx)

	// Recovery handling, capture every possible panic's
	defer e.handleRecovery(ctx)

	if e.a.requestIDEnabled {
		ctx.setRequestID()
	}

	// 'OnRequest' server extension point
	e.publishOnRequestEvent(ctx)

	// Middlewares, interceptors, targeted controller
	if len(e.mwChain) == 0 {
		ctx.Log().Error("'init.go' file introduced in release v0.10; please check your 'app-base-dir/app' " +
			"and then add to your version control")
		ctx.Reply().Error(&Error{
			Reason:  ErrGeneric,
			Code:    http.StatusInternalServerError,
			Message: http.StatusText(http.StatusInternalServerError),
		})
	} else {
		e.mwChain[0].Next(ctx)
	}

	e.writeReply(ctx)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Engine Unexported methods
//______________________________________________________________________________

func (e *engine) newContext() *Context {
	return &Context{a: e.a, e: e}
}

// handleRecovery method handles application panics and recovers from it.
// Panic gets translated into HTTP Internal Server Error (Status 500).
func (e *engine) handleRecovery(ctx *Context) {
	if r := recover(); r != nil {
		ctx.Log().Errorf("Internal Server Error on %s", ctx.Req.Path)

		st := aruntime.NewStacktrace(r, e.a.Config())
		buf := acquireBuffer()
		defer releaseBuffer(buf)

		st.Print(buf)
		ctx.Log().Error(buf.String())

		err := ErrPanicRecovery
		if er, ok := r.(error); ok && er == ErrRenderResponse {
			err = er
		}

		ctx.Reply().Error(&Error{
			Reason:  err,
			Code:    http.StatusInternalServerError,
			Message: http.StatusText(http.StatusInternalServerError),
			Data:    r,
		})

		e.writeReply(ctx)
	}
}

// writeReply method writes the response on the wire based on `Reply` instance.
func (e *engine) writeReply(ctx *Context) {
	if ctx.Reply().err != nil {
		e.a.errorMgr.Handle(ctx)
	}

	// don't go forward, if:
	// 	- Response already written on the wire, refer to method `Reply().Done()`
	// 	- Static file route
	if ctx.Reply().done || ctx.IsStaticRoute() {
		return
	}

	// 'OnPreReply' server extension point
	e.publishOnPreReplyEvent(ctx)

	// HTTP headers
	ctx.writeHeaders()

	// Set Cookies
	ctx.writeCookies()

	if ctx.Reply().redirect { // handle redirects
		ctx.Log().Debugf("Redirecting to '%s' with status '%d'", ctx.Reply().path, ctx.Reply().Code)
		http.Redirect(ctx.Res, ctx.Req.Unwrap(), ctx.Reply().path, ctx.Reply().Code)
		return
	}

	bodyAllowed := bodyAllowedForStatus(ctx.Reply().Code)
	if bodyAllowed {
		// check ContentType
		if ess.IsStrEmpty(ctx.Reply().ContType) {
			ctx.Reply().ContentType(ctx.detectContentType().String())
		}

		// resolving view template
		if e.a.viewMgr != nil {
			e.a.viewMgr.resolve(ctx)
		}

		// Render and detect the errors earlier. So that framework can write the
		// error info without messing with response on the wire.
		ctx.render()

		// Gzip, 1kb above TODO make it configurable from aah.conf
		if e.a.gzipEnabled && ctx.Req.IsGzipAccepted &&
			ctx.Reply().gzip && ctx.Reply().Body() != nil && ctx.Reply().Body().Len() > 1024 {
			ctx.wrapGzipWriter()
		}
	}

	// HTTP ContentType
	ctx.Res.Header().Set(ahttp.HeaderContentType, ctx.Reply().ContType)

	// HTTP Status
	ctx.Res.WriteHeader(ctx.Reply().Code)

	// Write response on the wire
	if bodyAllowed {
		e.writeBody(ctx)

	}

	// 'OnAfterReply' server extension point
	e.publishOnAfterReplyEvent(ctx)

	// Send data to access log channel
	if e.a.accessLogEnabled {
		e.sendToAccessLog(ctx)
	}

	// Dump request and response
	if e.a.dumpLogEnabled {
		e.dump(ctx)
	}
}

func (e *engine) writeBody(ctx *Context) {
	if e.a.dumpLogEnabled && e.a.dumpLog.dumpResponseBody {
		ctx.a.dumpLog.addResBodyIntoCtx(ctx)
	}

	// For Prod && HTML && minifier exists
	if e.a.IsProfileProd() && ctx.Reply().isHTML() && e.a.viewMgr.minifier != nil {
		if err := e.a.viewMgr.minifier(ctx.Reply().ContType, ctx.Res, ctx.Reply().Body()); err != nil {
			ctx.Log().Errorf("Minifier error: %s", err.Error())
		}
		return
	}

	// For all cases
	if _, err := ctx.Reply().Body().WriteTo(ctx.Res); err != nil {
		ctx.Log().Error(err)
	}
}

func (e *engine) sendToAccessLog(ctx *Context) {
	al := e.a.accessLog.logPool.Get().(*accessLog)
	al.StartTime = ctx.Get(reqStartTimeKey).(time.Time)

	// All the bytes have been written on the wire
	// so calculate elapsed time
	al.ElapsedDuration = time.Since(al.StartTime)

	req := *ctx.Req
	al.Request = &req
	al.RequestID = firstNonZeroString(req.Header.Get(e.a.requestIDHeaderKey), "-")
	al.ResStatus = ctx.Res.Status()
	al.ResBytes = ctx.Res.BytesWritten()
	al.ResHdr = ctx.Res.Header()

	e.a.accessLog.logChan <- al
}

func (e *engine) releaseContext(ctx *Context) {
	ahttp.ReleaseResponseWriter(ctx.Res)
	ahttp.ReleaseRequest(ctx.Req)
	security.ReleaseSubject(ctx.subject)
	releaseBuffer(ctx.Reply().Body())

	ctx.reset()
	e.ctxPool.Put(ctx)
}
