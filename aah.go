// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"aahframework.org/aah/i18n"
	"aahframework.org/aah/router"
	"aahframework.org/config"
	"aahframework.org/essentials"
	"aahframework.org/log"
)

// aah application variables
var (
	appName       string
	appImportPath string
	appProfile    string
	appBaseDir    string
	appIsPackaged bool
	appConfig     *config.Config
	appRoutes     *router.Router

	goPath   string
	goSrcDir string

	appDefaultProfile = "dev"
	appProfilePrefix  = "env_"
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Global methods
//___________________________________

// AppName method returns aah application name from app config otherwise app name
// of the base directory.
func AppName() string {
	return appName
}

// AppProfile returns aah application configuration profile name
// For e.g.: dev, prod, etc. Default is `dev`
func AppProfile() string {
	return appProfile
}

// AppBaseDir method returns the application base or binary current directory
// 	For e.g.:
// 		$GOPATH/src/github.com/user/myproject
// 		<app/binary/path>
func AppBaseDir() string {
	return appBaseDir
}

// AppConfig method returns aah application configuration instance.
func AppConfig() *config.Config {
	return appConfig
}

// AppMode method returns aah application mode. Default is "web" For e.g.: web or api
func AppMode() string {
	return AppConfig().StringDefault("mode", "web")
}

// AppDefaultI18nLang method returns aah application i18n default language if
// configured other framework defaults to "en".
func AppDefaultI18nLang() string {
	return AppConfig().StringDefault("i18n.default", "en")
}

// AllAppProfiles method returns all the aah application environment profile names.
func AllAppProfiles() []string {
	var profiles []string

	for _, v := range AppConfig().Keys() {
		if strings.HasPrefix(v, appProfilePrefix) {
			profiles = append(profiles, v[4:])
		}
	}

	return profiles
}

// IsSSLEnabled method returns true if aah application is enabled with SSL
// otherwise false.
func IsSSLEnabled() bool {
	return AppConfig().BoolDefault("http.ssl.enable", false)
}

// IsBehindProxy method returns true if aah application configured behind proxy
// based on application configuration.
func IsBehindProxy() bool {
	return AppConfig().BoolDefault("http.proxy", false)
}

// IsCookieEnabled method returns true if aah application is enabled with Cookie
// otherwise false.
func IsCookieEnabled() bool {
	return AppConfig().BoolDefault("cookie.enable", false)
}

// SetAppProfile method sets given profile as current aah application profile.
//		For Example:
//
//		aah.SetAppProfile("prod")
func SetAppProfile(profile string) error {
	return AppConfig().SetProfile(appProfilePrefix + AppProfile())
}

// Init method initializes `aah` application, if anything goes wrong during
// initialize process, it will log it as fatal msg and exit.
func Init(importPath, profile string) {
	logAsFatal(initPath(importPath))

	if ess.IsStrEmpty(profile) {
		appProfile = appDefaultProfile
	} else {
		appProfile = profile
	}

	logAsFatal(initConfig(appConfigDir()))
	appName = AppConfig().StringDefault("name", filepath.Base(appBaseDir))

	logAsFatal(SetAppProfile(AppProfile()))

	logAsFatal(initLogs(appLogsDir(), AppConfig()))

	log.Info("----- aah framework -----")
	log.Infof("App Name: %v", AppName())
	log.Infof("App Profile: %v", AppProfile())
	log.Infof("App Mode: %v", AppMode())

	logAsFatal(initI18n(appI18nDir()))
	log.Infof("App i18n Locales: %v", strings.Join(i18n.Locales(), ", "))

	logAsFatal(initRoutes(appConfigDir()))
	log.Infof("App Route Domain Addresses: %v", strings.Join(appRoutes.DomainAddresses(), ", "))

	// TODO initControllers

	// TODO Validate Routes

	logAsFatal(initViews(appViewsDir()))

	logAsFatal(initTests(appTestsDir()))

}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

func appDir() string {
	if appIsPackaged {
		return AppBaseDir()
	}
	return filepath.Join(AppBaseDir(), "app")
}

func appLogsDir() string {
	return filepath.Join(AppBaseDir(), "logs")
}

func appConfigDir() string {
	return filepath.Join(AppBaseDir(), "config")
}

func appI18nDir() string {
	return filepath.Join(AppBaseDir(), "i18n")
}

func appViewsDir() string {
	return filepath.Join(appDir(), "views")
}

func appTestsDir() string {
	return filepath.Join(AppBaseDir(), "tests")
}

func logAsFatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func initPath(importPath string) error {
	var err error
	goPath, err = ess.GoPath()
	if err != nil {
		return err
	}

	appImportPath = path.Clean(importPath)
	goSrcDir = filepath.Join(goPath, "src")
	appBaseDir = filepath.Join(goSrcDir, filepath.FromSlash(appImportPath))

	if !ess.IsFileExists(appBaseDir) {
		return fmt.Errorf("aah application directory does not exists: %s", appImportPath)
	}

	appIsPackaged = !ess.IsFileExists(appDir())

	return nil
}

func initConfig(cfgDir string) error {
	confPath := filepath.Join(cfgDir, "app.conf")
	if !ess.IsFileExists(confPath) {
		return fmt.Errorf("aah application configuration does not exists: %v", confPath)
	}

	cfg, err := config.LoadFile(confPath)
	if err != nil {
		return err
	}

	appConfig = cfg

	return nil
}

func initLogs(logsDir string, cfg *config.Config) error {
	if logCfg, found := cfg.GetSubConfig("log"); found {
		receiver := logCfg.StringDefault("receiver", "")
		if strings.EqualFold(receiver, "file") {
			file := logCfg.StringDefault("file", "")
			if ess.IsStrEmpty(file) {
				logFileName := strings.Replace(AppName(), " ", "-", -1)
				logCfg.SetString("file", filepath.Join(logsDir, logFileName+".log"))
			} else if !filepath.IsAbs(file) {
				logCfg.SetString("file", filepath.Join(logsDir, file))
			}
		}

		logger, err := log.Newc(logCfg)
		if err != nil {
			return err
		}

		log.SetOutput(logger)
	}

	return nil
}

func initI18n(i18nDir string) error {
	return i18n.LoadMessage(i18nDir)
}

func initRoutes(cfgDir string) error {
	routesPath := filepath.Join(cfgDir, "routes.conf")
	if !ess.IsFileExists(routesPath) {
		return fmt.Errorf("aah application routes configuration does not exists: %v", routesPath)
	}

	appRoutes = router.New(routesPath)
	return appRoutes.Load()
}

func initViews(viewsDir string) error {

	// TODO initViews

	return nil
}

func initTests(testsDir string) error {

	// TODO initTests

	return nil
}
