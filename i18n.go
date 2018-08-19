// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"path"

	"aahframework.org/i18n"
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
