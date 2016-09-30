// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package router

import (
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/go-aah/aah/ahttp"
	"github.com/go-aah/config"
	"github.com/go-aah/essentials"
	"github.com/go-aah/log"
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
	}

	// Routes is a Route-slice
	Routes []Route

	// PathParam is single URL Path parameter (not a query string values)
	PathParam struct {
		Key   string
		Value string
	}

	// PathParams is a PathParam-slice, as returned by the route tree.
	PathParams []PathParam
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Router methods
//___________________________________

// New method creates a Router with given routes configuration path.
func New(configPath string) *Router {
	return &Router{configPath: configPath}
}

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

// Load method loads a configuration from `routes.conf`
func (r *Router) Load() (err error) {
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
	log.Tracef("No. of domain route configs found: %v", len(domains))

	for _, key := range domains {
		domainCfg, _ := r.config.GetSubConfig(key)

		// domain host name
		host, found := domainCfg.String("host")
		if !found {
			err = fmt.Errorf("''%v.host' key is missing", key)
			return
		}

		domain := &Domain{
			Name: domainCfg.StringDefault("name", key),
			Host: host,
			Port: domainCfg.StringDefault("port", ""),
		}
		log.Tracef("Domain key: %v", domain.key())

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
			log.Info("static routes exists")
		}

		// loading namespace routes
		if domainCfg.IsExists("routes") {
			routesCfg, _ := domainCfg.GetSubConfig("routes")
			// var routes Routes

			routes, er := parseRoutesSection(routesCfg, "")
			if er != nil {
				err = er
				return
			}
			log.Tracef("No. of routes found: %v", len(routes))

			for idx := range routes {
				route := routes[idx]
				if err = domain.addRoute(&route); err != nil {
					return
				}

				log.Tracef("Route Name: %v, Path: %v, Method: %v, Controller: %v, Action: %v",
					route.Name, route.Path, route.Method, route.Controller, route.Action)
			}
		}

		// add domain routes
		r.domains[domain.key()] = domain
		// fmt.Println(domain.trees["GET"], domain.trees["POST"])
		// fmt.Println(domain.routes)
	} // End of domains

	return
}

func parseRoutesSection(cfg *config.Config, prefixPath string) (routes Routes, err error) {
	for _, routeName := range cfg.Keys() {
		// getting 'path'
		routePath, found := cfg.String(routeName + ".path")
		if !found {
			err = fmt.Errorf("'%v.path' key is missing", routeName)
			return
		}

		// getting 'method'
		routeMethod, found := cfg.String(routeName + ".method")
		if !found {
			// default to GET, if method not found
			routeMethod = ahttp.MethodGet
		}

		// getting 'controller'
		routeController, found := cfg.String(routeName + ".controller")
		if !found {
			err = fmt.Errorf("'%v.controller' key is missing", routeName)
			return
		}

		// getting 'action', if not found it will default to `HTTPMethodActionMap`
		// based on `routeMethod`
		routeAction := cfg.StringDefault(routeName+".action", HTTPMethodActionMap[strings.ToUpper(routeMethod)])

		// TODO action params

		routes = append(routes, Route{
			Name:       routeName,
			Path:       path.Join(prefixPath, routePath),
			Method:     routeMethod,
			Controller: routeController,
			Action:     routeAction,
		})

		// loading child routes
		if childRoutes, found := cfg.GetSubConfig(routeName + ".routes"); found {
			croutes, er := parseRoutesSection(childRoutes, routePath)
			if er != nil {
				err = er
				return
			}

			routes = append(routes, croutes...)
		}
	}

	return
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Domain methods
//___________________________________

// Lookup finds a route information, path parameters, redirect trailing slash
// indicator for given `ahttp.Request` by domain and request URI
// otherwise returns nil and false.
func (d *Domain) Lookup(req *ahttp.Request) (*Route, *PathParams, bool) {
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

// Reverse composes reverse URL by route name and order of arguments or zero
// otherwise returns empty string
func (d *Domain) Reverse(routeName string, args ...interface{}) string {
	route, found := d.routes[routeName]
	if !found {
		return ""
	}

	routePath := route.Path
	argsLen := len(args)
	paramCnt := countParams(routePath)
	if paramCnt == 0 && argsLen == 0 { // static URLs
		return routePath
	} else if int(paramCnt) != argsLen { // not enough arguments suppiled
		log.Errorf("Not enough arguments, path: '%v' params count: %v, suppiled args count: %v",
			routePath, paramCnt, argsLen)
		return routePath
	}

	// compose URL with values
	url := "/"
	idx := 0
	for _, segment := range strings.Split(route.Path, "/") {
		if ess.IsStrEmpty(segment) {
			continue
		}

		if segment[0] == paramByte || segment[0] == wildByte {
			url = path.Join(url, fmt.Sprintf("%v", args[idx]))
			idx++
			continue
		}
		url = path.Join(url, segment)
	}

	return url
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
// Common unexported methods
//___________________________________

func suffixCommaValue(s, v string) string {
	if ess.IsStrEmpty(s) {
		s = v
	} else {
		s += ", " + v
	}
	return s
}
