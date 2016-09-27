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
		methodNotAllowed      bool
		redirectTrailingSlash bool
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

// Lookup finds a route information, path parameters, redirect trailing slash
// indicator for given `ahttp.Request` by domain and request URI
// otherwise returns nil and false.
func (r *Router) Lookup(req *ahttp.Request) (*Route, *PathParams, bool) {
	// get domain router for request
	domain := r.Domain(req)
	if domain == nil {
		return nil, nil, false
	}

	// get route tree for request method
	tree, found := domain.trees[req.Method]
	if !found {
		return nil, nil, false
	}

	routeName, pathParams, rts, err := tree.find(req.URL.Path)
	if routeName != nil && err == nil {
		return domain.routes[routeName.(string)], &pathParams, rts
	}

	log.Warnf("Error while route lookup: %v", err)
	return nil, nil, false
}

// LookupByName method to find route information by route name
func (r *Router) LookupByName(req *ahttp.Request, name string) *Route {
	domain := r.Domain(req)
	if domain == nil {
		return nil
	}

	if route, found := domain.routes[name]; found {
		return route
	}
	return nil
}

// Allowed returns the header value for `Allow` otherwise empty string.
func (r *Router) Allowed(req *ahttp.Request) string {
	domain := r.Domain(req)
	if domain == nil {
		return ""
	}
	return domain.allowed(req.Method, req.URL.Path)
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

		// domain name
		name := domainCfg.StringDefault("name", key)

		// domain host name
		host, found := domainCfg.String("host")
		if !found {
			err = fmt.Errorf("''%v.host' key is missing", key)
			return
		}

		// domain host port no.
		port := domainCfg.StringDefault("port", "")

		domain := &Domain{
			Name: name,
			Host: host,
			Port: port,
		}
		log.Tracef("Domain key: %v", domain.key())

		// loading global configuration
		if domainCfg.IsExists("global") {
			globalCfg, _ := domainCfg.GetSubConfig("global")

			domain.catchAll = globalCfg.BoolDefault("catch_all", true)
			domain.methodNotAllowed = globalCfg.BoolDefault("method_not_allowed", true)
			domain.redirectTrailingSlash = globalCfg.BoolDefault("redirect_trailing_slash", true)
			log.Tracef("Domain global config [catchAll: %v, methodNotAllowed: %v, redirectTrailingSlash: %v]",
				domain.catchAll, domain.methodNotAllowed, domain.redirectTrailingSlash)

			if domain.catchAll {
				// TODO catch all
			}

			// TODO process not_found & panic
			notFoundHandle := createRoute(globalCfg, "not_found")
			panicHandle := createRoute(globalCfg, "panic")
			_ = notFoundHandle
			_ = panicHandle
			// log.Tracef("Not found handler: %v.%v", notFoundHandle.Controller, notFoundHandle.Action)
			// log.Tracef("Panic handler: %v.%v", panicHandle.Controller, panicHandle.Action)
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

			for _, route := range routes {
				log.Tracef("Route Name: %v, Path: %v, Method: %v, Controller: %v, Action: %v",
					route.Name, route.Path, route.Method, route.Controller, route.Action)

				err = domain.addRoute(&route)
				if err != nil {
					return
				}
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

func createRoute(cfg *config.Config, routeName string) *Route {
	return nil
}

func (d *Domain) initIfNot() {
	if d.trees == nil {
		d.trees = make(map[string]*node)
	}

	if d.routes == nil {
		d.routes = make(map[string]*Route)
	}
}

func (d *Domain) key() string {
	if ess.IsStrEmpty(d.Port) {
		return strings.ToLower(d.Host)
	}
	return strings.ToLower(d.Host + ":" + d.Port)
}

func (d *Domain) addRoute(route *Route) error {
	d.initIfNot()

	tree := d.trees[route.Method]
	if tree == nil {
		tree = new(node)
		d.trees[route.Method] = tree
	}

	err := tree.add(route.Path, route.Name)
	if err == nil {
		d.routes[route.Name] = route
	}
	return err
}

func (d *Domain) allowed(requestMethod, path string) (allow string) {
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

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Route methods
//___________________________________

// Add method adds route into slice
func (r Routes) Add(route Route) {
	r = append(r, route)
	_ = r
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
