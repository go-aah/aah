// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"aahframework.org/config.v0"
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// app methods
//______________________________________________________________________________

func (a *app) Config() *config.Config {
	return a.cfg
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// app Unexported methods
//______________________________________________________________________________

func (a *app) initConfig() error {
	aahConf := filepath.Join(a.configDir(), "aah.conf")
	cfg, err := config.LoadFile(aahConf)
	if err != nil {
		return err
	}

	a.cfg = cfg
	a.sc = make(chan os.Signal, 2)
	return nil
}

func (a *app) listenForHotConfigReload() {
	signal.Notify(a.sc, syscall.SIGHUP)
	for {
		<-a.sc
		a.Log().Warn("Hangup signal (SIGHUP) received")
		if a.IsProfile(defaultEnvProfile) {
			a.Log().Info("Currently active environment profile is 'dev', config hot-reload is not applicable")
			continue
		}
		a.hotReloadConfig()
	}
}

func (a *app) hotReloadConfig() {
	a.hotReload = true
	defer func() { a.hotReload = false }()

	a.Log().Info("Configuration reload and application reinitialization is in-progress ...")
	var err error

	if err = a.initConfig(); err != nil {
		a.Log().Errorf("Unable to reload aah.conf: %v", err)
		return
	}

	if err = a.initConfigValues(); err != nil {
		a.Log().Errorf("Unable to reinitialize aah application variables: %v", err)
		return
	}

	if err = a.initLog(); err != nil {
		a.Log().Errorf("Unable to reinitialize application logger: %v", err)
		return
	}

	if err = a.initI18n(); err != nil {
		a.Log().Errorf("Unable to reinitialize application i18n: %v", err)
		return
	}

	if err = a.initRouter(); err != nil {
		a.Log().Errorf("Unable to reinitialize application %v", err)
		return
	}

	if err = a.initView(); err != nil {
		a.Log().Errorf("Unable to reinitialize application views: %v", err)
		return
	}

	if err = a.initSecurity(); err != nil {
		a.Log().Errorf("Unable to reinitialize application security manager: %v", err)
		return
	}

	if a.accessLogEnabled {
		if err = a.initAccessLog(); err != nil {
			a.Log().Errorf("Unable to reinitialize application access log: %v", err)
			return
		}
	}

	if a.dumpLogEnabled {
		if err = a.initDumpLog(); err != nil {
			a.Log().Errorf("Unable to reinitialize application dump log: %v", err)
			return
		}
	}

	a.Log().Info("Configuration reload and application reinitialization is successfully")
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// View Template methods
//______________________________________________________________________________

// tmplConfig method provides access to application config on templates.
func (vm *viewManager) tmplConfig(key string) interface{} {
	if value, found := vm.a.Config().Get(key); found {
		return sanatizeValue(value)
	}
	vm.a.Log().Warnf("app config key not found: %s", key)
	return ""
}
