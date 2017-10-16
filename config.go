// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"aahframework.org/config.v0"
	ess "aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
)

var (
	appConfig   *config.Config
	isHotReload = false
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package methods
//___________________________________

// AppConfig method returns aah application configuration instance.
func AppConfig() *config.Config {
	return appConfig
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

func appConfigDir() string {
	return filepath.Join(AppBaseDir(), "config")
}

func initConfig(cfgDir string) error {
	confPath := filepath.Join(cfgDir, "aah.conf")

	cfg, err := config.LoadFile(confPath)
	if err != nil {
		return fmt.Errorf("aah application %s", err)
	}

	appConfig = cfg

	return nil
}

func hotReloadConfig() {
	isHotReload = true
	defer func() { isHotReload = false }()

	log.Info("Configuration reload and application reinitialization is in-progress ...")
	var err error

	cfgDir := appConfigDir()
	if err = initConfig(cfgDir); err != nil {
		log.Errorf("Unable to reload aah.conf: %v", err)
		return
	}

	if err = initAppVariables(); err != nil {
		log.Errorf("Unable to reinitialize aah application variables: %v", err)
		return
	}

	logDir := appLogsDir()
	if err = initLogs(logDir, AppConfig()); err != nil {
		log.Errorf("Unable to reinitialize application logger: %v", err)
		return
	}

	i18nDir := appI18nDir()
	if ess.IsFileExists(i18nDir) {
		if err = initI18n(i18nDir); err != nil {
			log.Errorf("Unable to reinitialize application i18n: %v", err)
			return
		}
	}

	if err = initRoutes(cfgDir, AppConfig()); err != nil {
		log.Errorf("Unable to reinitialize application %v", err)
		return
	}

	viewDir := appViewsDir()
	if ess.IsFileExists(viewDir) {
		if err = initViewEngine(viewDir, AppConfig()); err != nil {
			log.Errorf("Unable to reinitialize application views: %v", err)
			return
		}
	}

	if err = initSecurity(AppConfig()); err != nil {
		log.Errorf("Unable to reinitialize application security manager: %v", err)
		return
	}

	if AppConfig().BoolDefault("server.access_log.enable", false) {
		if err = initAccessLog(logDir, AppConfig()); err != nil {
			log.Errorf("Unable to reinitialize application access log: %v", err)
			return
		}
	}

	if AppConfig().BoolDefault("server.dump_log.enable", false) {
		if err = initDumpLog(logDir, AppConfig()); err != nil {
			log.Errorf("Unable to reinitialize application dump log: %v", err)
			return
		}
	}

	log.Info("Configuration reload and application reinitialization is successful")
}

func listenForHotConfigReload() {
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGHUP)
	for {
		<-sc
		log.Warn("Hangup signal (SIGHUP) received")
		if appProfile == appDefaultProfile {
			log.Info("Currently active environment profile is 'dev', config hot reload is not applicable")
			continue
		}
		hotReloadConfig()
	}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Template methods
//___________________________________

// tmplConfig method provides access to application config on templates.
func tmplConfig(key string) interface{} {
	if value, found := AppConfig().Get(key); found {
		return sanatizeValue(value)
	}
	log.Warnf("app config key not found: %v", key)
	return ""
}
