// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"path/filepath"
	"strings"
	"testing"

	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
	"aahframework.org/test.v0/assert"
)

func TestAahInitAppVariables(t *testing.T) {
	cfgDir := filepath.Join(getTestdataPath(), appConfigDir())
	err := initConfig(cfgDir)
	assert.Nil(t, err)

	err = initAppVariables()
	assert.NotNil(t, err)
	assert.Equal(t, "profile doesn't exists: env.dev", err.Error())

	AppConfig().SetString("env.dev.test_value", "dev test value")
	err = initAppVariables()
	assert.Nil(t, err)

	assert.Equal(t, "aahframework", AppName())
	assert.Equal(t, "127.0.0.1", AppHTTPAddress())
	assert.Equal(t, "80", AppHTTPPort())
	assert.Equal(t, "2006-01-02", AppDateFormat())
	assert.Equal(t, "2006-01-02 15:04:05", AppDateTimeFormat())
	assert.Equal(t, "en", AppDefaultI18nLang())
	assert.True(t, ess.IsStrEmpty(AppImportPath()))
	assert.False(t, AppIsSSLEnabled())
	assert.Equal(t, "dev", AppProfile())
	assert.Equal(t, "1m30s", appHTTPReadTimeout.String())
	assert.Equal(t, "1m30s", appHTTPWriteTimeout.String())
	assert.Equal(t, 1048576, appHTTPMaxHdrBytes)
	assert.False(t, appInitialized)
	assert.Equal(t, int64(33554432), appMultipartMaxMemory)
	assert.True(t, ess.IsStrEmpty(appSSLCert))
	assert.True(t, ess.IsStrEmpty(appSSLKey))

	AppConfig().SetString("env.default", "dev")
	profiles := AllAppProfiles()
	assert.NotNil(t, profiles)
	assert.True(t, len(profiles) == 1)
	assert.Equal(t, "dev", profiles[0])

	// App port no
	AppConfig().SetString("server.port", "")
	assert.Equal(t, "80", AppHTTPPort())
	AppConfig().SetBool("server.ssl.enable", true)
	assert.Equal(t, "443", AppHTTPPort())

	// app packaged
	assert.False(t, isBinDirExists())
	assert.False(t, isAppDirExists())

	// init auto cert
	AppConfig().SetBool("server.ssl.lets_encrypt.enable", true)
	defer ess.DeleteFiles(filepath.Join(getTestdataPath(), "autocert"))
	AppConfig().SetString("server.ssl.lets_encrypt.cache_dir", filepath.Join(getTestdataPath(), "autocert"))
	err = initAppVariables()
	assert.Nil(t, err)
	assert.NotNil(t, appAutocertManager)

	AppConfig().SetBool("server.ssl.enable", false)
	err = initAppVariables()
	assert.Equal(t, "let's encrypt enabled, however SSL 'server.ssl.enable' is not enabled for application", err.Error())

	// revert values
	AppConfig().SetString("server.port", appDefaultHTTPPort)
	AppConfig().SetBool("server.ssl.lets_encrypt.enable", false)

	// config error scenario

	// unsupported read timeout
	AppConfig().SetString("server.timeout.read", "20h")
	err = initAppVariables()
	assert.NotNil(t, err)
	assert.Equal(t, "'server.timeout.{read|write}' value is not a valid time unit", err.Error())
	AppConfig().SetString("server.timeout.read", "90s")

	// read timeout parsing error
	AppConfig().SetString("server.timeout.read", "ss")
	err = initAppVariables()
	assert.Equal(t, "'server.timeout.read': time: invalid duration ss", err.Error())
	AppConfig().SetString("server.timeout.read", "90s")

	// write timout pasring error
	AppConfig().SetString("server.timeout.write", "mm")
	err = initAppVariables()
	assert.Equal(t, "'server.timeout.write': time: invalid duration mm", err.Error())
	AppConfig().SetString("server.timeout.write", "90s")

	// max header bytes parsing error
	AppConfig().SetString("server.max_header_bytes", "2sb")
	err = initAppVariables()
	assert.Equal(t, "'server.max_header_bytes' value is not a valid size unit", err.Error())
	AppConfig().SetString("server.max_header_bytes", "1mb")

	// ssl cert required if enabled
	AppConfig().SetBool("server.ssl.enable", true)
	err = initAppVariables()
	assert.True(t, (!appIsLetsEncrypt && strings.Contains(err.Error(), "server.ssl.cert")))
	assert.True(t, (!appIsLetsEncrypt && strings.Contains(err.Error(), "server.ssl.key")))
	AppConfig().SetBool("server.ssl.enable", false)

	// multipart size parsing error
	AppConfig().SetString("request.multipart_size", "2sb")
	err = initAppVariables()
	assert.Equal(t, "'request.multipart_size' value is not a valid size unit", err.Error())
	AppConfig().SetString("request.multipart_size", "12mb")
}

func TestAahRecover(t *testing.T) {
	defer aahRecover()

	cfgDir := filepath.Join(getTestdataPath(), appConfigDir())
	err := initConfig(cfgDir)
	assert.Nil(t, err)

	panic("this is recover test")
}

func TestAahLogDir(t *testing.T) {
	logsDir := filepath.Join(getTestdataPath(), "logs")
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
	logger, _ := log.Newc(cfg)
	log.SetOutput(logger)
}

func TestWritePID(t *testing.T) {
	pidfile := filepath.Join(getTestdataPath(), "test-app.pid")
	defer ess.DeleteFiles(pidfile)

	cfgDir := filepath.Join(getTestdataPath(), appConfigDir())
	err := initConfig(cfgDir)
	assert.Nil(t, err)

	writePID("test-app", getTestdataPath(), AppConfig())
	assert.True(t, ess.IsFileExists(pidfile))
}
