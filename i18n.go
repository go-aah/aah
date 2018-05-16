// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"path"

	"aahframework.org/ahttp.v0"
	"aahframework.org/i18n.v0"
)

const keyLocale = "Locale"

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// app methods
//______________________________________________________________________________

func (a *app) I18n() *i18n.I18n {
	return a.i18n
}

// DefaultI18nLang method returns application i18n default language if
// configured otherwise framework defaults to "en".
func (a *app) DefaultI18nLang() string {
	return a.Config().StringDefault("i18n.default", "en")
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// app Unexported methods
//______________________________________________________________________________

func (a *app) initI18n() error {
	i18nPath := path.Join(a.VirtualBaseDir(), "i18n")
	if !a.VFS().IsExists(i18nPath) {
		// i18n directory not exists, scenario could be only API application
		return nil
	}

	ai18n := i18n.NewWithVFS(a.VFS())
	ai18n.DefaultLocale = a.DefaultI18nLang()
	if err := ai18n.Load(i18nPath); err != nil {
		return err
	}

	a.i18n = ai18n
	return nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// View Template methods
//______________________________________________________________________________

// tmplI18n method is mapped to Go template func for resolving i18n values.
func (vm *viewManager) tmplI18n(viewArgs map[string]interface{}, key string, args ...interface{}) string {
	if locale, ok := viewArgs[keyLocale].(*ahttp.Locale); ok {
		if len(args) == 0 {
			return vm.a.I18n().Lookup(locale, key)
		}

		sanatizeArgs := make([]interface{}, 0)
		for _, value := range args {
			sanatizeArgs = append(sanatizeArgs, sanatizeValue(value))
		}
		return vm.a.I18n().Lookup(locale, key, sanatizeArgs...)
	}
	return ""
}
