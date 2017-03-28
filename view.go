// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"html/template"
	"path/filepath"
	"strings"

	"aahframework.org/ahttp.v0"
	"aahframework.org/atemplate.v0"
	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
)

var (
	appTemplateEngine        atemplate.TemplateEnginer
	appTemplateExt           string
	appDefaultTmplLayout     string
	appTemplateCaseSensitive bool
	isExternalTmplEngine     bool
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Global methods
//___________________________________

// AppTemplateEngine method returns aah application Template Engine instance.
func AppTemplateEngine() atemplate.TemplateEnginer {
	return appTemplateEngine
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

func appViewsDir() string {
	return filepath.Join(AppBaseDir(), "views")
}

func initTemplateEngine(viewDir string, appCfg *config.Config) error {
	// application config values
	appTemplateExt = appCfg.StringDefault("view.ext", ".html")
	appDefaultTmplLayout = "master" + appTemplateExt
	appTemplateCaseSensitive = appCfg.BoolDefault("view.case_sensitive", false)

	// initialize if external TemplateEngine is not registered.
	if appTemplateEngine == nil {
		tmplEngineName := appCfg.StringDefault("view.engine", "go")
		switch tmplEngineName {
		case "go":
			appTemplateEngine = &atemplate.TemplateEngine{}
		}

		isExternalTmplEngine = false
	} else {
		isExternalTmplEngine = true
	}

	appTemplateEngine.Init(appCfg, viewDir)

	return appTemplateEngine.Load()
}

// handlePreReplyStage method does 1) sets response header, 2) if HTML content type
// finds appropriate template based request and route information,
// 3) Adds request info into view args.
func handlePreReplyStage(ctx *Context) {
	reply := ctx.Reply()

	// ContentType
	if ess.IsStrEmpty(reply.ContType) {
		if !ess.IsStrEmpty(ctx.Req.AcceptContentType.Mime) &&
			ctx.Req.AcceptContentType.Mime != "*/*" { // based on 'Accept' Header
			reply.ContentType(ctx.Req.AcceptContentType.Raw())
		} else if ct := defaultContentType(); ct != nil { // as per 'render.default' in aah.conf
			reply.ContentType(ct.Raw())
		}
	}

	// HTML response
	if AppMode() == appModeWeb && ahttp.ContentTypeHTML.IsEqual(reply.ContType) {
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
			htmlRdr.ViewArgs[k] = v
		}

		// ViewArgs values from framework
		htmlRdr.ViewArgs["Host"] = ctx.Req.Host
		htmlRdr.ViewArgs["HTTPMethod"] = ctx.Req.Method
		htmlRdr.ViewArgs["RequestPath"] = ctx.Req.Path
		htmlRdr.ViewArgs["Locale"] = ctx.Req.Locale
		htmlRdr.ViewArgs["ClientIP"] = ctx.Req.ClientIP
		htmlRdr.ViewArgs["IsJSONP"] = ctx.Req.IsJSONP
		htmlRdr.ViewArgs["HTTPReferer"] = ctx.Req.Referer
		htmlRdr.ViewArgs["AahVersion"] = Version

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
	tmplName := ctx.action.Name + appTemplateExt
	htmlRdr := ctx.Reply().Rdr.(*HTML)
	if !ess.IsStrEmpty(htmlRdr.Filename) {
		tmplName = htmlRdr.Filename
	}

	log.Tracef("Layout: %s, Template Path: %s, Template Name: %s", htmlRdr.Layout, tmplPath, tmplName)
	htmlRdr.Template = appTemplateEngine.Get(htmlRdr.Layout, tmplPath, tmplName)
	if htmlRdr.Template == nil {
		tmplFile := filepath.Join("views", "pages", controllerName, tmplName)
		if !appTemplateCaseSensitive {
			tmplFile = strings.ToLower(tmplFile)
		}

		log.Errorf("template not found: %s", tmplFile)
	}
}

func init() {
	atemplate.AddTemplateFunc(template.FuncMap{
		"config": tmplConfig,
		"rurl":   tmplURL,
		"rurlm":  tmplURLm,
		"i18n":   tmplI18n,
		"pparam": tmplPathParam,
		"fparam": tmplFormParam,
		"qparam": tmplQueryParam,
	})
}
