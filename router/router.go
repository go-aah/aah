// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org/router source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

// Package router provides routing implementation for aah framework.
// Routes config file format is similar to HOCON syntax aka typesafe config
// it gets parsed by `aahframework.org/config`.
//
// aah router internally uses customized version of radix tree implementation from
// `github.com/julienschmidt/httprouter` developer by `@julienschmidt`.
package router

import (
	"errors"
	"fmt"
	"path"
	"regexp"
	"strings"

	"aahframework.org/ahttp"
	"aahframework.org/config"
	"aahframework.org/essentials"
	"aahframework.org/log"
	"aahframework.org/security"
	"aahframework.org/security/scheme"
	"aahframework.org/vfs"
)

const (
	wildcardSubdomainPrefix = "*."
	methodWebSocket         = "WS"
	autoRouteNameSuffix     = "__aah"
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

	// ErrRouteConstraintFailed returned when request route constraints failed.
	ErrRouteConstraintFailed = errors.New("router: route constraints failed")
)

// aah application interface for minimal purpose
type application interface {
	Config() *config.Config
	Log() log.Loggerer
	VFS() *vfs.VFS
	SecurityManager() *security.Manager
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package methods
//______________________________________________________________________________

// New method returns the Router instance.
func New(configPath string, appCfg *config.Config) *Router {
	return &Router{
		configPath: configPath,
		aCfg:       appCfg,
	}
}

// NewWithApp method creates router instance with aah application instance.
func NewWithApp(app interface{}, configPath string) (*Router, error) {
	a, ok := app.(application)
	if !ok {
		return nil, fmt.Errorf("router: not a valid aah application instance")
	}

	rtr := &Router{configPath: configPath, app: a}
	if err := rtr.Load(); err != nil {
		return nil, err
	}

	return rtr, nil
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

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Router
//______________________________________________________________________________

// Router is used to register all application routes and finds the appropriate
// route information for incoming request path.
type Router struct {
	Domains []*Domain

	configPath string
	rootDomain *Domain
	app        application
	config     *config.Config
	aCfg       *config.Config // kept for backward purpose, to be removed in subsequent release
}

// Load method loads a configuration from given file e.g. `routes.conf` and
// applies env profile override values if available.
func (r *Router) Load() (err error) {
	if !r.isExists(r.configPath) {
		return fmt.Errorf("router: configuration does not exists: %v", r.configPath)
	}

	r.config, err = r.readConfig(r.configPath)
	if err != nil {
		return err
	}

	// apply aah.conf env variables
	if envRoutesValues, found := r.appConfig().GetSubConfig("routes"); found {
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
	if len(r.Domains) == 1 {
		return r.Domains[0] // only one domain scenario
	}

	// Extact match of host value
	// for e.g.: sample.com:8080, www.sample.com:8080, admin.sample.com:8080
	if domain := r.findDomain(host); domain != nil {
		return domain
	}

	// Wildcard match of host value
	// for e.g.: router.conf value is `*.sample.com:8080` it matches
	// {subdomain}.sample.com
	if idx := strings.IndexByte(host, '.'); idx > 0 {
		return r.findDomain(wildcardSubdomainPrefix + host[idx+1:])
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
	var addresses []string
	for _, d := range r.Domains {
		addresses = append(addresses, d.Key)
	}
	return addresses
}

// RegisteredActions method returns all the controller name and it's actions
// configured in the "routes.conf".
func (r *Router) RegisteredActions() map[string]map[string]uint8 {
	methods := map[string]map[string]uint8{}
	for _, d := range r.Domains {
		for _, route := range d.routes {
			if route.IsStatic || route.Method == methodWebSocket ||
				strings.HasSuffix(route.Name, autoRouteNameSuffix) {
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

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Router unexpoted methods
//______________________________________________________________________________

func (r *Router) findDomain(key string) *Domain {
	key = strings.ToLower(key)
	for _, d := range r.Domains {
		if d.Key == key {
			return d
		}
	}
	return nil
}

func (r *Router) isExists(name string) bool {
	if r.app == nil {
		return vfs.IsExists(nil, name)
	}
	return vfs.IsExists(r.app.VFS(), name)
}

func (r *Router) readConfig(name string) (*config.Config, error) {
	if r.app == nil {
		return config.LoadFile(name)
	}
	return config.VFSLoadFile(r.app.VFS(), r.configPath)
}

func (r *Router) processRoutesConfig() (err error) {
	domains := r.config.KeysByPath("domains")
	if len(domains) == 0 {
		return ErrNoDomainRoutesConfigFound
	}

	_ = r.config.SetProfile("domains")

	// allocate for no. of domains
	r.Domains = make([]*Domain, len(domains))
	log.Debugf("Domain count: %d", len(domains))

	for idx, key := range domains {
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
			r.appConfig().StringDefault("server.port", "8080")))
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
			trees:                 make(map[string]*tree),
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
		domain.inferKey()
		log.Debugf("Domain: %s, routes found: %d", domain.Key, len(domain.routes))
		if log.IsLevelTrace() { // process only if log level is trace
			// Static Files routes
			log.Trace("Routes: Static")
			for _, dr := range domain.routes {
				if dr.IsStatic {
					log.Trace(dr)
				}
			}

			// Application routes
			log.Trace("Routes: Application")
			for _, dr := range domain.routes {
				if !dr.IsStatic {
					log.Trace(dr)
				}
			}
		}

		r.Domains[idx] = domain
		for _, t := range domain.trees {
			t.root.inferwnode()
		}
	} // End of domains

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

	maxBodySizeStr := r.appConfig().StringDefault("request.max_body_size", "5mb")
	routes, err := parseSectionRoutes(routesCfg, &parentRouteInfo{
		Auth:              domain.DefaultAuth,
		MaxBodySizeStr:    maxBodySizeStr,
		CORS:              domain.CORS,
		AntiCSRFCheck:     domain.AntiCSRFEnabled,
		CORSEnabled:       domain.CORSEnabled,
		AuthorizationInfo: &authorizationInfo{Satisfy: "either"},
	})
	if err != nil {
		return err
	}

	for idx := range routes {
		if err = domain.AddRoute(routes[idx]); err != nil {
			return err
		}
	}

	// Add form login route per security.conf for configured domains
	if r.app != nil && r.app.SecurityManager() != nil {
		authSchemes := r.app.SecurityManager().AuthSchemes()
		if len(authSchemes) > 0 {
			if routeNames, result := domain.isAuthConfigured(r.app.SecurityManager()); !result {
				log.Errorf("Auth schemes are configured in 'security.conf', however "+
					"these routes have invaild auth scheme or not configured: %s",
					strings.Join(routeNames, ", "))
				return fmt.Errorf("routes configuration error in domain '%s', please check the logs", domain.Name)
			}
		}

		for kn, s := range authSchemes {
			switch sv := s.(type) {
			case *scheme.FormAuth:
				maxBodySize, _ := ess.StrToBytes(maxBodySizeStr)
				name := kn + "_login_submit" + autoRouteNameSuffix // for e.g.: form_auth_login_submit__aah
				if domain.LookupByName(name) == nil {              // add only if not exists
					_ = domain.AddRoute(&Route{Name: name, Path: sv.LoginSubmitURL,
						Method: ahttp.MethodPost, Auth: kn, MaxBodySize: maxBodySize})
				}
			case *scheme.OAuth2:
				_ = domain.AddRoute(&Route{
					Name:   kn + "_login" + autoRouteNameSuffix,
					Path:   sv.LoginURL,
					Method: ahttp.MethodGet,
					Auth:   kn,
				})
				_ = domain.AddRoute(&Route{
					Name:   kn + "_redirect" + autoRouteNameSuffix,
					Path:   sv.RedirectURL,
					Method: ahttp.MethodGet,
					Auth:   kn,
				})
			}
		}
	}

	return nil
}

func (r *Router) appConfig() *config.Config {
	if r.aCfg == nil {
		return r.app.Config()
	}
	return r.aCfg
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//______________________________________________________________________________

var payloadSupported = regexp.MustCompile(`(POST|PUT|DELETE)`)

func parseSectionRoutes(cfg *config.Config, routeInfo *parentRouteInfo) (routes []*Route, err error) {
	for _, routeName := range cfg.Keys() {
		// getting 'path'
		routePath, found := cfg.String(routeName + ".path")
		if !found && ess.IsStrEmpty(routeInfo.PrefixPath) {
			err = fmt.Errorf("'%v.path' key is missing", routeName)
			return
		}

		if found && routePath[0] == '^' {
			routePath = addSlashPrefix(routePath[1:])
		} else {
			routePath = path.Join(routeInfo.PrefixPath, addSlashPrefix(routePath))
		}
		routePath = path.Clean(strings.TrimSpace(routePath))

		// route segment parameter constraints
		actualRoutePath, routeConstraints, er := parseRouteConstraints(routeName, routePath)
		if er != nil {
			err = er
			return
		}

		// getting 'method', default to GET, if method not found
		routeMethod := strings.ToUpper(cfg.StringDefault(routeName+".method", ahttp.MethodGet))

		// getting 'target' info for e.g.: controller, websocket
		routeTarget := cfg.StringDefault(routeName+".controller", cfg.StringDefault(routeName+".websocket", routeInfo.Target))

		// getting 'action', if not found it will default to `HTTPMethodActionMap`
		// based on `routeMethod`. For multiple HTTP method mapping scenario,
		// this is required attribute.
		routeAction := cfg.StringDefault(routeName+".action", findActionByHTTPMethod(routeMethod))

		notToSkip := true
		if cfg.IsExists(routeName + ".routes") {
			if ess.IsStrEmpty(routeTarget) || ess.IsStrEmpty(routeAction) {
				notToSkip = false
			}
		}

		if notToSkip && ess.IsStrEmpty(routeTarget) {
			err = fmt.Errorf("'%v.controller' or '%v.websocket' key is missing", routeName, routeName)
			return
		}
		if notToSkip && ess.IsStrEmpty(routeAction) {
			err = fmt.Errorf("'%v.action' key is missing or it seems to be multiple HTTP methods", routeName)
			return
		}

		// getting route authentication scheme name
		routeAuth := strings.TrimSpace(cfg.StringDefault(routeName+".auth", routeInfo.Auth))

		// getting route max body size, GitHub go-aah/aah#83
		routeMaxBodySize, er := ess.StrToBytes(cfg.StringDefault(routeName+".max_body_size", routeInfo.MaxBodySizeStr))
		if er != nil {
			log.Warnf("'%v.max_body_size' value is not a valid size unit, fallback to global limit", routeName)
		}
		if !payloadSupported.MatchString(routeMethod) {
			routeMaxBodySize = 0
		}

		// getting Anti-CSRF check value, GitHub go-aah/aah#115
		routeAntiCSRFCheck := cfg.BoolDefault(routeName+".anti_csrf_check", routeInfo.AntiCSRFCheck)

		// Authorization Info
		routeAuthorizationInfo, er := parseAuthorizationInfo(cfg, routeName, routeInfo)
		if er != nil {
			err = er
			return
		}

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
					Name:              routeName,
					Path:              actualRoutePath,
					Method:            strings.TrimSpace(m),
					Target:            routeTarget,
					Action:            routeAction,
					ParentName:        routeInfo.ParentName,
					Auth:              routeAuth,
					MaxBodySize:       routeMaxBodySize,
					IsAntiCSRFCheck:   routeAntiCSRFCheck,
					CORS:              cors,
					Constraints:       routeConstraints,
					authorizationInfo: routeAuthorizationInfo,
				})
			}
		}

		// loading child routes
		if childRoutes, found := cfg.GetSubConfig(routeName + ".routes"); found {
			croutes, er := parseSectionRoutes(childRoutes, &parentRouteInfo{
				ParentName:        routeName,
				PrefixPath:        routePath,
				Target:            routeTarget,
				Auth:              routeAuth,
				MaxBodySizeStr:    routeInfo.MaxBodySizeStr,
				AntiCSRFCheck:     routeAntiCSRFCheck,
				CORS:              cors,
				CORSEnabled:       routeInfo.CORSEnabled,
				AuthorizationInfo: routeAuthorizationInfo,
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

		// add route if directory found and list dir is enabled
		if route.ListDir && dirFound {
			rt := *route
			rt.Path = path.Clean(routePath) + "/"
			routes = append(routes, &rt)
		}

		routes = append(routes, route)
	}

	return
}
