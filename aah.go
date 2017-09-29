// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

// Package aah is A scalable, performant, rapid development Web framework for Go
// https://aahframework.org
package aah

import (
	"bytes"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"time"

	"aahframework.org/aruntime.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
)

// aah application variables
var (
	appName               string
	appInstanceName       string
	appDesc               string
	appImportPath         string
	appProfile            string
	appBaseDir            string
	appIsPackaged         bool
	appHTTPReadTimeout    time.Duration
	appHTTPWriteTimeout   time.Duration
	appHTTPMaxHdrBytes    int
	appSSLCert            string
	appSSLKey             string
	appIsSSLEnabled       bool
	appIsLetsEncrypt      bool
	appIsProfileProd      bool
	appMultipartMaxMemory int64
	appMaxBodyBytesSize   int64
	appPID                int
	appInitialized        bool
	appBuildInfo          *BuildInfo

	appDefaultProfile  = "dev"
	appProfilePrefix   = "env."
	appDefaultHTTPPort = "8080"
	appLogFatal        = log.Fatal

	goPath   string
	goSrcDir string
)

// BuildInfo holds the aah application build information; such as BinaryName,
// Version and Date.
type BuildInfo struct {
	BinaryName string
	Version    string
	Date       string
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package methods
//___________________________________

// AppName method returns aah application name from app config `name` otherwise app name
// of the base directory.
func AppName() string {
	return appName
}

// AppInstanceName method returns aah application instane name from app config `instance_name`
// otherwise empty string.
func AppInstanceName() string {
	return appInstanceName
}

// AppDesc method returns aah application friendly description from app config
// otherwise empty string.
func AppDesc() string {
	return appDesc
}

// AppProfile returns aah application configuration profile name
// For e.g.: dev, prod, etc. Default is `dev`
func AppProfile() string {
	return appProfile
}

// AppBaseDir method returns the application base or binary current directory
// 	For e.g.:
// 		$GOPATH/src/github.com/user/myproject
// 		<app/binary/path/base/directory>
func AppBaseDir() string {
	return appBaseDir
}

// AppImportPath method returns the application Go import path.
func AppImportPath() string {
	return appImportPath
}

// AppHTTPAddress method returns aah application HTTP address otherwise empty string
func AppHTTPAddress() string {
	return AppConfig().StringDefault("server.address", "")
}

// AppHTTPPort method returns aah application HTTP port number based on `server.port`
// value. Possible outcomes are user-defined port, `80`, `443` and `8080`.
func AppHTTPPort() string {
	port := firstNonZeroString(AppConfig().StringDefault("server.proxyport", ""),
		AppConfig().StringDefault("server.port", appDefaultHTTPPort))
	return parsePort(port)
}

// AppBuildInfo method return user application version no.
func AppBuildInfo() *BuildInfo {
	return appBuildInfo
}

// AllAppProfiles method returns all the aah application environment profile names.
func AllAppProfiles() []string {
	var profiles []string

	for _, v := range AppConfig().KeysByPath("env") {
		if v == "default" {
			continue
		}
		profiles = append(profiles, v)
	}

	return profiles
}

// AppIsSSLEnabled method returns true if aah application is enabled with SSL
// otherwise false.
func AppIsSSLEnabled() bool {
	return appIsSSLEnabled
}

// SetAppProfile method sets given profile as current aah application profile.
//		For Example:
//
//		aah.SetAppProfile("prod")
func SetAppProfile(profile string) error {
	if err := AppConfig().SetProfile(appProfilePrefix + profile); err != nil {
		return err
	}

	appProfile = profile
	appIsProfileProd = appProfile == "prod"
	return nil
}

// SetAppBuildInfo method sets the user application build info into aah instance.
func SetAppBuildInfo(bi *BuildInfo) {
	appBuildInfo = bi
}

// SetAppPackaged method sets the info of binary is packaged or not.
func SetAppPackaged(pack bool) {
	appIsPackaged = pack
}

// NewChildLogger method create a child logger from aah application default logger.
func NewChildLogger(ctx log.Fields) *log.Logger {
	return appLogger.New(ctx)
}

// Init method initializes `aah` application, if anything goes wrong during
// initialize process, it will log it as fatal msg and exit.
func Init(importPath string) {
	defer aahRecover()

	if appBuildInfo == nil {
		// aah CLI is accessing application for build purpose
		_ = log.SetLevel("warn")
		logAsFatal(initPath(importPath))
		logAsFatal(initConfig(appConfigDir()))
		logAsFatal(initAppVariables())
		logAsFatal(initRoutes(appConfigDir(), AppConfig()))
		_ = log.SetLevel("debug")
	} else {
		logAsFatal(initPath(importPath))
		logAsFatal(initConfig(appConfigDir()))

		// publish `OnInit` server event
		AppEventStore().sortAndPublishSync(&Event{Name: EventOnInit})

		logAsFatal(initAppVariables())
		logAsFatal(initLogs(appLogsDir(), AppConfig()))
		logAsFatal(initI18n(appI18nDir()))
		logAsFatal(initRoutes(appConfigDir(), AppConfig()))
		logAsFatal(initViewEngine(appViewsDir(), AppConfig()))
		logAsFatal(initSecurity(AppConfig()))
		if AppConfig().BoolDefault("server.access_log.enable", false) {
			logAsFatal(initAccessLog(appLogsDir(), AppConfig()))
		}
	}

	appInitialized = true
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

func aahRecover() {
	if r := recover(); r != nil {
		strace := aruntime.NewStacktrace(r, AppConfig())
		buf := &bytes.Buffer{}
		strace.Print(buf)

		log.Error("Recovered from panic:")
		log.Error(buf.String())
	}
}

func appLogsDir() string {
	return filepath.Join(AppBaseDir(), "logs")
}

func logAsFatal(err error) {
	if err != nil {
		appLogFatal(err)
	}
}

func initPath(importPath string) (err error) {
	appImportPath = path.Clean(importPath)
	if goPath, err = ess.GoPath(); err != nil && !appIsPackaged {
		return err
	}

	goSrcDir = filepath.Join(goPath, "src")
	appBaseDir = filepath.Join(goSrcDir, filepath.FromSlash(appImportPath))
	if appIsPackaged {
		appBaseDir = getWorkingDir()
	}

	if !ess.IsFileExists(appBaseDir) {
		return fmt.Errorf("aah application does not exists: %s", appImportPath)
	}

	return nil
}

func initAppVariables() error {
	var err error
	cfg := AppConfig()

	appName = cfg.StringDefault("name", filepath.Base(AppBaseDir()))
	appInstanceName = cfg.StringDefault("instance_name", "")
	appDesc = cfg.StringDefault("desc", "")

	appProfile = cfg.StringDefault("env.active", appDefaultProfile)
	if err = SetAppProfile(AppProfile()); err != nil {
		return err
	}

	readTimeout := cfg.StringDefault("server.timeout.read", "90s")
	writeTimeout := cfg.StringDefault("server.timeout.write", "90s")
	if !isValidTimeUnit(readTimeout, "s", "m") || !isValidTimeUnit(writeTimeout, "s", "m") {
		return errors.New("'server.timeout.{read|write}' value is not a valid time unit")
	}

	if appHTTPReadTimeout, err = time.ParseDuration(readTimeout); err != nil {
		return fmt.Errorf("'server.timeout.read': %s", err)
	}

	if appHTTPWriteTimeout, err = time.ParseDuration(writeTimeout); err != nil {
		return fmt.Errorf("'server.timeout.write': %s", err)
	}

	maxHdrBytesStr := cfg.StringDefault("server.max_header_bytes", "1mb")
	if maxHdrBytes, er := ess.StrToBytes(maxHdrBytesStr); er == nil {
		appHTTPMaxHdrBytes = int(maxHdrBytes)
	} else {
		return errors.New("'server.max_header_bytes' value is not a valid size unit")
	}

	appIsSSLEnabled = cfg.BoolDefault("server.ssl.enable", false)
	appIsLetsEncrypt = cfg.BoolDefault("server.ssl.lets_encrypt.enable", false)
	appSSLCert = cfg.StringDefault("server.ssl.cert", "")
	appSSLKey = cfg.StringDefault("server.ssl.key", "")
	if err = checkSSLConfigValues(AppIsSSLEnabled(), appIsLetsEncrypt, appSSLCert, appSSLKey); err != nil {
		return err
	}

	if err = initAutoCertManager(cfg); err != nil {
		return err
	}

	maxBodySizeStr := cfg.StringDefault("request.max_body_size", "5mb")
	if appMaxBodyBytesSize, err = ess.StrToBytes(maxBodySizeStr); err != nil {
		return errors.New("'request.max_body_size' value is not a valid size unit")
	}

	multipartMemoryStr := cfg.StringDefault("request.multipart_size", "32mb")
	if appMultipartMaxMemory, err = ess.StrToBytes(multipartMemoryStr); err != nil {
		return errors.New("'request.multipart_size' value is not a valid size unit")
	}

	return nil
}
