// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"

	"aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
	"aahframework.org/router.v0"
)

var appRouter *router.Router

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package methods
//___________________________________

// AppRouter method returns aah application router instance.
func AppRouter() *router.Router {
	return appRouter
}

// RouteMiddleware method performs the routing logic.
func RouteMiddleware(ctx *Context, m *Middleware) {
	if handleRoute(ctx) == flowStop {
		return
	}

	m.Next(ctx)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

func initRoutes(cfgDir string, appCfg *config.Config) error {
	routesPath := filepath.Join(cfgDir, "routes.conf")
	appRouter = router.New(routesPath, appCfg)

	if err := appRouter.Load(); err != nil {
		return fmt.Errorf("routes.conf: %s", err)
	}

	return nil
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
func handleRoute(ctx *Context) flowResult {
	domain := AppRouter().FindDomain(ctx.Req)
	if domain == nil {
		ctx.Log().Warnf("Domain not found, Host: %s, Path: %s", ctx.Req.Host, ctx.Req.Path)
		ctx.Reply().Error(&Error{
			Reason:  ErrDomainNotFound,
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

		ctx.Log().Warnf("Route not found, Host: %s, Path: %s", ctx.Req.Host, ctx.Req.Path)
		ctx.Reply().Error(&Error{
			Reason:  ErrRouteNotFound,
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
		if err := appEngine.serveStatic(ctx); err == errFileNotFound {
			ctx.Log().Warnf("Static file not found, Host: %s, Path: %s", ctx.Req.Host, ctx.Req.Path)
			ctx.Reply().done = false
			ctx.Reply().Error(&Error{
				Reason:  ErrStaticFileNotFound,
				Code:    http.StatusNotFound,
				Message: http.StatusText(http.StatusNotFound),
			})
		}
		return flowStop
	}

	// No controller or action found for the route
	if err := ctx.setTarget(route); err == errTargetNotFound {
		ctx.Log().Warnf("Target not found, Controller: %s, Action: %s", route.Controller, route.Action)
		ctx.Reply().Error(&Error{
			Reason:  ErrControllerOrActionNotFound,
			Code:    http.StatusNotFound,
			Message: http.StatusText(http.StatusNotFound),
		})
		return flowStop
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

func findReverseURLDomain(host, routeName string) (*router.Domain, int) {
	idx := strings.IndexByte(routeName, '.')
	if idx > 0 {
		subDomain := routeName[:idx]

		// Returning current subdomain
		if strings.HasPrefix(host, subDomain) {
			log.Tracef("ReverseURL: routeName: %s, host: %s", routeName, host)
			return AppRouter().Domains[host], idx
		}

		// Returning requested subdomain
		for k, v := range AppRouter().Domains {
			if strings.HasPrefix(k, subDomain) && v.IsSubDomain {
				log.Tracef("ReverseURL: routeName: %s, host: %s", routeName, v.Host)
				return v, idx
			}
		}
	}

	// return root domain
	root := AppRouter().RootDomain()
	log.Tracef("ReverseURL: routeName: %s, host: %s", routeName, root.Host)
	return root, idx
}

func createReverseURL(host, routeName string, margs map[string]interface{}, args ...interface{}) string {
	domain, idx := findReverseURLDomain(host, routeName)
	if idx > 0 {
		routeName = routeName[idx+1:]
	}

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

	return composeRouteURL(domain, routePath, anchorLink)
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
			log.Debugf("RedirectTrailingSlash: %d, %s ==> %s", reply.Code, reqPath, reply.path)
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
			ctx.Reply().Error(&Error{
				Reason:  ErrHTTPMethodNotAllowed,
				Code:    http.StatusMethodNotAllowed,
				Message: http.StatusText(http.StatusMethodNotAllowed),
			})
			return nil
		}
	}

	return errors.New("route not found")
}

func processAllowedMethods(reply *Reply, allowed, prefix string) bool {
	if !ess.IsStrEmpty(allowed) {
		allowed += ", " + ahttp.MethodOptions
		reply.Header(ahttp.HeaderAllow, allowed)
		log.Debugf("%sAllowed HTTP Methods: %s", prefix, allowed)
		return true
	}
	return false
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Template methods
//___________________________________

// tmplURL method returns reverse URL by given route name and args.
// Mapped to Go template func.
func tmplURL(viewArgs map[string]interface{}, args ...interface{}) template.URL {
	if len(args) == 0 {
		log.Errorf("router: template 'rurl' - route name is empty: %v", args)
		return template.URL("#")
	}
	return template.URL(createReverseURL(viewArgs["Host"].(string), args[0].(string), nil, args[1:]...))
}

// tmplURLm method returns reverse URL by given route name and
// map[string]interface{}. Mapped to Go template func.
func tmplURLm(viewArgs map[string]interface{}, routeName string, args map[string]interface{}) template.URL {
	return template.URL(createReverseURL(viewArgs["Host"].(string), routeName, args))
}
