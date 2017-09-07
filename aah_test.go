// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"errors"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0-unstable"
	"aahframework.org/test.v0/assert"
)

func TestAahInitAppVariables(t *testing.T) {
	cfgDir := filepath.Join(getTestdataPath(), appConfigDir())
	err := initConfig(cfgDir)
	assert.Nil(t, err)

	err = initAppVariables()
	assert.Nil(t, err)

	AppConfig().SetString("env.dev.test_value", "dev test value")
	err = initAppVariables()
	assert.Nil(t, err)

	assert.Equal(t, "aahframework", AppName())
	assert.Equal(t, "aah framework test config", AppDesc())
	assert.Equal(t, "127.0.0.1", AppHTTPAddress())
	assert.Equal(t, "80", AppHTTPPort())
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
	appIsSSLEnabled = true
	assert.Equal(t, "443", AppHTTPPort())

	// app packaged
	assert.False(t, appIsPackaged)

	// init auto cert
	AppConfig().SetBool("server.ssl.enable", true)
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

	SetAppPackaged(true)
	assert.True(t, appIsPackaged)

	// cleanup
	appConfig = nil
	SetAppPackaged(false)
}

func TestAahInitPath(t *testing.T) {
	err := initPath("github.com/jeevatkm/testapp")
	assert.NotNil(t, err)
	assert.Equal(t, "aah application does not exists: github.com/jeevatkm/testapp", err.Error())
	assert.True(t, !ess.IsStrEmpty(goSrcDir))
	assert.True(t, !ess.IsStrEmpty(goPath))

	// cleanup
	appImportPath, appBaseDir, goPath, goSrcDir = "", "", "", ""
	appIsPackaged = false
}

func TestAahRecover(t *testing.T) {
	defer aahRecover()

	cfgDir := filepath.Join(getTestdataPath(), appConfigDir())
	err := initConfig(cfgDir)
	assert.Nil(t, err)

	panic("this is recover test")
}

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

}

func TestWritePID(t *testing.T) {
	pidfile := filepath.Join(getTestdataPath(), "test-app")
	defer ess.DeleteFiles(pidfile + ".pid")

	cfgDir := filepath.Join(getTestdataPath(), appConfigDir())
	err := initConfig(cfgDir)
	assert.Nil(t, err)

	writePID(AppConfig(), "test-app", getTestdataPath())
	assert.True(t, ess.IsFileExists(pidfile+".pid"))
}

func TestAahBuildInfo(t *testing.T) {
	assert.Nil(t, AppBuildInfo())

	buildTime := time.Now().Format(time.RFC3339)
	SetAppBuildInfo(&BuildInfo{
		BinaryName: "testapp",
		Date:       buildTime,
		Version:    "1.0.0",
	})

	assert.NotNil(t, AppBuildInfo())
	assert.Equal(t, "testapp", AppBuildInfo().BinaryName)
	assert.Equal(t, buildTime, AppBuildInfo().Date)
	assert.Equal(t, "1.0.0", AppBuildInfo().Version)
}

func TestAahConfigValidation(t *testing.T) {
	err := checkSSLConfigValues(true, false, "/path/to/cert.pem", "/path/to/cert.key")
	assert.Equal(t, "SSL cert file not found: /path/to/cert.pem", err.Error())

	certPath := filepath.Join(getTestdataPath(), "cert.pem")
	defer ess.DeleteFiles(certPath)
	_ = ioutil.WriteFile(certPath, []byte("cert.pem file"), 0755)
	err = checkSSLConfigValues(true, false, certPath, "/path/to/cert.key")
	assert.Equal(t, "SSL key file not found: /path/to/cert.key", err.Error())
}

func TestAahAppInit(t *testing.T) {
	Init("aahframework.org/aah.v0-unstable/testdata")
	assert.NotNil(t, appConfig)
	assert.NotNil(t, appRouter)
	assert.NotNil(t, appSecurityManager)

	// reset it
	appConfig = nil
	appBaseDir = ""
}
