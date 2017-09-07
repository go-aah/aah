// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"fmt"
	"path/filepath"

	"aahframework.org/config.v0"
	"aahframework.org/log.v0-unstable"
)

var appConfig *config.Config

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
