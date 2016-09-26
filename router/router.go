// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package router

import (
	"errors"
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
		"get":     "Index",
		"post":    "Create",
		"put":     "Update",
		"patch":   "Update",
		"delete":  "Delete",
		"options": "Options",
		"head":    "Head",
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
		name                  string
		port                  string
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

	// PathParam is single URL Path parameter (not a query string values)
	PathParam struct {
		Key   string
		Value string
	}

	// PathParams is a Param-slice, as returned by the route tree.
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

	for _, key := range domains {
		// fmt.Println("domain:", key)
		domain := newDomain(key)
		domainCfg, _ := r.config.GetSubConfig(key)

		// Global configuration
		domain.catchAll = domainCfg.BoolDefault("global.catch_all", true)
		domain.methodNotAllowed = domainCfg.BoolDefault("global.method_not_allowed", true)
		domain.redirectTrailingSlash = domainCfg.BoolDefault("global.redirect_trailing_slash", true)

		// TODO process not_found & panic
		notFoundInfo := createRouteInfo(domainCfg, "global.not_found")
		panicInfo := createRouteInfo(domainCfg, "global.panic")
		_ = notFoundInfo
		_ = panicInfo

		// Routes configuration

		// add domain routes
		r.domains[domain.key()] = domain
		// fmt.Println(domain.Trees["get"], domain.Trees["post"])
		// fmt.Println(domain.routes)
	} // End of domains

	return nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Domain methods
//___________________________________

func newDomain(key string) *Domain {
	portIdx := strings.Index(key, "__")
	if portIdx == -1 {
		return &Domain{
			name: strings.ToLower(strings.Replace(key, "_", ".", -1)),
		}
	}

	parts := strings.Split(key, "__")
	parts[0] = strings.ToLower(strings.Replace(parts[0], "_", ".", -1))
	return &Domain{name: parts[0], port: parts[1]}
}

func createRouteInfo(cfg *config.Config, routeName string) *Route {
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
	if ess.IsStrEmpty(d.port) {
		return strings.ToLower(d.name)
	}
	return strings.ToLower(d.name + ":" + d.port)
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
