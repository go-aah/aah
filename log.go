// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"path/filepath"

	"aahframework.org/essentials"
	"aahframework.org/log"
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// app methods
//______________________________________________________________________________

func (a *app) Log() log.Loggerer {
	return a.logger
}

// AddLoggerHook method adds given logger into aah application default logger.
func (a *app) AddLoggerHook(name string, hook log.HookFunc) error {
	return a.Log().(*log.Logger).AddHook(name, hook)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// app Unexported methods
//______________________________________________________________________________

func (a *app) initLog() error {
	if !a.Config().IsExists("log") {
		log.Warn("Section 'log { ... }' configuration does not exists, initializing app logger with default values.")
	}

	if a.Config().StringDefault("log.receiver", "") == "file" {
		file := a.Config().StringDefault("log.file", "")
		if ess.IsStrEmpty(file) {
			a.Config().SetString("log.file", filepath.Join(a.logsDir(), a.binaryFilename()+".log"))
		} else if !filepath.IsAbs(file) {
			a.Config().SetString("log.file", filepath.Join(a.logsDir(), file))
		}
	}

	if !a.Config().IsExists("log.pattern") {
		a.Config().SetString("log.pattern", "%time:2006-01-02 15:04:05.000 %level:-5 %appname %insname %reqid %principal %message %fields")
	}

	al, err := log.New(a.Config())
	if err != nil {
		return err
	}

	al.AddContext(log.Fields{
		"appname": a.Name(),
		"insname": a.InstanceName(),
	})

	a.logger = al
	log.SetDefaultLogger(al)
	return nil
}
