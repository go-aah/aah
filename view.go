// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"fmt"
	"html/template"
	"path"
	"path/filepath"
	"strings"

	"aahframe.work/aah/ahttp"
	"aahframe.work/aah/essentials"
	"aahframe.work/aah/internal/util"
	"aahframe.work/aah/security"
	"aahframe.work/aah/view"
)

const (
	defaultViewEngineName = "go"
	defaultViewFileExt    = ".html"
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// app methods
//______________________________________________________________________________

func (a *app) ViewEngine() view.Enginer {
	if a.viewMgr == nil {
		return nil
	}
	return a.viewMgr.engine
}

func (a *app) AddTemplateFunc(funcs template.FuncMap) {
	view.AddTemplateFunc(funcs)
}

func (a *app) AddViewEngine(name string, engine view.Enginer) error {
	return view.AddEngine(name, engine)
}

func (a *app) SetMinifier(fn MinifierFunc) {
	if a.viewMgr == nil {
		a.viewMgr = &viewManager{a: a}
	}

	if a.viewMgr.minifier != nil {
		a.Log().Warnf("Changing Minifier from: '%s'  to '%s'",
			ess.GetFunctionInfo(a.viewMgr.minifier).QualifiedName, ess.GetFunctionInfo(fn).QualifiedName)
	}
	a.viewMgr.minifier = fn
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// app Unexported methods
//______________________________________________________________________________

func (a *app) initView() error {
	viewsDir := path.Join(a.VirtualBaseDir(), "views")
	if !a.VFS().IsExists(viewsDir) {
		// view directory not exists, scenario could be API, WebSocket application
		a.SecurityManager().AntiCSRF.Enabled = false
		return nil
	}

	engineName := a.Config().StringDefault("view.engine", defaultViewEngineName)
	viewEngine, found := view.GetEngine(engineName)
	if !found {
		return fmt.Errorf("view: named engine not found: %s", engineName)
	}

	viewMgr := &viewManager{
		a:                     a,
		engineName:            engineName,
		fileExt:               a.Config().StringDefault("view.ext", defaultViewFileExt),
		defaultTmplLayout:     "master" + a.Config().StringDefault("view.ext", defaultViewFileExt),
		filenameCaseSensitive: a.Config().BoolDefault("view.case_sensitive", false),
		defaultLayoutEnabled:  a.Config().BoolDefault("view.default_layout", true),
		notFoundTmpl: template.Must(template.New("not_found").Parse(`
		<strong>{{ .ViewNotFound }}</strong>
	`)),
	}

	// Add Framework template methods
	a.AddTemplateFunc(template.FuncMap{
		"config":          viewMgr.tmplConfig,
		"i18n":            viewMgr.tmplI18n,
		"rurl":            viewMgr.tmplURL,
		"rurlm":           viewMgr.tmplURLm,
		"pparam":          viewMgr.tmplPathParam,
		"fparam":          viewMgr.tmplFormParam,
		"qparam":          viewMgr.tmplQueryParam,
		"session":         viewMgr.tmplSessionValue,
		"flash":           viewMgr.tmplFlashValue,
		"isauthenticated": viewMgr.tmplIsAuthenticated,
		"hasrole":         viewMgr.tmplHasRole,
		"hasallroles":     viewMgr.tmplHasAllRoles,
		"hasanyrole":      viewMgr.tmplHasAnyRole,
		"ispermitted":     viewMgr.tmplIsPermitted,
		"ispermittedall":  viewMgr.tmplIsPermittedAll,
		"anticsrftoken":   viewMgr.tmplAntiCSRFToken,
	})

	if err := viewEngine.Init(a.VFS(), a.Config(), viewsDir); err != nil {
		return err
	}

	viewMgr.engine = viewEngine
	if a.viewMgr != nil && a.viewMgr.minifier != nil {
		viewMgr.minifier = a.viewMgr.minifier
	}

	a.viewMgr = viewMgr
	a.SecurityManager().AntiCSRF.Enabled = true
	a.viewMgr.setHotReload(a.IsProfileDev() && !a.IsPackaged())

	return nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// View Manager
//______________________________________________________________________________

type viewManager struct {
	a                     *app
	engineName            string
	engine                view.Enginer
	fileExt               string
	defaultTmplLayout     string
	filenameCaseSensitive bool
	defaultLayoutEnabled  bool
	notFoundTmpl          *template.Template
	minifier              MinifierFunc
}

// resolve method resolves the view template based available facts, such as
// controller name, action and user provided inputs.
func (vm *viewManager) resolve(ctx *Context) {
	// Resolving view by convention and configuration
	reply := ctx.Reply()
	if reply.Rdr == nil {
		reply.Rdr = &htmlRender{}
	}

	htmlRdr, ok := reply.Rdr.(*htmlRender)
	if !ok || htmlRdr.Template != nil {
		// 1. If its not type `htmlRender`, possibly custom render implementation
		// 2. Template already populated in it
		// So no need to go forward
		return
	}

	if len(htmlRdr.Layout) == 0 && vm.defaultLayoutEnabled {
		htmlRdr.Layout = vm.defaultTmplLayout
	}

	if htmlRdr.ViewArgs == nil {
		htmlRdr.ViewArgs = make(map[string]interface{})
	}

	for k, v := range ctx.ViewArgs() {
		if _, found := htmlRdr.ViewArgs[k]; found {
			continue
		}
		htmlRdr.ViewArgs[k] = v
	}

	// Add ViewArgs values from framework
	vm.addFrameworkValuesIntoViewArgs(ctx)

	var tmplPath, tmplName string

	// If user not provided the template info, auto resolve by convention
	if len(htmlRdr.Filename) == 0 {
		tmplName = ctx.action.Name + vm.fileExt
		tmplPath = filepath.Join(ctx.controller.Namespace, ctx.controller.NoSuffixName)
	} else {
		// User provided view info like layout, filename.
		// Taking full-control of view rendering.
		// Scenario's:
		//  1. filename with relative path
		//  2. filename with root page path
		tmplName = filepath.Base(htmlRdr.Filename)
		tmplPath = filepath.Dir(htmlRdr.Filename)

		if strings.HasPrefix(htmlRdr.Filename, "/") {
			tmplPath = strings.TrimLeft(tmplPath, "/")
		} else {
			tmplPath = filepath.Join(ctx.controller.Namespace, ctx.controller.NoSuffixName, tmplPath)
		}
	}

	tmplPath = filepath.Join("pages", tmplPath)

	ctx.Log().Tracef("view(layout:%s path:%s name:%s)", htmlRdr.Layout, tmplPath, tmplName)
	var err error
	if htmlRdr.Template, err = vm.engine.Get(htmlRdr.Layout, tmplPath, tmplName); err != nil {
		if err == view.ErrTemplateNotFound {
			tmplFile := filepath.Join("views", tmplPath, tmplName)
			if !vm.filenameCaseSensitive {
				tmplFile = strings.ToLower(tmplFile)
			}

			ctx.Log().Errorf("template not found: %s", tmplFile)
			if vm.a.IsProfileProd() {
				htmlRdr.ViewArgs["ViewNotFound"] = "View Not Found"
			} else {
				htmlRdr.ViewArgs["ViewNotFound"] = "View Not Found: " + tmplFile
			}
			htmlRdr.Layout = ""
			htmlRdr.Template = vm.notFoundTmpl
		} else {
			ctx.Log().Error(err)
		}
	}
}

func (vm *viewManager) addFrameworkValuesIntoViewArgs(ctx *Context) {
	html := ctx.Reply().Rdr.(*htmlRender)
	html.ViewArgs["Scheme"] = ctx.Req.Scheme
	html.ViewArgs["Host"] = ctx.Req.Host
	html.ViewArgs["HTTPMethod"] = ctx.Req.Method
	html.ViewArgs["RequestPath"] = ctx.Req.Path
	html.ViewArgs["Locale"] = ctx.Req.Locale()
	html.ViewArgs["ClientIP"] = ctx.Req.ClientIP()
	html.ViewArgs["IsJSONP"] = ctx.Req.IsJSONP()
	html.ViewArgs["IsAJAX"] = ctx.Req.IsAJAX()
	html.ViewArgs["HTTPReferer"] = ctx.Req.Referer
	html.ViewArgs["AahVersion"] = Version
	html.ViewArgs[KeyViewArgRequest] = ctx.Req
	if ctx.subject != nil {
		html.ViewArgs[KeyViewArgSubject] = ctx.Subject()
	}

	html.ViewArgs["EnvProfile"] = vm.a.Profile()
	html.ViewArgs["AppBuildInfo"] = vm.a.BuildInfo()
}

func (vm *viewManager) setHotReload(v bool) {
	if hr, ok := vm.engine.(interface {
		SetHotReload(r bool)
	}); ok {
		hr.SetHotReload(v)
	}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// View Template methods
//______________________________________________________________________________

//
// Request Parameters
//

// tmplPathParam method returns Request Path Param value for the given key.
func (vm *viewManager) tmplPathParam(viewArgs map[string]interface{}, key string) interface{} {
	return vm.tmplRequestParameters(viewArgs, "P", key)
}

// tmplFormParam method returns Request Form value for the given key.
func (vm *viewManager) tmplFormParam(viewArgs map[string]interface{}, key string) interface{} {
	return vm.tmplRequestParameters(viewArgs, "F", key)
}

// tmplQueryParam method returns Request Query String value for the given key.
func (vm *viewManager) tmplQueryParam(viewArgs map[string]interface{}, key string) interface{} {
	return vm.tmplRequestParameters(viewArgs, "Q", key)
}

func (vm *viewManager) tmplRequestParameters(viewArgs map[string]interface{}, fn, key string) interface{} {
	req := viewArgs[KeyViewArgRequest].(*ahttp.Request)
	switch fn {
	case "Q":
		return util.SanitizeValue(req.QueryValue(key))
	case "F":
		return util.SanitizeValue(req.FormValue(key))
	case "P":
		return util.SanitizeValue(req.PathValue(key))
	}
	return ""
}

//
// Configuration view functions
//

// tmplConfig method provides access to application config on templates.
func (vm *viewManager) tmplConfig(key string) interface{} {
	if value, found := vm.a.Config().Get(key); found {
		return util.SanitizeValue(value)
	}
	vm.a.Log().Warnf("Configuration key not found: '%s'", key)
	return ""
}

//
// i18n view functions
//

// tmplI18n method is mapped to Go template func for resolving i18n values.
func (vm *viewManager) tmplI18n(viewArgs map[string]interface{}, key string, args ...interface{}) string {
	if locale, ok := viewArgs[keyLocale].(*ahttp.Locale); ok {
		if len(args) == 0 {
			return vm.a.I18n().Lookup(locale, key)
		}

		sanatizeArgs := make([]interface{}, 0)
		for _, value := range args {
			sanatizeArgs = append(sanatizeArgs, util.SanitizeValue(value))
		}
		return vm.a.I18n().Lookup(locale, key, sanatizeArgs...)
	}
	return ""
}

//
// Route view functions
//

// tmplURL method returns reverse URL by given route name and args.
// Mapped to Go template func.
func (vm *viewManager) tmplURL(viewArgs map[string]interface{}, args ...interface{}) template.URL {
	if len(args) == 0 {
		vm.a.Log().Errorf("router: template 'rurl' - route name is empty: %v", args)
		return template.URL("#")
	}
	domain, routeName := vm.a.findRouteURLDomain(viewArgs["Host"].(string), args[0].(string))
	/* #nosec */
	return template.URL(createRouteURL(vm.a.Log(), domain, routeName, nil, args[1:]...))
}

// tmplURLm method returns reverse URL by given route name and
// map[string]interface{}. Mapped to Go template func.
func (vm *viewManager) tmplURLm(viewArgs map[string]interface{}, routeName string, args map[string]interface{}) template.URL {
	domain, rn := vm.a.findRouteURLDomain(viewArgs["Host"].(string), routeName)
	/* #nosec */
	return template.URL(createRouteURL(vm.a.Log(), domain, rn, args))
}

//
// Session and Flash view functions
//

// tmplSessionValue method returns session value for the given key. If session
// object unavailable this method returns nil.
func (vm *viewManager) tmplSessionValue(viewArgs map[string]interface{}, key string) interface{} {
	if sub := vm.getSubjectFromViewArgs(viewArgs); sub != nil {
		if sub.Session != nil {
			value := sub.Session.Get(key)
			return util.SanitizeValue(value)
		}
	}
	return nil
}

// tmplFlashValue method returns session value for the given key. If session
// object unavailable this method returns nil.
func (vm *viewManager) tmplFlashValue(viewArgs map[string]interface{}, key string) interface{} {
	if sub := vm.getSubjectFromViewArgs(viewArgs); sub != nil {
		if sub.Session != nil {
			return util.SanitizeValue(sub.Session.GetFlash(key))
		}
	}
	return nil
}

//
// Security view functions
//

// tmplIsAuthenticated method returns the value of `Session.IsAuthenticated`.
func (vm *viewManager) tmplIsAuthenticated(viewArgs map[string]interface{}) bool {
	if sub := vm.getSubjectFromViewArgs(viewArgs); sub != nil {
		if sub.Session != nil {
			return sub.Session.IsAuthenticated
		}
	}
	return false
}

// tmplHasRole method returns the value of `Subject.HasRole`.
func (vm *viewManager) tmplHasRole(viewArgs map[string]interface{}, role string) bool {
	if sub := vm.getSubjectFromViewArgs(viewArgs); sub != nil {
		return sub.HasRole(role)
	}
	return false
}

// tmplHasAllRoles method returns the value of `Subject.HasAllRoles`.
func (vm *viewManager) tmplHasAllRoles(viewArgs map[string]interface{}, roles ...string) bool {
	if sub := vm.getSubjectFromViewArgs(viewArgs); sub != nil {
		return sub.HasAllRoles(roles...)
	}
	return false
}

// tmplHasAnyRole method returns the value of `Subject.HasAnyRole`.
func (vm *viewManager) tmplHasAnyRole(viewArgs map[string]interface{}, roles ...string) bool {
	if sub := vm.getSubjectFromViewArgs(viewArgs); sub != nil {
		return sub.HasAnyRole(roles...)
	}
	return false
}

// tmplIsPermitted method returns the value of `Subject.IsPermitted`.
func (vm *viewManager) tmplIsPermitted(viewArgs map[string]interface{}, permission string) bool {
	if sub := vm.getSubjectFromViewArgs(viewArgs); sub != nil {
		return sub.IsPermitted(permission)
	}
	return false
}

// tmplIsPermittedAll method returns the value of `Subject.IsPermittedAll`.
func (vm *viewManager) tmplIsPermittedAll(viewArgs map[string]interface{}, permissions ...string) bool {
	if sub := vm.getSubjectFromViewArgs(viewArgs); sub != nil {
		return sub.IsPermittedAll(permissions...)
	}
	return false
}

// tmplAntiCSRFToken method returns the salted Anti-CSRF secret for the view,
// if enabled otherwise empty string.
func (vm *viewManager) tmplAntiCSRFToken(viewArgs map[string]interface{}) string {
	if vm.a.SecurityManager().AntiCSRF.Enabled {
		if cs, found := viewArgs[keyAntiCSRF]; found {
			return vm.a.SecurityManager().AntiCSRF.SaltCipherSecret(cs.([]byte))
		}
	}
	return ""
}

func (vm *viewManager) getSubjectFromViewArgs(viewArgs map[string]interface{}) *security.Subject {
	if sv, found := viewArgs[KeyViewArgSubject]; found {
		return sv.(*security.Subject)
	}
	return nil
}
