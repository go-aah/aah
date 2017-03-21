// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"reflect"
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
func handleRtsOptionsMna(domain *router.Domain, req *http.Request, rts bool) *Reply {
	reqMethod := req.Method
	reqPath := req.URL.Path

	// Redirect Trailing Slash
	if reqMethod != ahttp.MethodConnect && reqPath != router.SlashString {
		if rts && domain.RedirectTrailingSlash {
			reply := NewReply().MovedPermanently()
			if reqMethod != ahttp.MethodGet {
				reply.TemporaryRedirect()
			}

			if len(reqPath) > 1 && reqPath[len(reqPath)-1] == '/' {
				req.URL.Path = reqPath[:len(reqPath)-1]
			} else {
				req.URL.Path = reqPath + "/"
			}

			reply.Redirect(req.URL.String())

			log.Debugf("RedirectTrailingSlash: %d, %s ==> %s", reply.Code, reqPath, reply.redirectURL)
			return reply
		}
	}

	// HTTP: OPTIONS
	if reqMethod == ahttp.MethodOptions {
		if domain.AutoOptions {
			if allowed := domain.Allowed(reqMethod, reqPath); !ess.IsStrEmpty(allowed) {
				allowed += ", " + ahttp.MethodOptions
				log.Debugf("Auto 'OPTIONS' allowed HTTP Methods: %s", allowed)
				return NewReply().Header(ahttp.HeaderAllow, allowed)
			}
		}
	}

	// 405 Method Not Allowed
	if domain.MethodNotAllowed {
		if allowed := domain.Allowed(reqMethod, reqPath); !ess.IsStrEmpty(allowed) {
			allowed += ", " + ahttp.MethodOptions
			log.Debugf("Allowed HTTP Methods for 405 response: %s", allowed)
			return NewReply().
				MethodNotAllowed().
				Header(ahttp.HeaderAllow, allowed).
				Text("405 Method Not Allowed")
		}
	}

	return nil
}

// handleRouteNotFound method is used for 1. route action not found, 2. route is
// not found and 3. static file/directory.
func handleRouteNotFound(w ahttp.ResponseWriter, req *http.Request, domain *router.Domain, route *router.Route) {
	// handle effectively to reduce heap allocation
	if domain.NotFoundRoute == nil {
		log.Warnf("Route not found: %s, isStaticFile: false", req.URL.Path)
		appEngine.writeReply(w, req, NewReply().NotFound().Text("404 Not Found"))
		return
	}

	log.Warnf("Route not found: %s, isStaticFile: %v", req.URL.Path, route.IsStatic)
	c := appEngine.prepareController(w, req, domain.NotFoundRoute)
	if c == nil {
		appEngine.writeReply(w, req, NewReply().NotFound().Text("404 Not Found"))
		return
	}

	defer appEngine.putController(c)

	target := reflect.ValueOf(c.target)
	notFoundAction := target.MethodByName(c.action.Name)

	log.Debugf("Calling user defined not-found action: %s.%s", c.controller, c.action.Name)
	notFoundAction.Call([]reflect.Value{reflect.ValueOf(route.IsStatic)})
	appEngine.writeReply(w, req, c.reply)
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
	return template.URL(createReverseURL(host, routeName, nil, args[1:]...))
}

// tmplURLm method returns reverse URL by given route name and
// map[string]interface{}. Mapped to Go template func.
func tmplURLm(viewArgs map[string]interface{}, routeName string, args map[string]interface{}) template.URL {
	host := viewArgs["Host"].(string)
	return template.URL(createReverseURL(host, routeName, args))
}
