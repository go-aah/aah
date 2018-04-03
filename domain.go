// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/router source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package router

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"

	"aahframework.org/ahttp.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Domain
//___________________________________

// Domain is used to hold domain related routes and it's route configuration
type Domain struct {
	Name                  string
	Host                  string
	Port                  string
	IsSubDomain           bool
	MethodNotAllowed      bool
	RedirectTrailingSlash bool
	AutoOptions           bool
	DefaultAuth           string
	CORS                  *CORS
	CORSEnabled           bool
	trees                 map[string]*node
	routes                map[string]*Route
}

// Lookup method finds a route, path parameters, redirect trailing slash
// indicator for given `ahttp.Request` by domain and request URI
// otherwise returns nil and false.
func (d *Domain) Lookup(req *ahttp.Request) (*Route, ahttp.PathParams, bool) {
	// HTTP method override support
	overrideMethod := req.Header.Get(ahttp.HeaderXHTTPMethodOverride)
	if !ess.IsStrEmpty(overrideMethod) && req.Method == ahttp.MethodPost {
		req.Method = overrideMethod
	}

	// get route tree for request method
	tree, found := d.lookupRouteTree(req)
	if !found {
		return nil, nil, false
	}

	route, pathParams, rts, err := tree.find(req.Path)
	if route != nil && err == nil {
		return route.(*Route), pathParams, rts
	} else if rts { // possible Redirect Trailing Slash
		return nil, nil, rts
	}

	return nil, nil, false
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

	tree := d.trees[route.Method]
	if tree == nil {
		tree = new(node)
		d.trees[route.Method] = tree
	}

	if err := tree.add(route.Path, route); err != nil {
		return err
	}

	d.routes[route.Name] = route
	return nil
}

// Allowed method returns the value for header `Allow` otherwise empty string.
func (d *Domain) Allowed(requestMethod, path string) (allowed string) {
	if path == "*" { // server-wide
		for method := range d.trees {
			if method == ahttp.MethodOptions {
				continue
			}

			// add request method to list of allowed methods
			allowed = suffixCommaValue(allowed, method)
		}
	} else { // specific path
		for method := range d.trees {
			// Skip the requested method - we already tried this one
			if method == requestMethod || method == ahttp.MethodOptions {
				continue
			}

			value, _, _, _ := d.trees[method].find(path)
			if value != nil {
				// add request method to list of allowed methods
				allowed = suffixCommaValue(allowed, method)
			}
		}
	}

	return
}

// ReverseURLm composes reverse URL by route name and key-value pair arguments.
// Additional key-value pairs composed as URL query string.
// If error occurs then method logs it and returns empty string.
func (d *Domain) ReverseURLm(routeName string, args map[string]interface{}) string {
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
		if ess.IsStrEmpty(segment) {
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

	rURL, err := url.Parse(reverseURL)
	if err != nil {
		log.Error(err)
		return ""
	}

	return rURL.String()
}

// ReverseURL method composes route reverse URL for given route and
// arguments based on index order. If error occurs then method logs it
// and returns empty string.
func (d *Domain) ReverseURL(routeName string, args ...interface{}) string {
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
		if ess.IsStrEmpty(segment) {
			continue
		}

		if segment[0] == paramByte || segment[0] == wildByte {
			reverseURL = path.Join(reverseURL, values[idx])
			idx++
			continue
		}

		reverseURL = path.Join(reverseURL, segment)
	}

	rURL, err := url.Parse(reverseURL)
	if err != nil {
		log.Error(err)
		return ""
	}

	return rURL.String()
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Domain unexpoted methods
//___________________________________

func (d *Domain) key() string {
	if d.Port == "" {
		return strings.ToLower(d.Host)
	}
	return strings.ToLower(d.Host + ":" + d.Port)
}

func (d *Domain) lookupRouteTree(req *ahttp.Request) (*node, bool) {
	// get route tree for request method
	if tree, found := d.trees[req.Method]; found {
		return tree, true
	}

	// get route tree for CORS access control method
	if req.Method == ahttp.MethodOptions && d.CORSEnabled {
		if tree, found := d.trees[req.Header.Get(ahttp.HeaderAccessControlRequestMethod)]; found {
			return tree, true
		}
	}

	return nil, false
}
