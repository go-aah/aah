// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"fmt"
	"html/template"
	"path/filepath"

	"aahframework.org/config.v0-unstable"
	"aahframework.org/log.v0-unstable"
)

var appConfig *config.Config

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Global methods
//___________________________________

// AppConfig method returns aah application configuration instance.
func AppConfig() *config.Config {
	return appConfig
}

// MergeAppConfig method allows to you to merge external config into aah
// application anytime.
func MergeAppConfig(cfg *config.Config) {
	defer aahRecover()

	if err := AppConfig().Merge(cfg); err != nil {
		log.Errorf("Unable to merge config into aah application[%s]: %s", AppName(), err)
	}

	initInternal()
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

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Template methods
//___________________________________

// tmplConfig method provides access to application config on templates.
func tmplConfig(key string) template.HTML {
	if value, found := AppConfig().Get(key); found {
		return template.HTML(template.HTMLEscapeString(fmt.Sprintf("%v", value)))
	}
	log.Errorf("app config key not found: %v", key)
	return template.HTML("")
}
