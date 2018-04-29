// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/router source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

// Package router provides routing implementation for aah framework.
// Routes config file format is similar to HOCON syntax aka typesafe config
// it gets parsed by `go-aah/config`.
//
// aah framework router uses radix tree implementation of
// https://github.com/julienschmidt/httprouter
package router

import (
	"errors"
	"fmt"
	"path"
	"strings"

	"aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
)

const (
	wildcardSubdomainPrefix = "*."
	methodWebSocket         = "WS"
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

	// ErrNoDomainRoutesConfigFound returned when routes config file not found or doesn't
	// have `domains { ... }` config information.
	ErrNoDomainRoutesConfigFound = errors.New("router: no domain routes config found")
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package methods
//___________________________________

// New method returns the Router instance.
func New(configPath string, appCfg *config.Config) *Router {
	return &Router{
		configPath: configPath,
		appCfg:     appCfg,
	}
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
// Router
//___________________________________

// Router is used to register all application routes and finds the appropriate
// route information for incoming request path.
type Router struct {
	Domains      map[string]*Domain
	rootDomain   *Domain
	singleDomain *Domain
	addresses    []string
	configPath   string
	config       *config.Config
	appCfg       *config.Config
}

// Load method loads a configuration from given file e.g. `routes.conf` and
// applies env profile override values if available.
func (r *Router) Load() (err error) {
	if !ess.IsFileExists(r.configPath) {
		return fmt.Errorf("router: configuration does not exists: %v", r.configPath)
	}

	r.config, err = config.LoadFile(r.configPath)
	if err != nil {
		return err
	}

	// apply aah.conf env variables
	if envRoutesValues, found := r.appCfg.GetSubConfig("routes"); found {
		log.Debug("env routes {...} values found, applying it")
		if err = r.config.Merge(envRoutesValues); err != nil {
			return fmt.Errorf("router: routes.conf: %s", err)
		}
	}

	err = r.processRoutesConfig()
	return
}

// FindDomain returns domain routes configuration based on http request
// otherwise nil.
func (r *Router) FindDomain(req *ahttp.Request) *Domain {
	// DEPRECATED to be removed in v1.0 release
	return r.Lookup(req.Host)
}

// Lookup method returns domain for given host otherwise nil.
func (r *Router) Lookup(host string) *Domain {
	if r.singleDomain != nil {
		// only one domain scenario
		return r.singleDomain
	}
	host = strings.ToLower(host)

	// Extact match of host value
	// for e.g.: sample.com:8080, www.sample.com:8080, admin.sample.com:8080
	if domain, found := r.Domains[host]; found {
		return domain
	}

	// Wildcard match of host value
	// for e.g.: router.conf value is `*.sample.com:8080` it matches
	// {subdomain}.sample.com
	if idx := strings.IndexByte(host, '.'); idx > 0 {
		if domain, found := r.Domains[wildcardSubdomainPrefix+host[idx+1:]]; found {
			return domain
		}
	}

	return nil
}

// RootDomain method returns the root domain registered in the routes.conf.
// For e.g.: sample.com, admin.sample.com, *.sample.com.
// Root Domain is `sample.com`.
func (r *Router) RootDomain() *Domain {
	return r.rootDomain
}

// DomainAddresses method returns domain addresses (host:port) from
// routes configuration.
func (r *Router) DomainAddresses() []string {
	return r.addresses
}

// RegisteredActions method returns all the controller name and it's actions
// configured in the "routes.conf".
func (r *Router) RegisteredActions() map[string]map[string]uint8 {
	methods := map[string]map[string]uint8{}
	for _, d := range r.Domains {
		for _, route := range d.routes {
			if route.IsStatic || route.Method == methodWebSocket {
				continue
			}
			addRegisteredAction(methods, route)
		}
	}
	return methods
}

// RegisteredWSActions method returns all the WebSocket name and it's actions
// configured in the "routes.conf".
func (r *Router) RegisteredWSActions() map[string]map[string]uint8 {
	methods := map[string]map[string]uint8{}
	for _, d := range r.Domains {
		for _, route := range d.routes {
			if route.Method == methodWebSocket {
				addRegisteredAction(methods, route)
			}
		}
	}
	return methods
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Router unexpoted methods
//___________________________________

func (r *Router) processRoutesConfig() (err error) {
	domains := r.config.KeysByPath("domains")
	if len(domains) == 0 {
		return ErrNoDomainRoutesConfigFound
	}

	_ = r.config.SetProfile("domains")

	// allocate for no. of domains
	r.Domains = make(map[string]*Domain)
	log.Debugf("Domain count: %d", len(domains))

	for _, key := range domains {
		domainCfg, _ := r.config.GetSubConfig(key)

		// domain host name
		host, found := domainCfg.String("host")
		if !found {
			err = fmt.Errorf("'%v.host' key is missing", key)
			return
		}

		// Router takes the port-no in the order they found-
		//   1) routes.conf `domains.<domain-name>.port`
		//   2) aah.conf `server.port`
		//   3) 8080
		port := strings.TrimSpace(domainCfg.StringDefault("port",
			r.appCfg.StringDefault("server.port", "8080")))
		if port == "80" || port == "443" {
			port = ""
		}

		domain := &Domain{
			Name:                  domainCfg.StringDefault("name", key),
			Host:                  host,
			Port:                  port,
			IsSubDomain:           domainCfg.BoolDefault("subdomain", false),
			MethodNotAllowed:      domainCfg.BoolDefault("method_not_allowed", true),
			RedirectTrailingSlash: domainCfg.BoolDefault("redirect_trailing_slash", true),
			AutoOptions:           domainCfg.BoolDefault("auto_options", true),
			DefaultAuth:           domainCfg.StringDefault("default_auth", ""),
			AntiCSRFEnabled:       domainCfg.BoolDefault("anti_csrf_check", true),
			CORSEnabled:           domainCfg.BoolDefault("cors.enable", false),
			trees:                 make(map[string]*node),
			routes:                make(map[string]*Route),
		}

		// Domain Level CORS configuration
		if domain.CORSEnabled {
			baseCORSCfg, _ := domainCfg.GetSubConfig("cors")
			domain.CORS = processBaseCORSSection(baseCORSCfg)
		}

		// Not Found route support is removed in aah v0.8 release,
		// in-favor of Centralized Error Handler.
		// Refer to https://docs.aahframework.org/centralized-error-handler.html

		// processing static routes
		if err = r.processStaticRoutes(domain, domainCfg); err != nil {
			return
		}

		// processing namespace routes
		if err = r.processRoutes(domain, domainCfg); err != nil {
			return
		}

		// add domain routes
		key := domain.key()
		log.Debugf("Domain: %s, routes found: %d", key, len(domain.routes))
		if log.IsLevelTrace() {
			// don't spend time here, process only if log level is trace
			// Static Files routes
			log.Trace("Static Files Routes")
			for _, dr := range domain.routes {
				if dr.IsStatic {
					log.Tracef("Route Name: %v, Path: %v, IsDir: %v, Dir: %v, ListDir: %v, IsFile: %v, File: %v",
						dr.Name, dr.Path, dr.IsDir(), dr.Dir, dr.ListDir, dr.IsFile(), dr.File)
				}
			}

			// Application routes
			log.Trace("Application Routes")
			for _, dr := range domain.routes {
				if dr.IsStatic {
					continue
				}
				parentInfo := ""
				if !ess.IsStrEmpty(dr.ParentName) {
					parentInfo = fmt.Sprintf("(parent: %s)", dr.ParentName)
				}
				log.Tracef("Route Name: %v %v, Path: %v, Method: %v, Target: %v, Action: %v, Auth: %v, MaxBodySize: %v\nCORS: [%v]\nValidation Rules:%v\n",
					dr.Name, parentInfo, dr.Path, dr.Method, dr.Target, dr.Action, dr.Auth, dr.MaxBodySize,
					dr.CORS, dr.validationRules)
			}
		}

		r.Domains[key] = domain
		r.addresses = append(r.addresses, key)

	} // End of domains

	// Only one domain scenario
	if len(r.Domains) == 1 {
		r.singleDomain = r.Domains[r.addresses[0]]
	}

	// find out root domain
	// Note: Assuming of one domain and multiple sub-domains configured
	// otherwise it will have first non-subdomain reference.
	for _, d := range r.Domains {
		if !d.IsSubDomain {
			r.rootDomain = d
			break
		}
	}

	r.config.ClearProfile()
	return
}

func (r *Router) processStaticRoutes(domain *Domain, domainCfg *config.Config) error {
	staticCfg, found := domainCfg.GetSubConfig("static")
	if !found {
		return nil
	}

	routes, err := parseStaticSection(staticCfg)
	if err != nil {
		return err
	}

	for idx := range routes {
		if err = domain.AddRoute(routes[idx]); err != nil {
			return err
		}
	}

	return nil
}

func (r *Router) processRoutes(domain *Domain, domainCfg *config.Config) error {
	routesCfg, found := domainCfg.GetSubConfig("routes")
	if !found {
		return nil
	}

	routes, err := parseRoutesSection(routesCfg, &parentRouteInfo{
		Auth:          domain.DefaultAuth,
		CORS:          domain.CORS,
		AntiCSRFCheck: domain.AntiCSRFEnabled,
		CORSEnabled:   domain.CORSEnabled,
	})
	if err != nil {
		return err
	}

	for idx := range routes {
		if err = domain.AddRoute(routes[idx]); err != nil {
			return err
		}
	}
	return nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Route
//___________________________________

// Route holds the single route details.
type Route struct {
	IsAntiCSRFCheck bool
	IsStatic        bool
	ListDir         bool
	MaxBodySize     int64
	Name            string
	Path            string
	Method          string
	Target          string
	Action          string
	ParentName      string
	Auth            string
	Dir             string
	File            string
	CORS            *CORS

	validationRules map[string]string
}

type parentRouteInfo struct {
	ParentName    string
	PrefixPath    string
	Target        string
	Auth          string
	AntiCSRFCheck bool
	CORS          *CORS
	CORSEnabled   bool
}

// IsDir method returns true if serving directory otherwise false.
func (r *Route) IsDir() bool {
	return !ess.IsStrEmpty(r.Dir) && ess.IsStrEmpty(r.File)
}

// IsFile method returns true if serving single file otherwise false.
func (r *Route) IsFile() bool {
	return !ess.IsStrEmpty(r.File)
}

// ValidationRule methdo returns `validation rule, true` if exists for path param
// otherwise `"", false`
func (r *Route) ValidationRule(name string) (string, bool) {
	rules, found := r.validationRules[name]
	return rules, found
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

func parseRoutesSection(cfg *config.Config, routeInfo *parentRouteInfo) (routes []*Route, err error) {
	for _, routeName := range cfg.Keys() {
		// getting 'path'
		routePath, found := cfg.String(routeName + ".path")
		if !found && ess.IsStrEmpty(routeInfo.PrefixPath) {
			err = fmt.Errorf("'%v.path' key is missing", routeName)
			return
		}

		// path must begin with '/'
		if !ess.IsStrEmpty(routePath) && routePath[0] != slashByte {
			err = fmt.Errorf("'%v.path' [%v], path must begin with '/'", routeName, routePath)
			return
		}

		routePath = path.Join(routeInfo.PrefixPath, routePath)

		// Split validation rules from path params
		pathParamRules := make(map[string]string)
		actualRoutePath := "/"
		for _, seg := range strings.Split(routePath, "/")[1:] {
			if len(seg) == 0 {
				continue
			}

			if seg[0] == paramByte || seg[0] == wildByte {
				param, rules, exists, valid := checkValidationRule(seg)
				if exists {
					if valid {
						pathParamRules[param[1:]] = rules
					} else {
						err = fmt.Errorf("'%v.path' has invalid validation rule '%v'", routeName, routePath)
						return
					}
				}

				actualRoutePath = path.Join(actualRoutePath, param)
			} else {
				actualRoutePath = path.Join(actualRoutePath, seg)
			}
		}

		// getting 'method', default to GET, if method not found
		routeMethod := strings.ToUpper(cfg.StringDefault(routeName+".method", ahttp.MethodGet))

		// check child routes exists
		notToSkip := true
		if cfg.IsExists(routeName + ".routes") {
			if !cfg.IsExists(routeName+".action") &&
				(!cfg.IsExists(routeName+".controller") || !cfg.IsExists(routeName+".websocket")) {
				notToSkip = false
			}
		}

		// getting 'target' info for e.g.: controller, websocket
		routeTarget := cfg.StringDefault(routeName+".controller",
			cfg.StringDefault(routeName+".websocket", routeInfo.Target))
		if ess.IsStrEmpty(routeTarget) && notToSkip {
			err = fmt.Errorf("'%v.controller' or '%v.websocket' key is missing", routeName, routeName)
			return
		}

		// getting 'action', if not found it will default to `HTTPMethodActionMap`
		// based on `routeMethod`. For multiple HTTP method mapping scenario,
		// this is required attribute.
		routeAction := cfg.StringDefault(routeName+".action", findActionByHTTPMethod(routeMethod))
		if ess.IsStrEmpty(routeAction) && notToSkip {
			err = fmt.Errorf("'%v.action' key is missing, it seems to be multiple HTTP methods", routeName)
			return
		}

		// getting route authentication scheme name
		routeAuth := cfg.StringDefault(routeName+".auth", routeInfo.Auth)

		// getting route max body size, GitHub go-aah/aah#83
		routeMaxBodySize, er := ess.StrToBytes(cfg.StringDefault(routeName+".max_body_size", "0kb"))
		if er != nil {
			log.Warnf("'%v.max_body_size' value is not a valid size unit, fallback to global limit", routeName)
		}

		// getting Anti-CSRF check value, GitHub go-aah/aah#115
		routeAntiCSRFCheck := cfg.BoolDefault(routeName+".anti_csrf_check", routeInfo.AntiCSRFCheck)

		// CORS
		var cors *CORS
		if routeInfo.CORSEnabled && routeMethod != methodWebSocket {
			if corsCfg, found := cfg.GetSubConfig(routeName + ".cors"); found {
				if corsCfg.BoolDefault("enable", true) {
					cors = processCORSSection(corsCfg, routeInfo.CORS)
				}
			} else {
				cors = routeInfo.CORS
			}
		}

		// 'anti_csrf_check', 'cors' and 'max_body_size' not applicable for WebSocket
		if routeMethod == methodWebSocket {
			routeAntiCSRFCheck = false
			cors = nil
			routeMaxBodySize = 0
		}

		if notToSkip {
			for _, m := range strings.Split(routeMethod, ",") {
				routes = append(routes, &Route{
					Name:            routeName,
					Path:            actualRoutePath,
					Method:          strings.TrimSpace(m),
					Target:          routeTarget,
					Action:          routeAction,
					ParentName:      routeInfo.ParentName,
					Auth:            routeAuth,
					MaxBodySize:     routeMaxBodySize,
					IsAntiCSRFCheck: routeAntiCSRFCheck,
					CORS:            cors,
					validationRules: pathParamRules,
				})
			}
		}

		// loading child routes
		if childRoutes, found := cfg.GetSubConfig(routeName + ".routes"); found {
			croutes, er := parseRoutesSection(childRoutes, &parentRouteInfo{
				ParentName:    routeName,
				PrefixPath:    routePath,
				Target:        routeTarget,
				Auth:          routeAuth,
				AntiCSRFCheck: routeAntiCSRFCheck,
				CORS:          cors,
				CORSEnabled:   routeInfo.CORSEnabled,
			})
			if er != nil {
				err = er
				return
			}

			routes = append(routes, croutes...)
		}
	}

	return
}

func parseStaticSection(cfg *config.Config) (routes []*Route, err error) {
	for _, routeName := range cfg.Keys() {
		route := &Route{Name: routeName, Method: ahttp.MethodGet, IsStatic: true}

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

		if fileFound {
			// GitHub #141 - for a file mapping
			//  - 'base_dir' attribute value is not provided and
			//  - file 'path' value relative path
			// then use 'public_assets.dir' as a default value.
			if dir, found := cfg.String(routeName + ".base_dir"); found {
				routeDir = dir
			} else if routeFile[0] != slashByte { // relative file path mapping
				if dir, found := cfg.String("public_assets.dir"); found {
					routeDir = dir
				} else {
					err = fmt.Errorf("'static.%v.base_dir' value is missing", routeName)
					return
				}
			}
		}

		route.Dir = routeDir
		route.File = routeFile
		route.ListDir = cfg.BoolDefault(routeName+".list", false)

		routes = append(routes, route)
	}

	return
}
