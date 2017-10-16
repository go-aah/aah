// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"fmt"
	"html/template"
	"path/filepath"
	"strings"

	"aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
	"aahframework.org/view.v0"
)

const appDefaultViewEngine = "go"

var (
	appViewEngine             view.Enginer
	appViewExt                string
	appDefaultTmplLayout      string
	appIsDefaultLayoutEnabled bool
	appViewFileCaseSensitive  bool
	appIsExternalTmplEngine   bool
	viewNotFoundTemplate      = template.Must(template.New("not_found").Parse(`
		<strong>{{ .ViewNotFound }}</strong>
	`))
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package methods
//___________________________________

// AppViewEngine method returns aah application view Engine instance.
func AppViewEngine() view.Enginer {
	return appViewEngine
}

// AddTemplateFunc method adds template func map into view engine.
func AddTemplateFunc(funcs template.FuncMap) {
	view.AddTemplateFunc(funcs)
}

// AddViewEngine method adds the given name and view engine to view store.
func AddViewEngine(name string, engine view.Enginer) error {
	return view.AddEngine(name, engine)
}

// SetMinifier method sets the given minifier func into aah framework.
// Note: currently minifier is called only for HTML contentType.
func SetMinifier(fn MinifierFunc) {
	if minifier == nil {
		minifier = fn
	} else {
		log.Warnf("Minifier is already set: %v", funcName(minifier))
	}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

func appViewsDir() string {
	return filepath.Join(AppBaseDir(), "views")
}

func initViewEngine(viewDir string, appCfg *config.Config) error {
	if !ess.IsFileExists(viewDir) {
		// view directory not exists
		return nil
	}

	// application config values
	appViewExt = appCfg.StringDefault("view.ext", ".html")
	appDefaultTmplLayout = "master" + appViewExt
	appViewFileCaseSensitive = appCfg.BoolDefault("view.case_sensitive", false)
	appIsDefaultLayoutEnabled = appCfg.BoolDefault("view.default_layout", true)

	// initialize view engine
	viewEngineName := appCfg.StringDefault("view.engine", appDefaultViewEngine)
	viewEngine, found := view.GetEngine(viewEngineName)
	if !found {
		return fmt.Errorf("view: named engine not found: %s", viewEngineName)
	}

	if err := viewEngine.Init(appCfg, viewDir); err != nil {
		return err
	}

	appIsExternalTmplEngine = viewEngineName != appDefaultViewEngine
	appViewEngine = viewEngine

	return nil
}

// resolveView method does -
//   1) Prepare ViewArgs
//   2) If HTML content type find appropriate template
func (e *engine) resolveView(ctx *Context) {
	if appViewEngine == nil || ctx.Reply().err != nil || !ctHTML.IsEqual(ctx.Reply().ContType) {
		return
	}

	// Resolving view by convention and configuration
	reply := ctx.Reply()
	if reply.Rdr == nil {
		reply.Rdr = &HTML{}
	}

	htmlRdr := reply.Rdr.(*HTML)

	if ess.IsStrEmpty(htmlRdr.Layout) && appIsDefaultLayoutEnabled {
		htmlRdr.Layout = appDefaultTmplLayout
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

	// ViewArgs values from framework
	addFrameworkValuesIntoViewArgs(ctx, htmlRdr)

	// find view template by convention if not provided
	findViewTemplate(ctx)
}

// defaultContentType method returns the Content-Type based on 'render.default'
// config from aah.conf
func defaultContentType() *ahttp.ContentType {
	switch AppConfig().StringDefault("render.default", "") {
	case "html":
		return ahttp.ContentTypeHTML
	case "json":
		return ahttp.ContentTypeJSON
	case "xml":
		return ahttp.ContentTypeXML
	case "text":
		return ahttp.ContentTypePlainText
	default:
		return nil
	}
}

func findViewTemplate(ctx *Context) {
	htmlRdr := ctx.Reply().Rdr.(*HTML)
	var tmplPath, tmplName string

	// If user not provided the template info, auto resolve by convention
	if ess.IsStrEmpty(htmlRdr.Filename) {
		tmplName = ctx.action.Name + appViewExt
		tmplPath = filepath.Join(ctx.controller.Namespace, tmplControllerName(ctx))
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
			tmplPath = filepath.Join(ctx.controller.Namespace, tmplControllerName(ctx), tmplPath)
		}
	}

	tmplPath = filepath.Join("pages", tmplPath)

	ctx.Log().Tracef("Layout: %s, Template Path: %s, Template Name: %s", htmlRdr.Layout, tmplPath, tmplName)
	var err error
	if htmlRdr.Template, err = appViewEngine.Get(htmlRdr.Layout, tmplPath, tmplName); err != nil {
		if err == view.ErrTemplateNotFound {
			tmplFile := filepath.Join("views", tmplPath, tmplName)
			if !appViewFileCaseSensitive {
				tmplFile = strings.ToLower(tmplFile)
			}

			ctx.Log().Errorf("template not found: %s", tmplFile)
			if appIsProfileProd {
				htmlRdr.ViewArgs["ViewNotFound"] = "View Not Found"
			} else {
				htmlRdr.ViewArgs["ViewNotFound"] = "View Not Found: " + tmplFile
			}
			htmlRdr.Layout = ""
			htmlRdr.Template = viewNotFoundTemplate
		} else {
			ctx.Log().Error(err)
		}
	}
}

// sanatizeValue method sanatizes string type value, rest we can't do any.
// It's a user responbility.
func sanatizeValue(value interface{}) interface{} {
	switch v := value.(type) {
	case string:
		return template.HTMLEscapeString(v)
	default:
		return v
	}
}

func tmplControllerName(ctx *Context) string {
	cName := ctx.controller.Name()

	if strings.HasSuffix(cName, controllerNameSuffix) {
		cName = cName[:len(cName)-controllerNameSuffixLen]
	}
	return cName
}

func addFrameworkValuesIntoViewArgs(ctx *Context, html *HTML) {
	html.ViewArgs["Scheme"] = ctx.Req.Scheme
	html.ViewArgs["Host"] = ctx.Req.Host
	html.ViewArgs["HTTPMethod"] = ctx.Req.Method
	html.ViewArgs["RequestPath"] = ctx.Req.Path
	html.ViewArgs["Locale"] = ctx.Req.Locale
	html.ViewArgs["ClientIP"] = ctx.Req.ClientIP
	html.ViewArgs["IsJSONP"] = ctx.Req.IsJSONP()
	html.ViewArgs["IsAJAX"] = ctx.Req.IsAJAX()
	html.ViewArgs["HTTPReferer"] = ctx.Req.Referer
	html.ViewArgs["AahVersion"] = Version
	html.ViewArgs["EnvProfile"] = AppProfile()
	html.ViewArgs["AppBuildInfo"] = AppBuildInfo()
	html.ViewArgs[KeyViewArgSubject] = ctx.Subject()
	html.ViewArgs[KeyViewArgRequestParams] = ctx.Req.Params
}

func init() {
	AddTemplateFunc(template.FuncMap{
		"config":          tmplConfig,
		"rurl":            tmplURL,
		"rurlm":           tmplURLm,
		"i18n":            tmplI18n,
		"pparam":          tmplPathParam,
		"fparam":          tmplFormParam,
		"qparam":          tmplQueryParam,
		"session":         tmplSessionValue,
		"flash":           tmplFlashValue,
		"isauthenticated": tmplIsAuthenticated,
		"hasrole":         tmplHasRole,
		"hasallroles":     tmplHasAllRoles,
		"hasanyrole":      tmplHasAnyRole,
		"ispermitted":     tmplIsPermitted,
		"ispermittedall":  tmplIsPermittedAll,
		"anitcsrftoken":   tmplAntiCSRFToken,
	})
}
