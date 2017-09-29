// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
	"aahframework.org/test.v0/assert"
)

func TestAahLogDir(t *testing.T) {
	logsDir := filepath.Join(getTestdataPath(), appLogsDir())
	logFile := filepath.Join(logsDir, "test.log")
	defer ess.DeleteFiles(logsDir)

	cfgDir := filepath.Join(getTestdataPath(), appConfigDir())
	err := initConfig(cfgDir)
	assert.Nil(t, err)

	err = initLogs(logsDir, AppConfig())
	assert.Nil(t, err)

	AppConfig().SetString("log.receiver", "file")
	AppConfig().SetString("log.file", logFile)
	err = initLogs(logsDir, AppConfig())
	assert.Nil(t, err)
	assert.True(t, ess.IsFileExists(logFile))

	cfg, _ := config.ParseString("")
	logger, _ := log.New(cfg)
	log.SetDefaultLogger(logger)
	err = AddLoggerHook("defaulthook", func(e log.Entry) {
		logger.Info(e)
	})
	assert.Nil(t, err)

	// relative path filename
	cfgRelativeFile, _ := config.ParseString(`
		log {
			receiver = "file"
			file = "my-test-file.log"
		}
		`)
	err = initLogs(logsDir, cfgRelativeFile)
	assert.Nil(t, err)

	// no filename mentioned
	cfgNoFile, _ := config.ParseString(`
	log {
		receiver = "file"
	}
	`)
	SetAppBuildInfo(&BuildInfo{
		BinaryName: "testapp",
		Date:       time.Now().Format(time.RFC3339),
		Version:    "1.0.0",
	})
	err = initLogs(logsDir, cfgNoFile)
	assert.Nil(t, err)
	appBuildInfo = nil

	appLogFatal = func(v ...interface{}) { t.Log(v) }
	logAsFatal(errors.New("test msg"))

	logger2 := NewChildLogger(log.Fields{
		"myname": "I'm child logger",
	})
	logger2.Info("Hi child logger")
}
