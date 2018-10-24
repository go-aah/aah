// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"errors"
	"fmt"
	"net/http"
	"path"
	"strings"

	"aahframe.work/ahttp"
	"aahframe.work/router"
	"aahframe.work/valpar"
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package methods
//______________________________________________________________________________

// RouteMiddleware method performs the routing logic.
func RouteMiddleware(ctx *Context, m *Middleware) {
	if handleRoute(ctx) == flowAbort {
		return
	}

	m.Next(ctx)
}

// CORSMiddleware provides Cross-Origin Resource Sharing (CORS) access
// control feature.
func CORSMiddleware(ctx *Context, m *Middleware) {
	// If CORS not enabled or nil move on
	if !ctx.domain.CORSEnabled || ctx.route.CORS == nil {
		m.Next(ctx)
		return
	}

	// Always add Vary for header Origin
	ctx.Reply().HeaderAppend(ahttp.HeaderVary, ahttp.HeaderOrigin)

	// CORS OPTIONS request
	if ctx.Req.Method == ahttp.MethodOptions {
		if h := ctx.Req.Header[ahttp.HeaderAccessControlRequestMethod]; len(h) > 0 && len(h[0]) > 0 {
			handleCORSPreflight(ctx)
			return
		}
	}

	// CORS headers
	cors := ctx.route.CORS
	var origin string
	if h := ctx.Req.Header[ahttp.HeaderOrigin]; len(h) > 0 {
		origin = h[0]
	}
	if cors.IsOriginAllowed(origin) {
		ctx.Reply().Header(ahttp.HeaderAccessControlAllowOrigin, origin)
	}

	if len(cors.ExposeHeaders) > 0 {
		ctx.Reply().Header(ahttp.HeaderAccessControlExposeHeaders, strings.Join(cors.ExposeHeaders, ", "))
	} else if len(cors.AllowHeaders) > 0 {
		ctx.Reply().Header(ahttp.HeaderAccessControlExposeHeaders, strings.Join(cors.AllowHeaders, ", "))
	}

	if cors.AllowCredentials {
		ctx.Reply().Header(ahttp.HeaderAccessControlAllowCredentials, "true")
	}

	m.Next(ctx)
}

func handleCORSPreflight(ctx *Context) {
	ctx.Log().Infof("CORS: preflight request - Path[%v]", ctx.Req.Path)
	ctx.Reply().
		HeaderAppend(ahttp.HeaderVary, ahttp.HeaderAccessControlRequestMethod).
		HeaderAppend(ahttp.HeaderVary, ahttp.HeaderAccessControlRequestHeaders)

	cors := ctx.route.CORS

	// Check Origin
	var origin string
	if h := ctx.Req.Header[ahttp.HeaderOrigin]; len(h) > 0 {
		origin = h[0]
	}
	if cors.IsOriginAllowed(origin) {
		ctx.Reply().Header(ahttp.HeaderAccessControlAllowOrigin, origin)
	} else {
		ctx.Log().Warnf("CORS: preflight request - invalid origin '%s' for %s %s",
			origin, ctx.Req.Method, ctx.Req.Path)
		ctx.Reply().BadRequest().Error(newError(router.ErrCORSOriginIsInvalid, http.StatusBadRequest))
		return
	}

	// Check Method
	var method string
	if h := ctx.Req.Header[ahttp.HeaderAccessControlRequestMethod]; len(h) > 0 {
		method = h[0]
	}
	if cors.IsMethodAllowed(method) {
		ctx.Reply().Header(ahttp.HeaderAccessControlAllowMethods, strings.Join(cors.AllowMethods, ", "))
	} else {
		ctx.Log().Warnf("CORS: preflight request - method not allowed '%s' for path %s",
			method, ctx.Req.Path)
		ctx.Reply().MethodNotAllowed().Error(newError(router.ErrCORSMethodNotAllowed, http.StatusMethodNotAllowed))
		return
	}

	// Check Headers
	var hdrs string
	if h := ctx.Req.Header[ahttp.HeaderAccessControlRequestHeaders]; len(h) > 0 {
		hdrs = h[0]
	}
	if cors.IsHeadersAllowed(hdrs) {
		if len(cors.AllowHeaders) > 0 {
			ctx.Reply().Header(ahttp.HeaderAccessControlAllowHeaders, strings.Join(cors.AllowHeaders, ", "))
		}
	} else {
		ctx.Log().Warnf("CORS: preflight request - headers not allowed '%s' for path %s",
			hdrs, ctx.Req.Path)
		ctx.Reply().Forbidden().Error(newError(router.ErrCORSHeaderNotAllowed, http.StatusForbidden))
		return
	}

	if cors.AllowCredentials {
		ctx.Reply().Header(ahttp.HeaderAccessControlAllowCredentials, "true")
	}

	if len(cors.MaxAge) > 0 {
		ctx.Reply().Header(ahttp.HeaderAccessControlMaxAge, cors.MaxAge)
	}

	ctx.Reply().Ok().Text("")
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// app Unexported methods
//______________________________________________________________________________

func (a *Application) initRouter() error {
	rtr, err := router.NewWithApp(a,
		path.Join(a.VirtualBaseDir(), "config", "routes.conf"))
	if err != nil {
		return fmt.Errorf("routes.conf: %s", err)
	}
	a.router = rtr
	return nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//______________________________________________________________________________

// handleRoute method handle route processing for the incoming request.
// It does-
//  - finding domain
//  - finding route
//  - handling static route
//  - handling redirect trailing slash
//  - auto options
//  - route not found
//  - if route found then it sets targeted controller into context
//  - adds the url path params into context if present
//
// Returns status as-
//  - flowCont
//  - flowStop
func handleRoute(ctx *Context) flowResult {
	ctx.domain = ctx.a.Router().Lookup(ctx.Req.Host)
	if ctx.domain == nil {
		ctx.Log().Warnf("Domain not found, Host: %s, Path: %s", ctx.Req.Host, ctx.Req.Path)
		ctx.Reply().NotFound().Error(newError(ErrDomainNotFound, http.StatusNotFound))
		return flowAbort
	}

	route, urlParams, rts := ctx.domain.Lookup(ctx.Req.Unwrap())
	if route == nil { // route not found
		if err := handleRtsOptionsMna(ctx, rts); err == nil {
			return flowAbort
		}

		ctx.Log().Warnf("Route not found, Host: %s, Path: %s", ctx.Req.Host, ctx.Req.Path)
		ctx.Reply().NotFound().Error(newError(ErrRouteNotFound, http.StatusNotFound))
		return flowAbort
	}
	ctx.route = route
	ctx.Req.URLParams = urlParams

	// Serving static file
	if route.IsStatic {
		if err := ctx.a.staticMgr.Serve(ctx); err == errFileNotFound {
			ctx.Log().Warnf("Static file not found, Host: %s, Path: %s", ctx.Req.Host, ctx.Req.Path)
			ctx.Reply().done = false
			ctx.Reply().NotFound().Error(newError(ErrStaticFileNotFound, http.StatusNotFound))
		}
		return flowAbort
	}

	// Apply route constraints
	if len(ctx.route.Constraints) > 0 {
		if errs := valpar.ValidateValues(ctx.Req.URLParams.ToMap(), ctx.route.Constraints); len(errs) > 0 {
			ctx.Log().Errorf("Route constraints failed: %s", errs)
			ctx.Reply().BadRequest().Error(newErrorWithData(router.ErrRouteConstraintFailed, http.StatusBadRequest, errs))
			return flowAbort
		}
	}

	return flowCont
}

// handleRtsOptionsMna method handles
// 1) Redirect Trailing Slash
// 2) Auto Options
// 3) Method not allowed
func handleRtsOptionsMna(ctx *Context, rts bool) error {
	reqMethod := ctx.Req.Method
	reqPath := ctx.Req.Path
	reply := ctx.Reply()
	domain := ctx.domain

	// Redirect Trailing Slash
	if reqMethod != ahttp.MethodConnect && reqPath != router.SlashString {
		if rts && domain.RedirectTrailingSlash {
			if reqMethod == ahttp.MethodGet {
				reply.MovedPermanently()
			} else {
				reply.TemporaryRedirect()
			}

			if len(reqPath) > 1 && reqPath[len(reqPath)-1] == '/' {
				ctx.Req.URL().Path = reqPath[:len(reqPath)-1]
			} else {
				ctx.Req.URL().Path = reqPath + "/"
			}

			reply.Redirect(ctx.Req.URL().String())
			ctx.Log().Debugf("RedirectTrailingSlash: %d, %s ==> %s", reply.Code, reqPath, reply.path)
			return nil
		}
	}

	// HTTP: OPTIONS
	if reqMethod == ahttp.MethodOptions {
		if domain.AutoOptions {
			if processAllowedMethods(reply, domain.Allowed(reqMethod, reqPath), "Auto 'OPTIONS', ") {
				ctx.Reply().Text("")
				return nil
			}
		}
	}

	// 405 Method Not Allowed
	if domain.MethodNotAllowed {
		if processAllowedMethods(reply, domain.Allowed(reqMethod, reqPath), "405 response, ") {
			ctx.Reply().MethodNotAllowed().Error(newError(ErrHTTPMethodNotAllowed, http.StatusMethodNotAllowed))
			return nil
		}
	}

	return errors.New("route not found")
}

func processAllowedMethods(reply *Reply, allowed, prefix string) bool {
	if len(allowed) > 0 {
		allowed += ", " + ahttp.MethodOptions
		reply.Header(ahttp.HeaderAllow, allowed)
		reply.ctx.Log().Debugf("%sAllowed HTTP Methods: %s", prefix, allowed)
		return true
	}
	return false
}
