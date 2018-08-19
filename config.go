// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"fmt"
	"os"
	"os/signal"
	"path"
	"syscall"

	"aahframework.org/config"
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
	cfg, err := config.VFSLoadFile(a.VFS(), path.Join(a.VirtualBaseDir(), "config", "aah.conf"))
	if err != nil {
		return fmt.Errorf("aah.conf: %s", err)
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
		if a.IsProfile(defaultEnvProfile) && !a.IsPackaged() {
			a.Log().Info("Application's active environment profile is 'dev' and it seems you're running it via 'aah run'. " +
				"So hot-reload is not applicable")
			continue
		}
		a.hotReloadConfig()
	}
}

func (a *app) hotReloadConfig() {
	a.hotReload = true
	defer func() { a.hotReload = false }()

	activeProfile := a.Profile()

	a.Log().Info("Application hot-reload and reinitialization starts ...")
	var err error

	if err = a.initConfig(); err != nil {
		a.Log().Errorf("Unable to reload aah.conf: %v", err)
		return
	}
	a.Log().Info("Configuration files reload succeeded")

	// Set activeProfile into reloaded configuration
	a.Config().SetString("env.active", activeProfile)

	if err = a.initConfigValues(); err != nil {
		a.Log().Errorf("Unable to reinitialize aah application variables: %v", err)
		return
	}
	a.Log().Info("Configuration values reinitialize succeeded")

	if err = a.initLog(); err != nil {
		a.Log().Errorf("Unable to reinitialize application logger: %v", err)
		return
	}
	a.Log().Info("Logging reinitialize succeeded")

	if err = a.initI18n(); err != nil {
		a.Log().Errorf("Unable to reinitialize application i18n: %v", err)
		return
	}
	if a.Type() == "web" {
		a.Log().Info("I18n reinitialize succeeded")
	}

	if err = a.initRouter(); err != nil {
		a.Log().Errorf("Unable to reinitialize application %v", err)
		return
	}
	a.Log().Info("Router reinitialize succeeded")

	if err = a.initView(); err != nil {
		a.Log().Errorf("Unable to reinitialize application views: %v", err)
		return
	}
	if a.Type() == "web" {
		a.Log().Info("View engine reinitialize succeeded")
	}

	if err = a.initSecurity(); err != nil {
		a.Log().Errorf("Unable to reinitialize application security manager: %v", err)
		return
	}
	a.Log().Info("Security reinitialize succeeded")

	if a.accessLogEnabled {
		if err = a.initAccessLog(); err != nil {
			a.Log().Errorf("Unable to reinitialize application access log: %v", err)
			return
		}
		a.Log().Info("Access logging reinitialize succeeded")
	}

	if a.dumpLogEnabled {
		if err = a.initDumpLog(); err != nil {
			a.Log().Errorf("Unable to reinitialize application dump log: %v", err)
			return
		}
		a.Log().Info("Server dump logging reinitialize succeeded")
	}

	a.Log().Info("Application hot-reload and reinitialization was successful")
}
