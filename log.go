// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"path/filepath"

	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
)

var appLogger *log.Logger

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package methods
//___________________________________

// AddLoggerHook method adds given logger into aah application default logger.
func AddLoggerHook(name string, hook log.HookFunc) error {
	return appLogger.AddHook(name, hook)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

func initLogs(logsDir string, appCfg *config.Config) error {
	if !appCfg.IsExists("log") {
		log.Debug("Section 'log {...}' configuration not exists, move on.")
		return nil
	}

	if appCfg.StringDefault("log.receiver", "") == "file" {
		file := appCfg.StringDefault("log.file", "")
		if ess.IsStrEmpty(file) {
			appCfg.SetString("log.file", filepath.Join(logsDir, getBinaryFileName()+".log"))
		} else if !filepath.IsAbs(file) {
			appCfg.SetString("log.file", filepath.Join(logsDir, file))
		}
	}

	if !appCfg.IsExists("log.pattern") {
		appCfg.SetString("log.pattern", "%time:2006-01-02 15:04:05.000 %level:-5 %appname %insname %reqid %principal %message %fields")
	}

	var err error
	appLogger, err = log.New(appCfg)
	if err != nil {
		return err
	}

	appLogger.AddContext(log.Fields{
		"appname": AppName(),
		"insname": AppInstanceName(),
	})
	log.SetDefaultLogger(appLogger)
	return nil
}
