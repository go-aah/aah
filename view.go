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

var (
	appViewEngine            view.Enginer
	appViewExt               string
	appDefaultTmplLayout     string
	appViewFileCaseSensitive bool
	isExternalTmplEngine     bool
	viewNotFoundTemplate     *template.Template
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Global methods
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

	// initialize if external View Engine is not registered.
	if appViewEngine == nil {
		isExternalTmplEngine = false
		viewEngineName := appCfg.StringDefault("view.engine", "go")
		viewEngine, found := view.GetEngine(viewEngineName)
		if !found {
			return fmt.Errorf("view: named engine not found: %s", viewEngineName)
		}

		appViewEngine = viewEngine
		return appViewEngine.Init(appCfg, viewDir)
	}

	isExternalTmplEngine = true
	return nil
}

// resolveView method does -
//   1) Prepare ViewArgs
//   2) If HTML content type find appropriate template
func (e *engine) resolveView(ctx *Context) {
	reply := ctx.Reply()

	// HTML response
	if ahttp.ContentTypeHTML.IsEqual(reply.ContType) && appViewEngine != nil {
		if reply.Rdr == nil {
			reply.Rdr = &HTML{}
		}

		htmlRdr := reply.Rdr.(*HTML)

		if ess.IsStrEmpty(htmlRdr.Layout) {
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
		htmlRdr.ViewArgs["Scheme"] = ctx.Req.Schema
		htmlRdr.ViewArgs["Host"] = ctx.Req.Host
		htmlRdr.ViewArgs["HTTPMethod"] = ctx.Req.Method
		htmlRdr.ViewArgs["RequestPath"] = ctx.Req.Path
		htmlRdr.ViewArgs["Locale"] = ctx.Req.Locale
		htmlRdr.ViewArgs["ClientIP"] = ctx.Req.ClientIP
		htmlRdr.ViewArgs["IsJSONP"] = ctx.Req.IsJSONP
		htmlRdr.ViewArgs["HTTPReferer"] = ctx.Req.Referer
		htmlRdr.ViewArgs["AahVersion"] = Version
		htmlRdr.ViewArgs["EnvProfile"] = AppProfile()
		htmlRdr.ViewArgs["AppBuildInfo"] = AppBuildInfo()

		// find view template by convention if not provided
		findViewTemplate(ctx)
	}
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
	controllerName := ctx.controller
	if strings.HasSuffix(controllerName, controllerNameSuffix) {
		controllerName = controllerName[:len(controllerName)-controllerNameSuffixLen]
	}

	tmplPath := filepath.Join("pages", controllerName)
	tmplName := ctx.action.Name + appViewExt
	htmlRdr := ctx.Reply().Rdr.(*HTML)
	if !ess.IsStrEmpty(htmlRdr.Filename) {
		tmplName = htmlRdr.Filename
	}

	log.Tracef("Layout: %s, Template Path: %s, Template Name: %s", htmlRdr.Layout, tmplPath, tmplName)
	var err error
	if htmlRdr.Template, err = appViewEngine.Get(htmlRdr.Layout, tmplPath, tmplName); err != nil {
		if err == view.ErrTemplateNotFound {
			tmplFile := filepath.Join("views", "pages", controllerName, tmplName)
			if !appViewFileCaseSensitive {
				tmplFile = strings.ToLower(tmplFile)
			}

			log.Errorf("template not found: %s", tmplFile)
			htmlRdr.ViewArgs["ViewNotFound"] = tmplFile
			htmlRdr.Layout = ""
			htmlRdr.Template = viewNotFoundTemplate
		} else {
			log.Error(err)
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

func addViewNotFoundTemplate(e *Event) {
	viewNotFoundTemplate = template.Must(template.New("not_found").Funcs(view.TemplateFuncMap).Parse(`
		<strong>View not found: {{ .ViewNotFound }}</strong>
	`))
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
		"isauthenticated": tmplIsAuthenticated,
		"flash":           tmplFlashValue,
	})

	OnStart(addViewNotFoundTemplate)
}
