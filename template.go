// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"html/template"
	"path/filepath"
	"strings"

	"aahframework.org/ahttp.v0-unstable"
	"aahframework.org/atemplate.v0-unstable"
	"aahframework.org/essentials.v0-unstable"
	"aahframework.org/log.v0-unstable"
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

func initTemplateEngine() error {
	// application config values
	appTemplateExt = AppConfig().StringDefault("template.ext", ".html")
	appDefaultTmplLayout = "master" + appTemplateExt
	appTemplateCaseSensitive = AppConfig().BoolDefault("template.case_sensitive", false)

	// initialize if external TemplateEngine is not registered.
	if appTemplateEngine == nil {
		tmplEngineName := AppConfig().StringDefault("template.engine", "go")
		switch tmplEngineName {
		case "go":
			appTemplateEngine = &atemplate.TemplateEngine{}
		}

		isExternalTmplEngine = false
	} else {
		isExternalTmplEngine = true
	}

	appTemplateEngine.Init(AppConfig(), appViewsDir())

	return appTemplateEngine.Load()
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Template middleware
//___________________________________

// TemplateMiddleware finds appropriate template based request and route
// information.
func templateMiddleware(c *Controller, m *Middleware) {
	m.Next(c)

	reply := c.Reply()

	// ContentType
	if ess.IsStrEmpty(reply.ContType) {
		if !ess.IsStrEmpty(c.Req.AcceptContentType.Mime) &&
			c.Req.AcceptContentType.Mime != "*/*" { // based on 'Accept' Header
			reply.ContentType(c.Req.AcceptContentType.Raw())
		} else { // default Content-Type defined 'render.default' in aah.conf
			reply.ContentType(defaultContentType().Raw())
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

		for k, v := range c.ViewArgs() {
			htmlRdr.ViewArgs[k] = v
		}

		// ViewArgs values from framework
		htmlRdr.ViewArgs["Host"] = c.Req.Host
		htmlRdr.ViewArgs["HTTPMethod"] = c.Req.Method
		htmlRdr.ViewArgs["Locale"] = c.Req.Locale
		htmlRdr.ViewArgs["ClientIP"] = c.Req.ClientIP
		htmlRdr.ViewArgs["RequestPath"] = c.Req.Path
		htmlRdr.ViewArgs["IsJSONP"] = c.Req.IsJSONP
		htmlRdr.ViewArgs["HTTPReferer"] = c.Req.Referer
		htmlRdr.ViewArgs["AahVersion"] = Version

		controllerName := c.controller
		if strings.HasSuffix(controllerName, controllerNameSuffix) {
			controllerName = controllerName[:len(controllerName)-controllerNameSuffixLen]
		}

		tmplPath := filepath.Join("pages", controllerName)
		tmplName := c.action.Name + appTemplateExt

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
}

func init() {
	atemplate.AddTemplateFunc(template.FuncMap{
		"config": tmplConfig,
		"url":    tmplURL,
		"urlm":   tmplURLm,
		"i18n":   tmplI18n,
	})
}
