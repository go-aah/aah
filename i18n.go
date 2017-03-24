// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"html/template"
	"path/filepath"

	"aahframework.org/ahttp.v0"
	"aahframework.org/i18n.v0"
)

var appI18n *i18n.I18n

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Global methods
//___________________________________

// AppDefaultI18nLang method returns aah application i18n default language if
// configured other framework defaults to "en".
func AppDefaultI18nLang() string {
	return AppConfig().StringDefault("i18n.default", "en")
}

// AppI18n method returns aah application I18n store instance.
func AppI18n() *i18n.I18n {
	return appI18n
}

// AppI18nLocales returns all the loaded locales from i18n store
func AppI18nLocales() []string {
	return appI18n.Locales()
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

func appI18nDir() string {
	return filepath.Join(AppBaseDir(), "i18n")
}

func initI18n(cfgDir string) error {
	appI18n = i18n.New()
	return appI18n.Load(cfgDir)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Template methods
//___________________________________

// tmplI18n method is mapped to Go template func for resolving i18n values.
func tmplI18n(viewArgs map[string]interface{}, key string, args ...interface{}) template.HTML {
	if locale, ok := viewArgs["Locale"].(*ahttp.Locale); ok {
		return template.HTML(template.HTMLEscapeString(AppI18n().Lookup(locale, key, args...)))
	}
	return template.HTML("")
}
