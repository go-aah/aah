// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"errors"
	"io"
	"net/http"
	"sync"

	"aahframework.org/ahttp.v0"
	"aahframework.org/ainsp.v0"
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

	// Standard frame type MTU size is 1500 bytes so 1400 bytes would make sense
	// to Gzip by default. Read: https://en.wikipedia.org/wiki/Maximum_transmission_unit
	defaultGzipMinSize = 1400
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
// HTTP Engine
//______________________________________________________________________________

// HTTP Engine for aah framework
type httpEngine struct {
	a        *app
	ctxPool  *sync.Pool
	mwStack  []MiddlewareFunc
	mwChain  []*Middleware
	registry *ainsp.TargetRegistry

	// http engine events/extensions
	onRequestFunc    EventCallbackFunc
	onPreReplyFunc   EventCallbackFunc
	onAfterReplyFunc EventCallbackFunc
	onPreAuthFunc    EventCallbackFunc
	onPostAuthFunc   EventCallbackFunc
}

func (e *httpEngine) Log() log.Loggerer {
	return e.a.logger
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// HTTP Engine - Server Extensions
//______________________________________________________________________________

func (e *httpEngine) OnRequest(sef EventCallbackFunc) {
	if e.onRequestFunc != nil {
		e.Log().Warnf("Changing 'OnRequest' server extension from '%s' to '%s'",
			funcName(e.onRequestFunc), funcName(sef))
	}
	e.onRequestFunc = sef
}

func (e *httpEngine) OnPreReply(sef EventCallbackFunc) {
	if e.onPreReplyFunc != nil {
		e.Log().Warnf("Changing 'OnPreReply' server extension from '%s' to '%s'",
			funcName(e.onPreReplyFunc), funcName(sef))
	}
	e.onPreReplyFunc = sef
}

func (e *httpEngine) OnAfterReply(sef EventCallbackFunc) {
	if e.onAfterReplyFunc != nil {
		e.Log().Warnf("Changing 'OnAfterReply' server extension from '%s' to '%s'",
			funcName(e.onAfterReplyFunc), funcName(sef))
	}
	e.onAfterReplyFunc = sef
}

func (e *httpEngine) OnPreAuth(sef EventCallbackFunc) {
	if e.onPreAuthFunc != nil {
		e.Log().Warnf("Changing 'OnPreAuth' server extension from '%s' to '%s'",
			funcName(e.onPreAuthFunc), funcName(sef))
	}
	e.onPreAuthFunc = sef
}

func (e *httpEngine) OnPostAuth(sef EventCallbackFunc) {
	if e.onPostAuthFunc != nil {
		e.Log().Warnf("Changing 'OnPostAuth' server extension from '%s' to '%s'",
			funcName(e.onPostAuthFunc), funcName(sef))
	}
	e.onPostAuthFunc = sef
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// HTTP Engine - Server Extension Publish
//______________________________________________________________________________

func (e *httpEngine) publishOnRequestEvent(ctx *Context) {
	if e.onRequestFunc != nil {
		ctx.decorated = true
		e.onRequestFunc(&Event{Name: EventOnRequest, Data: ctx})
		ctx.decorated = false
	}
}

func (e *httpEngine) publishOnPreReplyEvent(ctx *Context) {
	if e.onPreReplyFunc != nil {
		e.onPreReplyFunc(&Event{Name: EventOnPreReply, Data: ctx})
	}
}

func (e *httpEngine) publishOnAfterReplyEvent(ctx *Context) {
	if e.onAfterReplyFunc != nil {
		e.onAfterReplyFunc(&Event{Name: EventOnAfterReply, Data: ctx})
	}
}

func (e *httpEngine) publishOnPreAuthEvent(ctx *Context) {
	if e.onPreAuthFunc != nil {
		e.onPreAuthFunc(&Event{Name: EventOnPreAuth, Data: ctx})
	}
}

func (e *httpEngine) publishOnPostAuthEvent(ctx *Context) {
	if e.onPostAuthFunc != nil {
		e.onPostAuthFunc(&Event{Name: EventOnPostAuth, Data: ctx})
	}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Engine Unexported methods
//______________________________________________________________________________

func (e *httpEngine) newContext() *Context {
	return &Context{a: e.a, e: e}
}

// handleRecovery method handles application panics and recovers from it.
// Panic gets translated into HTTP Internal Server Error (Status 500).
func (e *httpEngine) handleRecovery(ctx *Context) {
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
func (e *httpEngine) writeReply(ctx *Context) {
	re := ctx.Reply()
	if re.err != nil {
		e.a.errorMgr.Handle(ctx)
	}

	// don't go forward, if:
	// 	- Response already written on the wire, refer to method `Reply().Done()`
	// 	- Static file route
	if re.done || ctx.IsStaticRoute() {
		return
	}

	// 'OnPreReply' server extension point
	e.publishOnPreReplyEvent(ctx)

	// HTTP headers
	ctx.writeHeaders()

	// Set Cookies
	ctx.writeCookies()

	if re.redirect { // handle redirects
		ctx.Log().Debugf("Redirecting to '%s' with status '%d'", re.path, re.Code)
		http.Redirect(ctx.Res, ctx.Req.Unwrap(), re.path, re.Code)
		return
	}

	// Check ContentType and detect it if need be
	if ess.IsStrEmpty(re.ContType) {
		re.ContentType(ctx.detectContentType().String())
	}

	if bodyAllowedForStatus(re.Code) {
		if e.a.viewMgr != nil && re.isHTML() {
			e.a.viewMgr.resolve(ctx)
		}

		e.writeOnWire(ctx)
	} else {
		ctx.Res.Header().Set(ahttp.HeaderContentType, re.ContType)
		ctx.Res.WriteHeader(re.Code)
	}

	// 'OnAfterReply' server extension point
	e.publishOnAfterReplyEvent(ctx)

	// Dump request and response
	if e.a.dumpLogEnabled {
		e.a.dumpLog.Dump(ctx)
	}
}

func (e *httpEngine) writeOnWire(ctx *Context) {
	re := ctx.Reply()
	if _, ok := re.Rdr.(*binaryRender); ok {
		e.writeBinary(ctx)
		return
	}

	// Render it
	if re.Rdr == nil {
		ctx.Res.Header().Set(ahttp.HeaderContentType, re.ContType)
		ctx.Res.WriteHeader(re.Code)
		return
	}
	re.body = acquireBuffer()
	if err := re.Rdr.Render(re.body); err != nil {
		ctx.Log().Error("Response render error: ", err)
		panic(ErrRenderResponse)
	}

	// Check response qualify for Gzip
	if e.a.gzipEnabled && ctx.Req.IsGzipAccepted &&
		re.gzip && re.body.Len() > defaultGzipMinSize {
		ctx.Res = wrapGzipWriter(ctx.Res)
	}

	ctx.Res.Header().Set(ahttp.HeaderContentType, re.ContType)
	ctx.Res.WriteHeader(re.Code)

	var w io.Writer = ctx.Res

	// If response dump log enabled with response body
	if e.a.dumpLogEnabled && e.a.dumpLog.logResponseBody {
		resBuf := acquireBuffer()
		w = io.MultiWriter([]io.Writer{w, resBuf}...)
		ctx.Set(keyAahResponseBodyBuf, resBuf)
	}

	// currently write error on wire is not propagated to error
	// since we can't do anything after that.
	// It could be network error, client is gone, etc.
	if re.isHTML() && e.minifierExists() {
		// HTML Minifier configured
		if err := e.a.viewMgr.minifier(re.ContType, w, re.body); err != nil {
			ctx.Log().Error(err)
		}
	} else {
		if _, err := re.body.WriteTo(w); err != nil {
			ctx.Log().Error(err)
		}
	}
}

func (e *httpEngine) writeBinary(ctx *Context) {
	re := ctx.Reply()

	// Check response qualify for Gzip
	if e.a.gzipEnabled && ctx.Req.IsGzipAccepted && re.gzip {
		ctx.Res = wrapGzipWriter(ctx.Res)
	}

	ctx.Res.Header().Set(ahttp.HeaderContentType, re.ContType)
	ctx.Res.WriteHeader(re.Code)

	// currently write error on wire is not propagated to error
	// since we can't do anything after that.
	// It could be network error, client is gone, etc.
	if err := re.Rdr.Render(ctx.Res); err != nil {
		ctx.Log().Error("Response write error: ", err)
	}
}

func (e *httpEngine) minifierExists() bool {
	return e.a.viewMgr != nil && e.a.viewMgr.minifier != nil
}

func (e *httpEngine) releaseContext(ctx *Context) {
	ahttp.ReleaseResponseWriter(ctx.Res)
	ahttp.ReleaseRequest(ctx.Req)
	security.ReleaseSubject(ctx.subject)
	releaseBuffer(ctx.Reply().Body())

	ctx.reset()
	e.ctxPool.Put(ctx)
}
