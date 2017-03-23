// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"bytes"
	"errors"
	"net/http"
	"os"

	"aahframework.org/ahttp.v0"
	"aahframework.org/aruntime.v0"
	"aahframework.org/log.v0"
	"aahframework.org/pool.v0"
)

var errFileNotFound = errors.New("file not found")

type (
	// Engine is the aah framework application server handler for request and response.
	// Implements `http.Handler` interface.
	engine struct {
		cPool *pool.Pool
		rPool *pool.Pool
		bPool *pool.Pool
	}

	byName []os.FileInfo
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Engine methods
//___________________________________

// ServeHTTP method implementation of http.Handler interface.
func (e *engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx := e.prepareContext(w, req)
	defer e.putContext(ctx)

	// Recovery handling, capture every possible panic(s)
	defer e.handleRecovery(ctx)

	// 'OnRequest' server extension point
	if onRequestFunc != nil {
		onRequestFunc(&Event{Name: EventOnRequest, Data: ctx})
	}

	domain := AppRouter().FindDomain(req)
	if domain == nil {
		ctx.Reply().NotFound().Text("404 Not Found")
		e.writeReply(ctx)
		return
	}

	route, pathParams, rts := domain.Lookup(req)
	if route == nil { // route not found
		if err := handleRtsOptionsMna(ctx, domain, rts); err == nil {
			e.writeReply(ctx)
			return
		}

		ctx.route = domain.NotFoundRoute
		handleRouteNotFound(ctx, domain, domain.NotFoundRoute)
		e.writeReply(ctx)
		return
	}

	ctx.route = route
	ctx.domain = domain

	// Serving static file
	if route.IsStatic {
		if err := serveStatic(ctx.Res, req, route, pathParams); err == errFileNotFound {
			handleRouteNotFound(ctx, domain, route)
			e.writeReply(ctx)
		}
		return
	}

	// No controller or action found for the route
	if err := ctx.setTarget(route); err == errTargetNotFound {
		handleRouteNotFound(ctx, domain, route)
		e.writeReply(ctx)
		return
	}

	// Path parameters
	if pathParams.Len() > 0 {
		ctx.Req.Params.Path = make(map[string]string, pathParams.Len())
		for _, v := range *pathParams {
			ctx.Req.Params.Path[v.Key] = v.Value
		}
	}

	// set defaults when actual value not found
	e.setDefaults(ctx)

	// Middlewares
	e.executeMiddlewares(ctx)

	// Write Reply on the wire
	e.writeReply(ctx)
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

		if AppProfile() == "prod" {
			ctx.Reply().InternalServerError().Text("500 Internal Server Error")
		} else { // detailed error info
			// TODO design server error page with stack trace info
			ctx.Reply().InternalServerError().Text("500 Internal Server Error: %s", buf.String())
		}

		e.writeReply(ctx)
	}
}

// prepareContext method gets controller, request from pool, set the targeted
// controller, parses the request and returns the controller.
func (e *engine) prepareContext(w http.ResponseWriter, req *http.Request) *Context {
	ctx, r := e.getContext(), e.getRequest()

	ctx.Req = ahttp.ParseRequest(req, r)
	ctx.Res = ahttp.WrapResponseWriter(w)
	ctx.reply = NewReply()
	ctx.viewArgs = make(map[string]interface{}, 0)

	return ctx
}

// setDefaults method sets default value based on aah app configuration
// when actual value is not found.
func (e *engine) setDefaults(ctx *Context) {
	if ctx.Req.Locale == nil {
		ctx.Req.Locale = ahttp.NewLocale(appConfig.StringDefault("i18n.default", "en"))
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
	if reply.done {
		return
	}

	// handle redirects
	if reply.redirect {
		log.Debugf("Redirecting to '%s' with status '%d'", reply.redirectURL, reply.Code)
		http.Redirect(ctx.Res, ctx.Req.Raw, reply.redirectURL, reply.Code)
		return
	}

	handlePreReplyStage(ctx)

	// 'OnPreReply' server extension point
	if onPreReplyFunc != nil {
		onPreReplyFunc(&Event{Name: EventOnPreReply, Data: ctx})
	}

	buf := e.getBuffer()
	defer e.putBuffer(buf)

	// Render and detect the errors earlier, framework can write error info
	// without messing with response.
	// HTTP Body
	if reply.Rdr != nil {
		if err := reply.Rdr.Render(buf); err != nil {
			log.Error("Render error: ", err)
			ctx.Res.WriteHeader(http.StatusInternalServerError)
			_, _ = ctx.Res.Write([]byte("500 Internal Server Error" + "\n"))
			return
		}
	}

	// HTTP headers
	for k, v := range reply.Hdr {
		for _, vv := range v {
			ctx.Res.Header().Add(k, vv)
		}
	}

	// ContentType
	ctx.Res.Header().Set(ahttp.HeaderContentType, reply.ContType)

	// HTTP status
	if reply.IsStatusSet() {
		ctx.Res.WriteHeader(reply.Code)
	} else {
		ctx.Res.WriteHeader(http.StatusOK)
	}

	// Write it on the wire
	_, _ = buf.WriteTo(ctx.Res)
}

// getContext method gets context from pool
func (e *engine) getContext() *Context {
	return e.cPool.Get().(*Context)
}

// getRequest method gets request from pool
func (e *engine) getRequest() *ahttp.Request {
	return e.rPool.Get().(*ahttp.Request)
}

// putContext method puts context back to pool
func (e *engine) putContext(ctx *Context) {
	// Try to close if `io.Closer` interface satisfies.
	if ctx.Res != nil {
		ctx.Res.(*ahttp.Response).Close()
	}

	// clear and put `ahttp.Request` into pool
	if ctx.Req != nil {
		ctx.Req.Reset()
		e.rPool.Put(ctx.Req)
	}

	// clear and put `aah.Context` into pool
	if ctx != nil {
		ctx.Reset()
		e.cPool.Put(ctx)
	}
}

// getBuffer method gets buffer from pool
func (e *engine) getBuffer() *bytes.Buffer {
	return e.bPool.Get().(*bytes.Buffer)
}

// putBPool puts buffer into pool
func (e *engine) putBuffer(b *bytes.Buffer) {
	b.Reset()
	e.bPool.Put(b)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

func newEngine() *engine {
	// TODO provide config for pool size
	return &engine{
		cPool: pool.NewPool(150, func() interface{} {
			return &Context{}
		}),
		rPool: pool.NewPool(150, func() interface{} {
			return &ahttp.Request{}
		}),
		bPool: pool.NewPool(60, func() interface{} {
			return &bytes.Buffer{}
		}),
	}
}
