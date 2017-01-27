// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
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
	"html/template"
	"net/url"
	"path"
	"strings"

	"aahframework.org/aah/ahttp"
	"aahframework.org/aah/atemplate"
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

	router *Router
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
		AutoOptions           bool
		catchAll              bool
		trees                 map[string]*node
		routes                map[string]*Route
	}

	// Route holds the single route details.
	Route struct {
		Name       string
		Path       string
		Method     string
		Controller string
		Action     string
		ParentName string

		// static route fields in-addition to above
		IsStatic bool
		Dir      string
		File     string
		ListDir  bool
	}

	// Routes is a Route-slice type
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

// Load method loads a configuration from given file e.g. `routes.conf`
func Load(configPath string) (err error) {
	if !ess.IsFileExists(configPath) {
		return fmt.Errorf("aah application routes configuration does not exists: %v", configPath)
	}

	router = &Router{configPath: configPath}

	router.config, err = config.LoadFile(router.configPath)
	if err != nil {
		return err
	}

	domains := router.config.KeysByPath("domains")
	if len(domains) == 0 {
		return ErrNoRoutesConfigFound
	}

	_ = router.config.SetProfile("domains")

	// allocate for no. of domains
	router.domains = make(map[string]*Domain, len(domains))
	log.Debugf("No. of domains found: %v", len(domains))

	for _, key := range domains {
		domainCfg, _ := router.config.GetSubConfig(key)

		// domain host name
		host, found := domainCfg.String("host")
		if !found {
			err = fmt.Errorf("'%v.host' key is missing", key)
			return
		}

		domain := &Domain{
			Name:   domainCfg.StringDefault("name", key),
			Host:   host,
			Port:   domainCfg.StringDefault("port", ""),
			trees:  make(map[string]*node),
			routes: make(map[string]*Route),
		}
		log.Debugf("Domain: %v", domain.key())

		// loading global configuration
		if domainCfg.IsExists("global") {
			globalCfg, _ := domainCfg.GetSubConfig("global")

			domain.catchAll = globalCfg.BoolDefault("catch_all", true)
			domain.MethodNotAllowed = globalCfg.BoolDefault("method_not_allowed", true)
			domain.RedirectTrailingSlash = globalCfg.BoolDefault("redirect_trailing_slash", true)
			domain.AutoOptions = globalCfg.BoolDefault("auto_options", true)
			log.Tracef("Domain global config "+
				"[catchAll: %v, methodNotAllowed: %v, redirectTrailingSlash: %v, autoOptions]",
				domain.catchAll, domain.MethodNotAllowed, domain.RedirectTrailingSlash, domain.AutoOptions)

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

			log.Debugf("Static routes found: %v", len(routes))

			for idx := range routes {
				route := routes[idx]
				if err = domain.addRoute(&route); err != nil {
					return
				}

				log.Tracef("Static:: Route Name: %v, Path: %v, Dir: %v, ListDir: %v, File: %v",
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
			log.Debugf("Routes found: %v", len(routes))

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
		router.domains[domain.key()] = domain

	} // End of domains

	return
}

// Reload method clears existing routes configuration and loads from
// routes configuration.
func Reload() error {
	// clean it
	router.domains = make(map[string]*Domain)
	router.config = nil

	// load fresh
	configPath := router.configPath
	return Load(configPath)
}

// FindDomain returns domain routes configuration based on http request
// otherwise nil.
func FindDomain(req *ahttp.Request) *Domain {
	if domain, found := router.domains[strings.ToLower(req.Host)]; found {
		return domain
	}
	return nil
}

// DomainAddresses method returns domain addresses (host:port) from
// routes configuration.
func DomainAddresses() []string {
	var addresses []string

	for k := range router.domains {
		addresses = append(addresses, k)
	}

	return addresses
}

// RegisteredActions method returns all the controller name and it's actions
// configured in the "routes.conf".
func RegisteredActions() map[string]map[string]uint8 {
	methods := map[string]map[string]uint8{}
	for _, d := range router.domains {
		for _, route := range d.routes {
			if route.IsStatic {
				continue
			}

			addRegisteredAction(methods, route)
		}

		// adding not found controller if present
		if d.NotFoundRoute != nil {
			addRegisteredAction(methods, d.NotFoundRoute)
		}
	}

	return methods
}

// IsDefaultAction method is to identify given action name is defined by
// aah framework in absence of user configured route action name.
func IsDefaultAction(action string) bool {
	for _, a := range HTTPMethodActionMap {
		if a == action {
			return true
		}
	}
	return false
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

	routeName, pathParams, rts, err := tree.find(req.Path)
	if routeName != nil && err == nil {
		return d.routes[routeName.(string)], &pathParams, rts
	} else if rts { // possible Redirect Trailing Slash
		return nil, nil, rts
	}

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

// ReverseURLm composes reverse URL by route name and key-value pair arguments or
// zero argument for static URL. Additional key-values composed as URL query
// string. If error occurs then method logs an error and returns empty string.
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
// arguments based on index order. If error occurs then method logs
// an error and returns empty string.
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

func (d *Domain) key() string {
	if ess.IsStrEmpty(d.Port) {
		return strings.ToLower(d.Host)
	}
	return strings.ToLower(d.Host + ":" + d.Port)
}

func (d *Domain) addRoute(route *Route) error {
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
	// TODO implement CatchAllRoutes
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Route methods
//___________________________________

// IsDir method returns true if serving directory otherwise false.
func (r *Route) IsDir() bool {
	return !ess.IsStrEmpty(r.Dir)
}

// IsFile method returns true if serving single file otherwise false.
func (r *Route) IsFile() bool {
	return !ess.IsStrEmpty(r.File)
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

func addRegisteredAction(methods map[string]map[string]uint8, route *Route) {
	if controller, found := methods[route.Controller]; found {
		controller[route.Action] = 1
	} else {
		methods[route.Controller] = map[string]uint8{route.Action: 1}
	}
}

func createGlobalRoute(cfg *config.Config, routeName string) (*Route, error) {
	controller, found := cfg.String(routeName + ".controller")
	if !found {
		return nil, fmt.Errorf("'global.%v.controller' key is missing", routeName)
	}

	action, found := cfg.String(routeName + ".action")
	if !found {
		return nil, fmt.Errorf("'global.%v.action' key is missing", routeName)
	}

	return &Route{
		Controller: controller,
		Action:     action,
	}, nil
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
			err = fmt.Errorf("'static.%v.path' key is missing", routeName)
			return
		}

		// path must begin with '/'
		if routePath[0] != slashByte {
			err = fmt.Errorf("'static.%v.path' [%v], path must begin with '/'", routeName, routePath)
			return
		}

		if strings.Contains(routePath, ":") || strings.Contains(routePath, "*") {
			err = fmt.Errorf("'static.%v.path' parameters can not be used with static", routeName)
			return
		}

		route.Path = path.Clean(routePath)

		routeDir, dirFound := cfg.String(routeName + ".dir")
		routeFile, fileFound := cfg.String(routeName + ".file")
		if dirFound && fileFound {
			err = fmt.Errorf("'static.%v.dir' & 'static.%v.file' key(s) cannot be used together", routeName, routeName)
			return
		}

		if !dirFound && !fileFound {
			err = fmt.Errorf("either 'static.%v.dir' or 'static.%v.file' key have to be present", routeName, routeName)
			return
		}

		if dirFound {
			route.Path = path.Join(route.Path, "*filepath")
		}

		route.Dir = routeDir
		route.File = routeFile
		route.ListDir = cfg.BoolDefault(routeName+".list", false)

		routes = append(routes, route)
	}

	return
}

// tmplURL method returns reverse URL by given route name and args.
// Mapped to Go template func.
func tmplURL(viewArgs map[string]interface{}, args ...interface{}) template.URL {
	if len(args) > 0 {
		host := viewArgs["Host"].(string)
		if domain, found := router.domains[host]; found {
			return template.URL(domain.ReverseURL(args[0].(string), args[1:]...))
		}
	}
	log.Errorf("route not found: %v", args)
	return template.URL("")
}

func init() {
	atemplate.AddTemplateFunc(template.FuncMap{
		"url": tmplURL,
	})
}
