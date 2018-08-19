// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org/router source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package router

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"aahframework.org/ahttp"
	"aahframework.org/essentials"
	"aahframework.org/log"
	"aahframework.org/security"
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Domain
//___________________________________

// Domain is used to hold domain related routes and it's route configuration
type Domain struct {
	IsSubDomain           bool
	MethodNotAllowed      bool
	RedirectTrailingSlash bool
	AutoOptions           bool
	AntiCSRFEnabled       bool
	CORSEnabled           bool
	Key                   string
	Name                  string
	Host                  string
	Port                  string
	DefaultAuth           string
	CORS                  *CORS
	trees                 map[string]*tree
	routes                map[string]*Route
}

// Lookup method looks up route if found it returns route, path parameters,
// redirect trailing slash indicator for given `ahttp.Request` by domain
// and request URI otherwise returns nil and false.
func (d *Domain) Lookup(req *http.Request) (*Route, ahttp.URLParams, bool) {
	// HTTP method override support
	if req.Method == ahttp.MethodPost {
		if h := req.Header[ahttp.HeaderXHTTPMethodOverride]; len(h) > 0 {
			req.Method = h[0]
		}
	}

	// get route tree for request method
	tree, found := d.trees[req.Method]
	if !found {
		// get route tree for CORS access control method
		if req.Method == ahttp.MethodOptions && d.CORSEnabled {
			if h := req.Header[ahttp.HeaderAccessControlRequestMethod]; len(h) > 0 {
				tree, found = d.trees[h[0]]
			}
		}
		if !found {
			return nil, nil, false
		}
	}

	return tree.lookup(req.URL.Path)
}

// LookupByName method returns the route for given route name otherwise nil.
func (d *Domain) LookupByName(name string) *Route {
	if route, found := d.routes[name]; found {
		return route
	}
	return nil
}

// AddRoute method adds the given route into domain routing tree.
func (d *Domain) AddRoute(route *Route) error {
	if ess.IsStrEmpty(route.Method) {
		return errors.New("router: method value is empty")
	}

	t := d.trees[route.Method]
	if t == nil {
		t = &tree{root: new(node), tralingSlash: d.RedirectTrailingSlash}
		d.trees[route.Method] = t
	}

	if err := t.add(strings.ToLower(route.Path), route); err != nil {
		return err
	}

	d.routes[route.Name] = route
	return nil
}

// Allowed method returns the value for header `Allow` otherwise empty string.
func (d *Domain) Allowed(requestMethod, path string) (allowed string) {
	if path == "*" { // server-wide
		for method := range d.trees {
			if method != ahttp.MethodOptions {
				// add request method to list of allowed methods
				allowed = suffixCommaValue(allowed, method)
			}
		}
		return
	}

	// specific path
	for method := range d.trees {
		// Skip the requested method - we already tried this one
		if method != requestMethod && method != ahttp.MethodOptions {
			if value, _, _ := d.trees[method].lookup(path); value != nil {
				// add request method to list of allowed methods
				allowed = suffixCommaValue(allowed, method)
			}
		}
	}

	return
}

// RouteURLNamedArgs composes reverse URL by route name and key-value pair arguments.
// Additional key-value pairs composed as URL query string.
// If error occurs then method logs it and returns empty string.
func (d *Domain) RouteURLNamedArgs(routeName string, args map[string]interface{}) string {
	route, found := d.routes[routeName]
	if !found {
		log.Errorf("route name '%v' not found", routeName)
		return ""
	}

	argsLen := len(args)
	pathParamCnt := countParams(route.Path)
	if pathParamCnt == 0 && argsLen == 0 { // static URLs or no path params
		return route.Path
	}

	if argsLen < int(pathParamCnt) { // not enough arguments suppiled
		log.Errorf("not enough arguments, path: '%v' params count: %v, suppiled values count: %v",
			route.Path, pathParamCnt, argsLen)
		return ""
	}

	// compose URL with values
	reverseURL := "/"
	for _, segment := range strings.Split(route.Path, "/")[1:] {
		if len(segment) == 0 {
			continue
		}

		if segment[0] == paramByte || segment[0] == wildByte {
			argName := segment[1:]
			if arg, found := args[argName]; found {
				reverseURL = path.Join(reverseURL, fmt.Sprintf("%v", arg))
				delete(args, argName)
				continue
			}

			log.Errorf("'%v' param not found in given map", segment[1:])
			return ""
		}

		reverseURL = path.Join(reverseURL, segment)
	}

	// add remaining params into URL Query parameters, if any
	if len(args) > 0 {
		urlValues := url.Values{}

		for k, v := range args {
			urlValues.Add(k, fmt.Sprintf("%v", v))
		}

		reverseURL = fmt.Sprintf("%s?%s", reverseURL, urlValues.Encode())
	}

	return reverseURL
}

// RouteURL method composes route reverse URL for given route and
// arguments based on index order. If error occurs then method logs it
// and returns empty string.
func (d *Domain) RouteURL(routeName string, args ...interface{}) string {
	route, found := d.routes[routeName]
	if !found {
		log.Errorf("route name '%v' not found", routeName)
		return ""
	}

	argsLen := len(args)
	pathParamCnt := countParams(route.Path)
	if pathParamCnt == 0 && argsLen == 0 { // static URLs or no path params
		return route.Path
	}

	// too many arguments
	if argsLen > int(pathParamCnt) {
		log.Errorf("too many arguments, path: '%v' params count: %v, suppiled values count: %v",
			route.Path, pathParamCnt, argsLen)
		return ""
	}

	// not enough arguments
	if argsLen < int(pathParamCnt) {
		log.Errorf("not enough arguments, path: '%v' params count: %v, suppiled values count: %v",
			route.Path, pathParamCnt, argsLen)
		return ""
	}

	var values []string
	for _, v := range args {
		values = append(values, fmt.Sprintf("%v", v))
	}

	// compose URL with values
	reverseURL := "/"
	idx := 0
	for _, segment := range strings.Split(route.Path, "/") {
		if len(segment) == 0 {
			continue
		}

		if segment[0] == paramByte || segment[0] == wildByte {
			reverseURL = path.Join(reverseURL, values[idx])
			idx++
			continue
		}

		reverseURL = path.Join(reverseURL, segment)
	}

	return reverseURL
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Domain unexpoted methods
//___________________________________

func (d *Domain) inferKey() {
	if len(d.Port) == 0 {
		d.Key = strings.ToLower(d.Host)
	} else {
		d.Key = strings.ToLower(d.Host + ":" + d.Port)
	}
}

func (d *Domain) isAuthConfigured(secMgr *security.Manager) ([]string, bool) {
	if !ess.IsStrEmpty(d.DefaultAuth) && secMgr.AuthScheme(d.DefaultAuth) != nil {
		return []string{}, true
	}

	names := []string{}
	for _, r := range d.routes {
		if r.IsStatic || r.Auth == "anonymous" || r.Auth == "authenticated" || r.Method == "WS" {
			continue
		}

		if r.Auth == "" {
			names = append(names, r.Name)
		} else if secMgr.AuthScheme(r.Auth) == nil {
			names = append(names, r.Name)
		}
	}

	return names, len(names) == 0
}
