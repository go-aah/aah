// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"aahframework.org/ahttp.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
	"aahframework.org/router.v0"
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
	if ctx.Req.Method == ahttp.MethodOptions &&
		ctx.Req.Header.Get(ahttp.HeaderAccessControlRequestMethod) != "" {
		handleCORSPreflight(ctx)
		return
	}

	// CORS headers
	cors := ctx.route.CORS
	origin := ctx.Req.Header.Get(ahttp.HeaderOrigin)
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
	origin := ctx.Req.Header.Get(ahttp.HeaderOrigin)
	if cors.IsOriginAllowed(origin) {
		ctx.Reply().Header(ahttp.HeaderAccessControlAllowOrigin, origin)
	} else {
		ctx.Log().Warnf("CORS: preflight request - invalid origin '%s' for %s %s",
			origin, ctx.Req.Method, ctx.Req.Path)
		ctx.Reply().BadRequest().Error(newError(router.ErrCORSOriginIsInvalid, http.StatusBadRequest))
		return
	}

	// Check Method
	method := ctx.Req.Header.Get(ahttp.HeaderAccessControlRequestMethod)
	if cors.IsMethodAllowed(method) {
		ctx.Reply().Header(ahttp.HeaderAccessControlAllowMethods, strings.Join(cors.AllowMethods, ", "))
	} else {
		ctx.Log().Warnf("CORS: preflight request - method not allowed '%s' for path %s",
			method, ctx.Req.Path)
		ctx.Reply().MethodNotAllowed().Error(newError(router.ErrCORSMethodNotAllowed, http.StatusMethodNotAllowed))
		return
	}

	// Check Headers
	hdrs := ctx.Req.Header.Get(ahttp.HeaderAccessControlRequestHeaders)
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

	if !ess.IsStrEmpty(cors.MaxAge) {
		ctx.Reply().Header(ahttp.HeaderAccessControlMaxAge, cors.MaxAge)
	}

	ctx.Reply().Ok().Text("")
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// app methods
//______________________________________________________________________________

func (a *app) Router() *router.Router {
	return a.router
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// app Unexported methods
//______________________________________________________________________________

func (a *app) initRouter() error {
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
//  - adds the pathParams into context if present
//
// Returns status as-
//  - flowCont
//  - flowStop
func handleRoute(ctx *Context) flowResult {
	domain := ctx.a.Router().Lookup(ctx.Req.Host)
	if domain == nil {
		ctx.Log().Warnf("Domain not found, Host: %s, Path: %s", ctx.Req.Host, ctx.Req.Path)
		ctx.Reply().NotFound().Error(newError(ErrDomainNotFound, http.StatusNotFound))
		return flowAbort
	}
	ctx.domain = domain

	route, pathParams, rts := domain.Lookup(ctx.Req.Unwrap())
	if route == nil { // route not found
		if err := handleRtsOptionsMna(ctx, domain, rts); err == nil {
			return flowAbort
		}

		ctx.Log().Warnf("Route not found, Host: %s, Path: %s", ctx.Req.Host, ctx.Req.Path)
		ctx.Reply().NotFound().Error(newError(ErrRouteNotFound, http.StatusNotFound))
		return flowAbort
	}
	ctx.route = route
	ctx.Req.PathParams = pathParams

	// Serving static file
	if route.IsStatic {
		if err := ctx.a.staticMgr.Serve(ctx); err == errFileNotFound {
			ctx.Log().Warnf("Static file not found, Host: %s, Path: %s", ctx.Req.Host, ctx.Req.Path)
			ctx.Reply().done = false
			ctx.Reply().NotFound().Error(newError(ErrStaticFileNotFound, http.StatusNotFound))
		}
		return flowAbort
	}

	return flowCont
}

func appendAnchorLink(routePath, anchorLink string) string {
	if ess.IsStrEmpty(anchorLink) {
		return routePath
	}
	return routePath + "#" + anchorLink
}

func getRouteNameAndAnchorLink(routeName string) (string, string) {
	anchorLink := ""
	hashIdx := strings.IndexByte(routeName, '#')
	if hashIdx > 0 {
		anchorLink = routeName[hashIdx+1:]
		routeName = routeName[:hashIdx]
	}
	return routeName, anchorLink
}

func composeRouteURL(domain *router.Domain, routePath, anchorLink string) string {
	if ess.IsStrEmpty(domain.Port) {
		routePath = fmt.Sprintf("//%s%s", domain.Host, routePath)
	} else {
		routePath = fmt.Sprintf("//%s:%s%s", domain.Host, domain.Port, routePath)
	}

	return appendAnchorLink(routePath, anchorLink)
}

func (a *app) findReverseURLDomain(host, routeName string) (*router.Domain, string) {
	idx := strings.IndexByte(routeName, '.')
	if idx > 0 {
		subDomain := routeName[:idx]

		// Returning current subdomain
		if strings.HasPrefix(host, subDomain) {
			return a.Router().Lookup(host), routeName[idx+1:]
		}

		// Returning requested subdomain
		for k, v := range a.Router().Domains {
			if strings.HasPrefix(k, subDomain) && v.IsSubDomain {
				return v, routeName[idx+1:]
			}
		}
	}

	// return root domain
	root := a.Router().RootDomain()
	a.Log().Tracef("ReverseURL: routeName: %s, host: %s", routeName, root.Host)
	return root, routeName
}

func createReverseURL(l log.Loggerer, domain *router.Domain, routeName string, margs map[string]interface{}, args ...interface{}) string {
	if routeName == "host" {
		return composeRouteURL(domain, "", "")
	}

	routeName, anchorLink := getRouteNameAndAnchorLink(routeName)
	var routePath string
	if margs == nil {
		routePath = domain.ReverseURL(routeName, args...)
	} else {
		routePath = domain.ReverseURLm(routeName, margs)
	}

	// URL escapes
	rURL, err := url.Parse(composeRouteURL(domain, routePath, anchorLink))
	if err != nil {
		l.Error(err)
		return ""
	}
	return rURL.String()
}

// handleRtsOptionsMna method handles 1) Redirect Trailing Slash 2) Options
// 3) Method not allowed
func handleRtsOptionsMna(ctx *Context, domain *router.Domain, rts bool) error {
	reqMethod := ctx.Req.Method
	reqPath := ctx.Req.Path
	reply := ctx.Reply()

	// Redirect Trailing Slash
	if reqMethod != ahttp.MethodConnect && reqPath != router.SlashString {
		if rts && domain.RedirectTrailingSlash {
			reply.MovedPermanently()
			if reqMethod != ahttp.MethodGet {
				reply.TemporaryRedirect()
			}

			if len(reqPath) > 1 && reqPath[len(reqPath)-1] == '/' {
				ctx.Req.Unwrap().URL.Path = reqPath[:len(reqPath)-1]
			} else {
				ctx.Req.Unwrap().URL.Path = reqPath + "/"
			}

			reply.Redirect(ctx.Req.Unwrap().URL.String())
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
	if !ess.IsStrEmpty(allowed) {
		allowed += ", " + ahttp.MethodOptions
		reply.Header(ahttp.HeaderAllow, allowed)
		reply.ctx.Log().Debugf("%sAllowed HTTP Methods: %s", prefix, allowed)
		return true
	}
	return false
}
