// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"fmt"
	"html/template"
	"path/filepath"
	"strings"

	"aahframework.org/essentials.v0"
	"aahframework.org/view.v0"
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
		a.Log().Warnf("Changing Minifier from: '%s'  to '%s'", funcName(a.viewMgr.minifier), funcName(fn))
	}
	a.viewMgr.minifier = fn
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// app Unexported methods
//______________________________________________________________________________

func (a *app) initView() error {
	viewsDir := "/app/views"
	if !a.VFS().IsExists(viewsDir) {
		// view directory not exists, scenario could be only API application
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
		"anitcsrftoken":   viewMgr.tmplAntiCSRFToken,
	})

	if err := viewEngine.Init(a.VFS(), a.Config(), viewsDir); err != nil {
		return err
	}

	viewMgr.engine = viewEngine
	if a.viewMgr != nil && a.viewMgr.minifier != nil {
		viewMgr.minifier = a.viewMgr.minifier
	}

	a.viewMgr = viewMgr

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

	if ess.IsStrEmpty(htmlRdr.Layout) && vm.defaultLayoutEnabled {
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
	if ess.IsStrEmpty(htmlRdr.Filename) {
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

	ctx.Log().Tracef("Layout: %s, Template Path: %s, Template Name: %s", htmlRdr.Layout, tmplPath, tmplName)
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
	html.ViewArgs[KeyViewArgRequestParams] = ctx.Req.Params
	if ctx.subject != nil {
		html.ViewArgs[KeyViewArgSubject] = ctx.Subject()
	}

	html.ViewArgs["EnvProfile"] = vm.a.Profile()
	html.ViewArgs["AppBuildInfo"] = vm.a.BuildInfo()
}
