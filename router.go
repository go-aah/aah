// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"

	"aahframework.org/ahttp.v0"
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

func initRoutes() error {
	routesPath := filepath.Join(appConfigDir(), "routes.conf")
	appRouter = router.New(routesPath, AppConfig())

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
	if ess.IsStrEmpty(routePath) {
		return "#"
	}

	if ess.IsStrEmpty(domain.Port) {
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
	for _, v := range AppRouter().Domains {
		if !v.IsSubDomain {
			log.Tracef("Returning root domain: %s", v.Host)
			return v, -1
		}
	}

	// final fallback, mostly it won't come here
	return AppRouter().Domains[host], -1
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Router middleware
//___________________________________

// RouterMiddleware finds the route of incoming request and moves forward.
// If routes not found it does appropriate response for the request.
func routerMiddleware(c *Controller, m *Middleware) {
	domain := AppRouter().FindDomain(c.Req)
	if domain == nil {
		c.Reply().NotFound().Text("404 Route Not Exists")
		return
	}

	route, pathParams, rts := domain.Lookup(c.Req)
	log.Tracef("Route: %#v, Path Params: %v, rts: %v ", route, pathParams, rts)

	if route != nil { // route found
		if route.IsStatic {
			if err := serveStatic(c, route, pathParams); err == errFileNotFound {
				handleNotFound(c, domain, route.IsStatic)
			}
			return
		}

		if err := c.setTarget(route); err == errTargetNotFound {
			handleNotFound(c, domain, false)
			return
		}

		// Path parameters
		if pathParams.Len() > 0 {
			c.Req.Params.Path = make(map[string]string, pathParams.Len())
			for _, v := range *pathParams {
				c.Req.Params.Path[v.Key] = v.Value
			}
		}

		c.domain = domain

		m.Next(c)

		return
	}

	// Redirect Trailing Slash
	if c.Req.Method != ahttp.MethodConnect && c.Req.Path != router.SlashString {
		if rts && domain.RedirectTrailingSlash {
			redirectTrailingSlash(c)
			return
		}
	}

	// HTTP: OPTIONS
	if c.Req.Method == ahttp.MethodOptions {
		if domain.AutoOptions {
			if allowed := domain.Allowed(c.Req.Method, c.Req.Path); !ess.IsStrEmpty(allowed) {
				allowed += ", " + ahttp.MethodOptions
				log.Debugf("Auto 'OPTIONS' allowed HTTP Methods: %s", allowed)
				c.Reply().Header(ahttp.HeaderAllow, allowed)
				return
			}
		}
	}

	// 405 Method Not Allowed
	if domain.MethodNotAllowed {
		if allowed := domain.Allowed(c.Req.Method, c.Req.Path); !ess.IsStrEmpty(allowed) {
			allowed += ", " + ahttp.MethodOptions
			log.Debugf("Allowed HTTP Methods for 405 response: %s", allowed)
			c.Reply().
				Status(http.StatusMethodNotAllowed).
				Header(ahttp.HeaderAllow, allowed).
				Text("405 Method Not Allowed")
			return
		}
	}

	// 404 not found
	handleNotFound(c, domain, false)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Template methods
//___________________________________

// tmplURL method returns reverse URL by given route name and args.
// Mapped to Go template func.
func tmplURL(viewArgs map[string]interface{}, args ...interface{}) template.URL {
	if len(args) == 0 {
		log.Errorf("route not found: %v", args)
		return template.URL("#")
	}

	host := viewArgs["Host"].(string)
	routeName := args[0].(string)

	domain, idx := findReverseURLDomain(host, routeName)
	if idx > 0 {
		routeName = routeName[idx+1:]
	}

	routeName, anchorLink := getRouteNameAndAnchorLink(routeName)
	routePath := domain.ReverseURL(routeName, args[1:]...)

	return template.URL(composeRouteURL(domain, routePath, anchorLink))
}

// tmplURLm method returns reverse URL by given route name and
// map[string]interface{}. Mapped to Go template func.
func tmplURLm(viewArgs map[string]interface{}, routeName string, args map[string]interface{}) template.URL {
	if len(args) == 0 {
		log.Errorf("route not found: %v", args)
		return template.URL("#")
	}

	host := viewArgs["Host"].(string)
	domain, idx := findReverseURLDomain(host, routeName)
	if idx > 0 {
		routeName = routeName[idx+1:]
	}

	routeName, anchorLink := getRouteNameAndAnchorLink(routeName)
	routePath := domain.ReverseURLm(routeName, args)

	return template.URL(composeRouteURL(domain, routePath, anchorLink))
}
