// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"path/filepath"

	"aahframework.org/config.v0"
	"aahframework.org/security.v0"
	"aahframework.org/security.v0/session"
)

const keySessionValues = "SessionValues"

var appSecurity *security.Security

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Global methods
//___________________________________

// AppSecurity method returns the application security instance,
// which manages the Session, CORS, CSRF, Security Headers, etc.
func AppSecurity() *security.Security {
	return appSecurity
}

// AppSessionManager method returns the application session manager.
// By default session is stateless.
func AppSessionManager() *session.Manager {
	return AppSecurity().SessionManager
}

// AddSessionStore method allows you to add custom session store which
// implements `session.Storer` interface. The `name` parameter is used in
// aah.conf on `session.store.type = "name"`.
func AddSessionStore(name string, store session.Storer) error {
	return session.AddStore(name, store)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

func initSecurity(cfgDir string, appCfg *config.Config) error {
	var err error
	configPath := filepath.Join(cfgDir, "security.conf")
	if appSecurity, err = security.New(configPath, appCfg); err != nil {
		return err
	}

	// Based on aah server SSL configuration `http.Cookie.Secure` value is set, even
	// though it's true in aah.conf at `security.session.secure = true`.
	if AppSessionManager() != nil {
		AppSessionManager().Options.Secure = AppIsSSLEnabled()
	}

	return nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Template methods
//___________________________________

// tmplSessionValue method returns session value for the given key. If session
// object unavailable this method returns nil.
func tmplSessionValue(viewArgs map[string]interface{}, key string) interface{} {
	if sv, found := viewArgs[keySessionValues]; found {
		value := sv.(*session.Session).Get(key)
		return sanatizeValue(value)
	}
	return nil
}

// tmplFlashValue method returns session value for the given key. If session
// object unavailable this method returns nil.
func tmplFlashValue(viewArgs map[string]interface{}, key string) interface{} {
	if sv, found := viewArgs[keySessionValues]; found {
		value := sv.(*session.Session).GetFlash(key)
		return sanatizeValue(value)
	}
	return nil
}
