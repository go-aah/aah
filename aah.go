// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"aahframework.org/aah/aruntime"
	"aahframework.org/aah/atemplate"
	"aahframework.org/aah/i18n"
	"aahframework.org/aah/render"
	"aahframework.org/aah/router"
	"aahframework.org/config"
	"aahframework.org/essentials"
	"aahframework.org/log"
)

// Version no. of aah framework
const Version = "0.2"

// aah application variables
var (
	appName                  string
	appImportPath            string
	appProfile               string
	appBaseDir               string
	appIsPackaged            bool
	appConfig                *config.Config
	appHTTPReadTimeout       time.Duration
	appHTTPWriteTimeout      time.Duration
	appSSLCert               string
	appSSLKey                string
	appMultipartMaxMemory    int64
	appTemplateEngine        atemplate.TemplateEnginer
	appTemplateExt           string
	appTemplateCaseSensitive bool
	appPID                   int

	appInitialized     bool
	isExternalTmplEng  bool
	isMultipartEnabled bool

	goPath   string
	goSrcDir string

	appDefaultProfile        = "dev"
	appProfilePrefix         = "env."
	appDefaultHTTPPort       = 8000
	appDefaultDateFormat     = "2006-01-02"
	appDefaultDateTimeFormat = "2006-01-02 15:04:05"
	appDefaultTmplLayout     = "master"
	appModeWeb               = "web"
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
// 		<app/binary/path/base/directory>
func AppBaseDir() string {
	if appIsPackaged {
		wd, _ := os.Getwd()
		if strings.HasSuffix(wd, "/bin") {
			wd = wd[:len(wd)-4]
		}
		return wd
	}
	return appBaseDir
}

// AppImportPath method returns the application Go import path.
func AppImportPath() string {
	return appImportPath
}

// AppConfig method returns aah application configuration instance.
func AppConfig() *config.Config {
	return appConfig
}

// AppMode method returns aah application mode. Default is "web" For e.g.: web or api
func AppMode() string {
	return AppConfig().StringDefault("mode", appModeWeb)
}

// AppHTTPAddress method returns aah application HTTP address otherwise empty string
func AppHTTPAddress() string {
	return AppConfig().StringDefault("http.address", "")
}

// AppHTTPPort method returns aah application HTTP port number if available
// otherwise returns default port number 8000.
func AppHTTPPort() int {
	return AppConfig().IntDefault("http.port", appDefaultHTTPPort)
}

// AppDateFormat method returns aah application date format
func AppDateFormat() string {
	return AppConfig().StringDefault("format.date", appDefaultDateFormat)
}

// AppDateTimeFormat method returns aah application date format
func AppDateTimeFormat() string {
	return AppConfig().StringDefault("format.datetime", appDefaultDateTimeFormat)
}

// AppDefaultI18nLang method returns aah application i18n default language if
// configured other framework defaults to "en".
func AppDefaultI18nLang() string {
	return AppConfig().StringDefault("i18n.default", "en")
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

// IsSSLEnabled method returns true if aah application is enabled with SSL
// otherwise false.
func IsSSLEnabled() bool {
	return AppConfig().BoolDefault("http.ssl.enable", false)
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
	if err := AppConfig().SetProfile(appProfilePrefix + profile); err != nil {
		return err
	}

	appProfile = profile
	return nil
}

// AddTemplateFunc method adds given Go template funcs into function map.
func AddTemplateFunc(funcMap template.FuncMap) {
	atemplate.AddTemplateFunc(funcMap)
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

// Init method initializes `aah` application, if anything goes wrong during
// initialize process, it will log it as fatal msg and exit.
func Init(importPath string) {
	defer aahRecover()

	logAsFatal(initPath(importPath))
	logAsFatal(initConfig(appConfigDir()))

	initInternal()
}

// Start method starts the HTTP server based on aah config "http.*".
func Start() {
	defer aahRecover()

	if !appInitialized {
		log.Fatal("aah application is not initialized, call `aah.Init` before the `aah.Start`.")
	}

	log.Infof("App Name: %v", AppName())
	log.Infof("App Profile: %v", AppProfile())
	log.Infof("App Mode: %v", AppMode())
	log.Debugf("App i18n Locales: %v", strings.Join(i18n.Locales(), ", "))
	log.Debugf("App Route Domains: %v", strings.Join(router.DomainAddresses(), ", "))

	address := AppHTTPAddress()
	server := &http.Server{
		Handler:      newEngine(),
		ReadTimeout:  appHTTPReadTimeout,
		WriteTimeout: appHTTPWriteTimeout,
	}

	writePID(AppName(), AppBaseDir(), AppConfig())

	// Unix Socket
	if strings.HasPrefix(address, "unix") {
		log.Infof("Listening and serving HTTP on %v", address)

		sockFile := address[5:]
		if err := os.Remove(sockFile); !os.IsNotExist(err) {
			logAsFatal(err)
		}

		listener, err := net.Listen("unix", sockFile)
		logAsFatal(err)

		defer func() {
			_ = listener.Close()
		}()

		server.Addr = address
		logAsFatal(server.Serve(listener))

		return
	}

	server.Addr = fmt.Sprintf("%s:%s", AppHTTPAddress(), strconv.Itoa(AppHTTPPort()))

	// HTTPS
	if IsSSLEnabled() {
		log.Infof("Listening and serving HTTPS on %v", server.Addr)
		logAsFatal(server.ListenAndServeTLS(appSSLCert, appSSLKey))
		return
	}

	// HTTP
	log.Infof("Listening and serving HTTP on %v", server.Addr)
	logAsFatal(server.ListenAndServe())
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
	return filepath.Join(AppBaseDir(), "views")
}

func appTestsDir() string {
	return filepath.Join(AppBaseDir(), "tests")
}

func logAsFatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func initInternal() {
	logAsFatal(initAppVariables())

	logAsFatal(initLogs(appLogsDir(), AppConfig()))

	logAsFatal(initI18n(appI18nDir()))

	logAsFatal(initRoutes(appConfigDir()))

	if AppMode() == appModeWeb {
		logAsFatal(initTemplateEngine(appViewsDir(), AppConfig()))
	}

	logAsFatal(initTests(appTestsDir()))

	appInitialized = true
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

	appIsPackaged = !ess.IsFileExists(appDir())

	if !appIsPackaged && !ess.IsFileExists(appBaseDir) {
		return fmt.Errorf("aah application does not exists: %s", appImportPath)
	}

	return nil
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

func initAppVariables() error {
	var err error

	appName = AppConfig().StringDefault("name", filepath.Base(appBaseDir))
	appProfile = AppConfig().StringDefault("env.active", appDefaultProfile)

	readTimeout := AppConfig().StringDefault("http.timeout.read", "90s")
	writeTimeout := AppConfig().StringDefault("http.timeout.write", "90s")
	if !(strings.HasSuffix(readTimeout, "s") || strings.HasSuffix(readTimeout, "m")) &&
		!(strings.HasSuffix(writeTimeout, "s") || strings.HasSuffix(writeTimeout, "m")) {
		return errors.New("'http.timeout.{read|write}' value is not a valid time unit")
	}

	if appHTTPReadTimeout, err = time.ParseDuration(readTimeout); err != nil {
		return fmt.Errorf("app config - 'http.timeout.read': %s", err)
	}

	if appHTTPWriteTimeout, err = time.ParseDuration(writeTimeout); err != nil {
		return fmt.Errorf("app config - 'http.timeout.write': %s", err)
	}

	appSSLCert = AppConfig().StringDefault("http.ssl.cert", "")
	appSSLKey = AppConfig().StringDefault("http.ssl.key", "")
	if IsSSLEnabled() && (ess.IsStrEmpty(appSSLCert) || ess.IsStrEmpty(appSSLKey)) {
		return errors.New("HTTP SSL is enabled, so 'http.ssl.cert' & 'http.ssl.key' value is required")
	}

	logAsFatal(SetAppProfile(AppProfile()))

	render.Init(AppConfig())

	appTemplateExt = AppConfig().StringDefault("template.ext", ".html")

	multipartMemoryStr := AppConfig().StringDefault("render.multipart.size", "32mb")
	if appMultipartMaxMemory, err = ess.StrToBytes(multipartMemoryStr); err != nil {
		return err
	}

	isMultipartEnabled = AppConfig().BoolDefault("render.multipart.enable", true)

	appTemplateCaseSensitive = AppConfig().BoolDefault("template.case_sensitive", false)

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
	return i18n.Load(i18nDir)
}

func initRoutes(cfgDir string) error {
	routesPath := filepath.Join(cfgDir, "routes.conf")

	if err := router.Load(routesPath); err != nil {
		return fmt.Errorf("routes.conf: %s", err)
	}

	return nil
}

func initTemplateEngine(viewsDir string, cfg *config.Config) error {
	// initialize only if external TemplateEngine is not registered.
	if appTemplateEngine == nil {
		appTemplateEngine = &atemplate.TemplateEngine{}
		isExternalTmplEng = false
	} else {
		isExternalTmplEng = true
	}

	appTemplateEngine.Init(AppConfig(), viewsDir)
	return appTemplateEngine.Load()
}

func initTests(testsDir string) error {

	// TODO initTests

	return nil
}

func writePID(appName, appBaseDir string, cfg *config.Config) {
	appPID = os.Getpid()
	pidfile := cfg.StringDefault("pidfile", appName+".pid")
	if !filepath.IsAbs(pidfile) {
		pidfile = filepath.Join(appBaseDir, pidfile)
	}

	if err := ioutil.WriteFile(pidfile, []byte(strconv.Itoa(appPID)), 0644); err != nil {
		log.Error(err)
	}
}
