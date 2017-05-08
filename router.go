// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"errors"
	"fmt"
	"html/template"
	"path/filepath"
	"reflect"
	"strings"

	"aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
	"aahframework.org/router.v0"
)

var appRouter *router.Router

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Global methods
//___________________________________

// AppRouter method returns aah application router instance.
func AppRouter() *router.Router {
	return appRouter
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
	if ess.IsStrEmpty(domain.Port) || domain.Port == "80" {
		routePath = fmt.Sprintf("//%s%s", domain.Host, routePath)
	} else {
		routePath = fmt.Sprintf("//%s:%s%s", domain.Host, domain.Port, routePath)
	}

	return appendAnchorLink(routePath, anchorLink)
}

func findReverseURLDomain(host, routeName string) (*router.Domain, int) {
	log.Tracef("ReverseURL routeName: %s", routeName)
	idx := strings.IndexByte(routeName, '.')
	if idx > 0 {
		subDomain := routeName[:idx]
		if strings.HasPrefix(host, subDomain) {
			log.Tracef("Returning current subdomain: %s.", subDomain)
			return AppRouter().Domains[host], idx
		}

		for k, v := range AppRouter().Domains {
			if strings.HasPrefix(k, subDomain) && v.IsSubDomain {
				log.Tracef("Returning requested subdomain: %s.", subDomain)
				return v, idx
			}
		}
	}

	// return root domain
	log.Trace("Returning root domain")
	return findRootDomain(), idx
}

func findRootDomain() *router.Domain {
	for _, v := range AppRouter().Domains {
		if v.IsSubDomain {
			continue
		}
		return v
	}
	return nil
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
				ctx.Req.Raw.URL.Path = reqPath[:len(reqPath)-1]
			} else {
				ctx.Req.Raw.URL.Path = reqPath + "/"
			}

			reply.Redirect(ctx.Req.Raw.URL.String())
			log.Debugf("RedirectTrailingSlash: %d, %s ==> %s", reply.Code, reqPath, reply.path)
			return nil
		}
	}

	// HTTP: OPTIONS
	if reqMethod == ahttp.MethodOptions {
		if domain.AutoOptions {
			if allowed := domain.Allowed(reqMethod, reqPath); !ess.IsStrEmpty(allowed) {
				allowed += ", " + ahttp.MethodOptions
				log.Debugf("Auto 'OPTIONS' allowed HTTP Methods: %s", allowed)
				reply.Header(ahttp.HeaderAllow, allowed)
				return nil
			}
		}
	}

	// 405 Method Not Allowed
	if domain.MethodNotAllowed {
		if allowed := domain.Allowed(reqMethod, reqPath); !ess.IsStrEmpty(allowed) {
			allowed += ", " + ahttp.MethodOptions
			log.Debugf("Allowed HTTP Methods for 405 response: %s", allowed)
			reply.MethodNotAllowed().
				Header(ahttp.HeaderAllow, allowed).
				Text("405 Method Not Allowed")
			return nil
		}
	}

	return errors.New("route not found")
}

// handleRouteNotFound method is used for 1. route action not found, 2. route is
// not found and 3. static file/directory.
func handleRouteNotFound(ctx *Context, domain *router.Domain, route *router.Route) {
	// handle effectively to reduce heap allocation
	if domain.NotFoundRoute == nil {
		log.Warnf("Route not found: %s, isStaticFile: false", ctx.Req.Path)
		ctx.Reply().NotFound().Text("404 Not Found")
		return
	}

	log.Warnf("Route not found: %s, isStaticFile: %v", ctx.Req.Path, route.IsStatic)
	if err := ctx.setTarget(route); err == errTargetNotFound {
		ctx.Reply().NotFound().Text("404 Not Found")
		return
	}

	target := reflect.ValueOf(ctx.target)
	notFoundAction := target.MethodByName(ctx.action.Name)

	log.Debugf("Calling user defined not-found action: %s.%s", ctx.controller, ctx.action.Name)
	notFoundAction.Call([]reflect.Value{reflect.ValueOf(route.IsStatic)})
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

	host := viewArgs["Host"].(string)
	routeName := args[0].(string)
	return template.URL(createReverseURL(host, routeName, nil, args[1:]...))
}

// tmplURLm method returns reverse URL by given route name and
// map[string]interface{}. Mapped to Go template func.
func tmplURLm(viewArgs map[string]interface{}, routeName string, args map[string]interface{}) template.URL {
	host := viewArgs["Host"].(string)
	return template.URL(createReverseURL(host, routeName, args))
}
