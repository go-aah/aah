// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

// Package router provides routes implementation for aah framework application.
// Routes config format is `forge` config syntax (go-aah/config) which
// is similar to HOCON syntax aka typesafe config.
//
// aah framework router uses radix tree of
// https://github.com/julienschmidt/httprouter
package router

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"

	"aahframework.org/aah/ahttp"
	"aahframework.org/config"
	"aahframework.org/essentials"
	"aahframework.org/log"
)

var (
	// HTTPMethodActionMap is default Controller Action name for corresponding
	// HTTP Method. If it's not provided in the route configuration.
	HTTPMethodActionMap = map[string]string{
		ahttp.MethodGet:     "Index",
		ahttp.MethodPost:    "Create",
		ahttp.MethodPut:     "Update",
		ahttp.MethodPatch:   "Update",
		ahttp.MethodDelete:  "Delete",
		ahttp.MethodOptions: "Options",
		ahttp.MethodHead:    "Head",
		ahttp.MethodTrace:   "Trace",
	}

	// ErrNoRoutesConfigFound returned when routes config file not found or doesn't
	// have config information.
	ErrNoRoutesConfigFound = errors.New("no domain routes config found")
)

type (
	// Router is used to register all application routes and finds the appropriate
	// route information for incoming request path.
	Router struct {
		domains    map[string]*Domain
		configPath string
		config     *config.Config
	}

	// Domain is used to hold domain related routes and it's route configuration
	Domain struct {
		Name                  string
		Host                  string
		Port                  string
		NotFoundRoute         *Route
		PanicRoute            *Route
		MethodNotAllowed      bool
		RedirectTrailingSlash bool
		catchAll              bool
		trees                 map[string]*node
		routes                map[string]*Route
	}

	// Route holds the single route details.
	Route struct {
		Name         string
		Path         string
		Method       string
		Controller   string
		Action       string
		ActionParams []string
		ParentName   string

		// static route fields in-addition to above
		IsStatic bool
		Dir      string
		File     string
		ListDir  bool
	}

	// Routes is a Route-slice
	Routes []Route

	// PathParam is single URL path parameter (not a query string values)
	PathParam struct {
		Key   string
		Value string
	}

	// PathParams is a PathParam-slice, as returned by the route tree.
	PathParams []PathParam
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Global methods
//___________________________________

// New method creates a Router with given routes configuration path.
func New(configPath string) *Router {
	return &Router{configPath: configPath}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Router methods
//___________________________________

// Domain returns domain routes configuration based on http request
// otherwise nil.
func (r *Router) Domain(req *ahttp.Request) *Domain {
	if domain, found := r.domains[strings.ToLower(req.Host)]; found {
		return domain
	}
	return nil
}

// Reload method clears existing routes configuration and loads from
// routes configuration.
func (r *Router) Reload() error {
	// clean it
	r.domains = make(map[string]*Domain)
	r.config = nil

	// load fresh
	return r.Load()
}

// DomainAddresses method returns domain addresses (host:port) from
// routes configuration.
func (r *Router) DomainAddresses() []string {
	var addresses []string

	for k := range r.domains {
		addresses = append(addresses, k)
	}

	return addresses
}

// Load method loads a configuration from `routes.conf`
func (r *Router) Load() (err error) {
	log.Info("Loading routes config ...")
	r.config, err = config.LoadFile(r.configPath)
	if err != nil {
		return err
	}

	domains := r.config.KeysByPath("domains")
	if len(domains) == 0 {
		return ErrNoRoutesConfigFound
	}

	_ = r.config.SetProfile("domains")

	// allocate for no. of domains
	r.domains = make(map[string]*Domain, len(domains))
	log.Debugf("No. of domain found: %v", len(domains))

	for _, key := range domains {
		log.Debug("-----------------------------")
		domainCfg, _ := r.config.GetSubConfig(key)

		// domain host name
		host, found := domainCfg.String("host")
		if !found {
			err = fmt.Errorf("'%v.host' key is missing", key)
			return
		}

		domain := &Domain{
			Name: domainCfg.StringDefault("name", key),
			Host: host,
			Port: domainCfg.StringDefault("port", ""),
		}
		log.Debugf("Domain key: %v", domain.key())

		// loading global configuration
		if domainCfg.IsExists("global") {
			globalCfg, _ := domainCfg.GetSubConfig("global")

			domain.catchAll = globalCfg.BoolDefault("catch_all", true)
			domain.MethodNotAllowed = globalCfg.BoolDefault("method_not_allowed", true)
			domain.RedirectTrailingSlash = globalCfg.BoolDefault("redirect_trailing_slash", true)
			log.Tracef("Domain global config [catchAll: %v, methodNotAllowed: %v, redirectTrailingSlash: %v]",
				domain.catchAll, domain.MethodNotAllowed, domain.RedirectTrailingSlash)

			if domain.catchAll {
				domain.addCatchAllRoutes()
			}

			// not found route
			if globalCfg.IsExists("not_found") {
				domain.NotFoundRoute, err = createGlobalRoute(globalCfg, "not_found")
				if err != nil {
					return
				}

				log.Tracef("Not found route: %v.%v", domain.NotFoundRoute.Controller,
					domain.NotFoundRoute.Action)
			}

			// not found route
			if globalCfg.IsExists("panic") {
				domain.PanicRoute, err = createGlobalRoute(globalCfg, "panic")
				if err != nil {
					return
				}

				log.Tracef("Panic route: %v.%v", domain.PanicRoute.Controller,
					domain.PanicRoute.Action)
			}
		}

		// loading static routes
		if domainCfg.IsExists("static") {
			staticCfg, _ := domainCfg.GetSubConfig("static")

			routes, er := parseStaticRoutesSection(staticCfg)
			if er != nil {
				err = er
				return
			}

			log.Debugf("No. of static routes found: %v", len(routes))

			for idx := range routes {
				route := routes[idx]
				if err = domain.addRoute(&route); err != nil {
					return
				}

				log.Tracef("Route Name: %v, Path: %v, Dir: %v, ListDir: %v, File: %v",
					route.Name, route.Path, route.Dir, route.ListDir, route.File)
			}
		}

		// loading namespace routes
		if domainCfg.IsExists("routes") {
			routesCfg, _ := domainCfg.GetSubConfig("routes")

			routes, er := parseRoutesSection(routesCfg, "", "")
			if er != nil {
				err = er
				return
			}
			log.Debugf("No. of routes found: %v", len(routes))

			for idx := range routes {
				route := routes[idx]
				if err = domain.addRoute(&route); err != nil {
					return
				}

				log.Tracef("Route Name: %v (%v), Path: %v, Method: %v, Controller: %v, Action: %v",
					route.Name, route.ParentName, route.Path, route.Method, route.Controller, route.Action)
			}
		}

		// add domain routes
		r.domains[domain.key()] = domain

	} // End of domains

	return
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Domain methods
//___________________________________

// Lookup finds a route information, path parameters, redirect trailing slash
// indicator for given `ahttp.Request` by domain and request URI
// otherwise returns nil and false.
func (d *Domain) Lookup(req *ahttp.Request) (*Route, *PathParams, bool) {
	// HTTP method override support
	overrideMethod := req.Header.Get(ahttp.HeaderXHTTPMethodOverride)
	if !ess.IsStrEmpty(overrideMethod) && req.Method == ahttp.MethodPost {
		req.Method = overrideMethod
	}

	// get route tree for request method
	tree, found := d.trees[req.Method]
	if !found {
		return nil, nil, false
	}

	routeName, pathParams, rts, err := tree.find(req.URL.Path)
	if routeName != nil && err == nil {
		return d.routes[routeName.(string)], &pathParams, rts
	} else if rts { // possible Redirect Trailing Slash
		return nil, nil, rts
	}

	log.Warnf("Route lookup error: %v", err)
	return nil, nil, false
}

// LookupByName method to find route information by route name
func (d *Domain) LookupByName(name string) *Route {
	if route, found := d.routes[name]; found {
		return route
	}
	return nil
}

// Allowed returns the header value for `Allow` otherwise empty string.
func (d *Domain) Allowed(requestMethod, path string) (allow string) {
	if path == "*" { // server-wide
		for method := range d.trees {
			if method == ahttp.MethodOptions {
				continue
			}

			// add request method to list of allowed methods
			allow = suffixCommaValue(allow, method)
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
				allow = suffixCommaValue(allow, method)
			}
		}
	}

	allow = suffixCommaValue(allow, ahttp.MethodOptions)
	return
}

// Reverse composes reverse URL by route name and key-value pair arguments or
// zero argument for static URL. If anything goes wrong then method logs
// error info and returns empty string
func (d *Domain) Reverse(routeName string, args ...interface{}) string {
	route, found := d.routes[routeName]
	if !found {
		log.Errorf("route name '%v' not found", routeName)
		return ""
	}

	if len(args) > 1 {
		log.Error("expected no. of arguments is 1 and key-value pair")
		return ""
	}

	// routePath := route.Path
	argsLen := len(args)
	pathParamCnt := countParams(route.Path)
	if pathParamCnt == 0 && argsLen == 0 { // static URLs
		return route.Path
	}

	if int(pathParamCnt) != argsLen { // not enough arguments suppiled
		log.Errorf("not enough arguments, path: '%v' params count: %v, suppiled values count: %v",
			route.Path, pathParamCnt, argsLen)
		return ""
	}

	keyValues, ok := args[0].(map[string]string)
	if !ok {
		log.Error("key-value pair expected")
		return ""
	}

	// compose URL with values
	reverseURL := "/"
	for _, segment := range strings.Split(route.Path, "/") {
		if ess.IsStrEmpty(segment) {
			continue
		}

		if segment[0] == paramByte || segment[0] == wildByte {
			argName := segment[1:]
			if arg, found := keyValues[argName]; found {
				reverseURL = path.Join(reverseURL, arg)
				delete(keyValues, argName)
				continue
			}

			log.Errorf("'%v' param not found in given map", segment[1:])
			return ""
		}
		reverseURL = path.Join(reverseURL, segment)
	}

	// add remaining params into URL Query parameters, if any
	if len(keyValues) > 0 {
		urlValues := url.Values{}

		for k, v := range keyValues {
			urlValues.Add(k, v)
		}

		reverseURL = fmt.Sprintf("%s?%s", reverseURL, urlValues.Encode())
	}

	return reverseURL
}

func createGlobalRoute(cfg *config.Config, routeName string) (*Route, error) {
	controller, found := cfg.String(routeName + ".controller")
	if !found {
		return nil, fmt.Errorf("'%v.controller' key is missing", routeName)
	}

	action, found := cfg.String(routeName + ".action")
	if !found {
		return nil, fmt.Errorf("'%v.action' key is missing", routeName)
	}

	return &Route{
		Controller: controller,
		Action:     action,
	}, nil
}

func (d *Domain) key() string {
	if ess.IsStrEmpty(d.Port) {
		return strings.ToLower(d.Host)
	}
	return strings.ToLower(d.Host + ":" + d.Port)
}

func (d *Domain) addRoute(route *Route) error {
	if d.trees == nil {
		d.trees = make(map[string]*node)
	}

	if d.routes == nil {
		d.routes = make(map[string]*Route)
	}

	tree := d.trees[route.Method]
	if tree == nil {
		tree = new(node)
		d.trees[route.Method] = tree
	}

	if err := tree.add(route.Path, route.Name); err != nil {
		return err
	}

	d.routes[route.Name] = route
	return nil
}

func (d *Domain) addCatchAllRoutes() {
	// TODO add it
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Path Param methods
//___________________________________

// Get returns the value of the first Path Param which key matches the
// given name. Otherwise an empty string is returned.
func (pp PathParams) Get(name string) string {
	for _, p := range pp {
		if p.Key == name {
			return p.Value
		}
	}
	return ""
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

func suffixCommaValue(s, v string) string {
	if ess.IsStrEmpty(s) {
		s = v
	} else {
		s += ", " + v
	}
	return s
}

func parseRoutesSection(cfg *config.Config, parentName, prefixPath string) (routes Routes, err error) {
	for _, routeName := range cfg.Keys() {
		// getting 'path'
		routePath, found := cfg.String(routeName + ".path")
		if !found {
			err = fmt.Errorf("'%v.path' key is missing", routeName)
			return
		}

		// path must begin with '/'
		if routePath[0] != '/' {
			err = fmt.Errorf("'%v.path' [%v], path must begin with '/'", routeName, routePath)
			return
		}

		// getting 'method', default to GET, if method not found
		routeMethod := strings.ToUpper(cfg.StringDefault(routeName+".method", ahttp.MethodGet))

		// getting 'controller'
		routeController, found := cfg.String(routeName + ".controller")
		if !found {
			err = fmt.Errorf("'%v.controller' key is missing", routeName)
			return
		}

		// getting 'action', if not found it will default to `HTTPMethodActionMap`
		// based on `routeMethod`
		routeAction := cfg.StringDefault(routeName+".action", HTTPMethodActionMap[routeMethod])

		// TODO action params

		routes = append(routes, Route{
			Name:       routeName,
			Path:       path.Join(prefixPath, routePath),
			Method:     routeMethod,
			Controller: routeController,
			Action:     routeAction,
			ParentName: parentName,
		})

		// loading child routes
		if childRoutes, found := cfg.GetSubConfig(routeName + ".routes"); found {
			croutes, er := parseRoutesSection(childRoutes, routeName, routePath)
			if er != nil {
				err = er
				return
			}

			routes = append(routes, croutes...)
		}
	}

	return
}

func parseStaticRoutesSection(cfg *config.Config) (routes Routes, err error) {
	for _, routeName := range cfg.Keys() {
		route := Route{Name: routeName, Method: ahttp.MethodGet, IsStatic: true}

		// getting 'path'
		routePath, found := cfg.String(routeName + ".path")
		if !found {
			err = fmt.Errorf("'%v.path' key is missing", routeName)
			return
		}

		// path must begin with '/'
		if routePath[0] != '/' {
			err = fmt.Errorf("'%v.path' [%v], path must begin with '/'", routeName, routePath)
			return
		}

		if strings.Contains(routePath, ":") || strings.Contains(routePath, "*") {
			err = fmt.Errorf("'%v.path' parameters can not be used with static", routeName)
			return
		}

		route.Path = routePath

		routeDir, dirFound := cfg.String(routeName + ".dir")
		routeFile, fileFound := cfg.String(routeName + ".file")
		if dirFound && fileFound {
			err = fmt.Errorf("'%v.dir' & '%v.file' key(s) cannot be used together", routeName, routeName)
			return
		}

		if !dirFound && !fileFound {
			err = fmt.Errorf("'%v.dir' or '%v.file' key have to be present", routeName, routeName)
			return
		}

		if dirFound {
			route.Path = path.Join(route.Path, "*filepath")
		}

		route.Dir = routeDir
		route.File = routeFile
		route.ListDir = cfg.BoolDefault(routeName+".list", false)

		// TODO controller name and action

		routes = append(routes, route)
	}

	return
}
