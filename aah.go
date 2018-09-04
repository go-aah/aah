// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

// Package aah is A secure, flexible, rapid Go web framework.
//
// Visit: https://aahframework.org to know more.
package aah

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"sync"
	"syscall"

	"aahframe.work/aah/ahttp"
	"aahframe.work/aah/ainsp"
	"aahframe.work/aah/aruntime"
	"aahframe.work/aah/cache"
	"aahframe.work/aah/config"
	"aahframe.work/aah/essentials"
	"aahframe.work/aah/i18n"
	"aahframe.work/aah/internal/settings"
	"aahframe.work/aah/log"
	"aahframe.work/aah/router"
	"aahframe.work/aah/security"
	"aahframe.work/aah/vfs"
	"aahframe.work/aah/ws"
)

// BuildInfo holds the aah application build information; such as BinaryName,
// Version and Date.
type BuildInfo struct {
	BinaryName string
	Version    string
	Date       string
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// aah application instance
//______________________________________________________________________________

func newApp() *app {
	aahApp := &app{
		RWMutex: sync.RWMutex{},
		vfs:     new(vfs.VFS),
		settings: &settings.Settings{
			VirtualBaseDir: "/app",
		},
		cacheMgr: cache.NewManager(),
	}

	aahApp.he = &HTTPEngine{
		a:       aahApp,
		ctxPool: new(sync.Pool),
		registry: &ainsp.TargetRegistry{
			Registry:   make(map[string]*ainsp.Target),
			SearchType: ctxPtrType,
		},
	}
	aahApp.he.ctxPool.New = func() interface{} { return aahApp.he.newContext() }

	aahApp.eventStore = &EventStore{
		a:           aahApp,
		subscribers: make(map[string]EventCallbacks),
		mu:          sync.RWMutex{},
	}

	return aahApp
}

// app struct represents aah application.
type app struct {
	sync.RWMutex
	buildInfo      *BuildInfo
	settings       *settings.Settings
	cfg            *config.Config
	vfs            *vfs.VFS
	tlsCfg         *tls.Config
	he             *HTTPEngine
	wse            *ws.Engine
	server         *http.Server
	redirectServer *http.Server
	router         *router.Router
	eventStore     *EventStore
	bindMgr        *bindManager
	i18n           *i18n.I18n
	securityMgr    *security.Manager
	viewMgr        *viewManager
	staticMgr      *staticManager
	errorMgr       *errorManager
	cacheMgr       *cache.Manager
	sc             chan os.Signal
	logger         log.Loggerer
	accessLog      *accessLogger
	dumpLog        *dumpLogger
}

func (a *app) Init(importPath string) error {
	a.settings.ImportPath = path.Clean(importPath)
	var err error

	if a.buildInfo == nil {
		// aah CLI is accessing application for build purpose
		_ = log.SetLevel("warn")
		if err = a.initPath(); err != nil {
			return err
		}
		if err = a.initConfig(); err != nil {
			return err
		}
		if err = a.settings.Refresh(a.Config()); err != nil {
			return err
		}
		if err = a.initRouter(); err != nil {
			return err
		}
		_ = log.SetLevel("debug")
	} else {
		for event := range a.EventStore().subscribers {
			a.EventStore().sortEventSubscribers(event)
		}

		if err = a.initPath(); err != nil {
			return err
		}
		if err = a.initConfig(); err != nil {
			return err
		}

		// publish `OnInit` server event
		a.EventStore().PublishSync(&Event{Name: EventOnInit})

		if err = a.settings.Refresh(a.Config()); err != nil {
			return err
		}
		if err = a.initLog(); err != nil {
			return err
		}
		if err = a.initI18n(); err != nil {
			return err
		}
		if err = a.initSecurity(); err != nil {
			return err
		}
		if err = a.initRouter(); err != nil {
			return err
		}
		if err = a.initBind(); err != nil {
			return err
		}
		if err = a.initView(); err != nil {
			return err
		}
		if err = a.initStatic(); err != nil {
			return err
		}
		if err = a.initError(); err != nil {
			return err
		}
		if a.settings.AccessLogEnabled {
			if err = a.initAccessLog(); err != nil {
				return err
			}
		}
		if a.settings.DumpLogEnabled {
			if err = a.initDumpLog(); err != nil {
				return err
			}
		}
		if a.IsWebSocketEnabled() {
			if a.wse, err = ws.New(a); err != nil {
				return err
			}
		}
	}

	a.settings.Initialized = true
	return nil
}

func (a *app) Name() string {
	if a.BuildInfo() == nil {
		return a.Config().StringDefault("name", path.Base(a.ImportPath()))
	}
	return a.Config().StringDefault("name", a.BuildInfo().BinaryName)
}

func (a *app) InstanceName() string {
	return a.Config().StringDefault("instance_name", "")
}

func (a *app) Type() string {
	return a.Config().StringDefault("type", "")
}

func (a *app) Desc() string {
	return a.Config().StringDefault("desc", "")
}

func (a *app) BaseDir() string {
	return a.settings.BaseDir
}

func (a *app) VirtualBaseDir() string {
	return a.settings.VirtualBaseDir
}

func (a *app) ImportPath() string {
	return a.settings.ImportPath
}

func (a *app) HTTPAddress() string {
	return a.Config().StringDefault("server.address", "")
}

func (a *app) HTTPPort() string {
	port := firstNonZeroString(
		a.Config().StringDefault("server.proxyport", ""),
		a.Config().StringDefault("server.port", settings.DefaultHTTPPort),
	)
	return a.parsePort(port)
}

func (a *app) BuildInfo() *BuildInfo {
	return a.buildInfo
}

func (a *app) SetBuildInfo(bi *BuildInfo) {
	a.buildInfo = bi
}

func (a *app) IsPackaged() bool {
	return a.settings.PackagedMode
}

// TODO remove pack parameter
func (a *app) SetPackaged(pack bool) {
	a.settings.PackagedMode = pack
}

func (a *app) IsEmbeddedMode() bool {
	return a.VFS().IsEmbeddedMode()
}

func (a *app) SetEmbeddedMode() {
	a.VFS().SetEmbeddedMode()
}

func (a *app) Profile() string {
	a.RLock()
	defer a.RUnlock()
	return a.settings.EnvProfile
}

func (a *app) SetProfile(profile string) error {
	a.Lock()
	defer a.Unlock()
	return a.settings.SetProfile(profile)
}

func (a *app) IsProfile(profile string) bool {
	return a.Profile() == profile
}

func (a *app) IsProfileDev() bool {
	return a.IsProfile("dev")
}

func (a *app) IsProfileProd() bool {
	return a.IsProfile("prod")
}

func (a *app) AllProfiles() []string {
	var profiles []string

	for _, v := range a.Config().KeysByPath("env") {
		if v == "default" {
			continue
		}
		profiles = append(profiles, v)
	}

	return profiles
}

func (a *app) IsSSLEnabled() bool {
	return a.settings.SSLEnabled
}

func (a *app) IsLetsEncryptEnabled() bool {
	return a.settings.LetsEncryptEnabled
}

func (a *app) IsWebSocketEnabled() bool {
	return a.cfg.BoolDefault("server.websocket.enable", false)
}

func (a *app) NewChildLogger(fields log.Fields) log.Loggerer {
	return a.Log().WithFields(fields)
}

func (a *app) SetTLSConfig(tlsCfg *tls.Config) {
	a.tlsCfg = tlsCfg
}

func (a *app) AddController(c interface{}, methods []*ainsp.Method) {
	a.HTTPEngine().registry.Add(c, methods)
}

func (a *app) AddWebSocket(w interface{}, methods []*ainsp.Method) {
	a.WSEngine().AddWebSocket(w, methods)
}

func (a *app) HTTPEngine() *HTTPEngine {
	return a.he
}

func (a *app) WSEngine() *ws.Engine {
	if a.wse == nil {
		a.Log().Warn("It seems WebSocket is not enabled, set 'server.websocket.enable' to true." +
			" Refer to https://docs.aahframework.org/websocket.html")
	}
	return a.wse
}

func (a *app) VFS() *vfs.VFS {
	return a.vfs
}

func (a *app) CacheManager() *cache.Manager {
	return a.cacheMgr
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// app Unexported methods
//______________________________________________________________________________

func (a *app) logsDir() string {
	return filepath.Join(a.BaseDir(), "logs")
}

func (a *app) initPath() error {
	defer func() {
		if err := a.VFS().AddMount(a.VirtualBaseDir(), a.BaseDir()); err != nil {
			if perr, ok := err.(*os.PathError); ok && perr == vfs.ErrMountExists {
				// Update app-base-dir to inferred base directory
				if m, err := a.VFS().FindMount(a.VirtualBaseDir()); err == nil {
					m.Proot = a.BaseDir()
				}
			}
		}
	}()

	// Application is packaged, it means built via `aah build`
	if a.IsPackaged() {
		ep, err := os.Executable()
		if err != nil {
			return err
		}

		if a.IsEmbeddedMode() {
			a.settings.BaseDir = filepath.Dir(ep)
		} else if a.settings.BaseDir, err = inferBaseDir(ep); err != nil {
			return err
		}

		a.settings.BaseDir = filepath.Clean(a.settings.BaseDir)
		return nil
	}

	if ess.IsFileExists("go.mod") {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		a.settings.BaseDir = cwd
		fmt.Println("basedir", a.settings.BaseDir)
		return nil
	}

	// If not packaged, get the GOPATH
	gopath, err := ess.GoPath()
	if err != nil {
		return err
	}

	// If its a physical location, we got the app base directory
	if filepath.IsAbs(a.ImportPath()) {
		if !ess.IsFileExists(a.ImportPath()) {
			return fmt.Errorf("path does not exists: %s", a.ImportPath())
		}

		a.settings.BaseDir = filepath.Clean(a.ImportPath())
		a.settings.PhysicalPathMode = true
		return nil
	}

	// Import path mode
	a.settings.BaseDir = filepath.Join(gopath, "src", filepath.FromSlash(a.ImportPath()))
	if !ess.IsFileExists(a.BaseDir()) {
		return fmt.Errorf("import path does not exists: %s", a.ImportPath())
	}

	return nil
}

func (a *app) binaryFilename() string {
	if a.buildInfo == nil {
		return ""
	}
	return ess.StripExt(a.BuildInfo().BinaryName)
}

func (a *app) parsePort(port string) string {
	if !ess.IsStrEmpty(port) {
		return port
	}

	if a.IsSSLEnabled() {
		return "443"
	}

	return "80"
}

func (a *app) aahRecover() {
	if r := recover(); r != nil {
		strace := aruntime.NewStacktrace(r, a.Config())
		buf := acquireBuffer()
		defer releaseBuffer(buf)
		strace.Print(buf)

		a.Log().Error("Recovered from panic:")
		a.Log().Error(buf.String())
	}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Config Definitions
//______________________________________________________________________________

func (a *app) Config() *config.Config {
	return a.cfg
}

func (a *app) initConfig() error {
	cfg, err := config.VFSLoadFile(a.VFS(), path.Join(a.VirtualBaseDir(), "config", "aah.conf"))
	if err != nil {
		return fmt.Errorf("aah.conf: %s", err)
	}

	a.cfg = cfg
	return nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Log Definitions
//______________________________________________________________________________

// Log method returns app logger instance.
func (a *app) Log() log.Loggerer {
	return a.logger
}

// AddLoggerHook method adds given logger into aah application default logger.
func (a *app) AddLoggerHook(name string, hook log.HookFunc) error {
	return a.Log().(*log.Logger).AddHook(name, hook)
}

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

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// i18n Definitions
//______________________________________________________________________________

const keyLocale = "Locale"

func (a *app) I18n() *i18n.I18n {
	return a.i18n
}

// DefaultI18nLang method returns application i18n default language if
// configured otherwise framework defaults to "en".
func (a *app) DefaultI18nLang() string {
	return a.Config().StringDefault("i18n.default", "en")
}

func (a *app) initI18n() error {
	i18nPath := path.Join(a.VirtualBaseDir(), "i18n")
	if !a.VFS().IsExists(i18nPath) {
		// i18n directory not exists, scenario could be only API application
		return nil
	}

	ai18n := i18n.NewWithVFS(a.VFS())
	ai18n.DefaultLocale = a.DefaultI18nLang()
	if err := ai18n.Load(i18nPath); err != nil {
		return err
	}

	a.i18n = ai18n
	return nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// App - Engines
//______________________________________________________________________________

// ServeHTTP method implementation of http.Handler interface.
func (a *app) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer a.aahRecover()
	if a.settings.Redirect {
		if a.he.doRedirect(w, r) {
			return
		}
	}

	if h := r.Header[ahttp.HeaderUpgrade]; len(h) > 0 {
		if h[0] == "websocket" || h[0] == "Websocket" {
			a.wse.Handle(w, r)
			return
		}
	}

	a.he.Handle(w, r)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// HotReload Definitions for Prod profile
//______________________________________________________________________________

func (a *app) listenForHotReload() {
	if a.IsProfile(settings.DefaultEnvProfile) || !a.IsPackaged() {
		return
	}

	a.sc = make(chan os.Signal, 2)
	signal.Notify(a.sc, syscall.SIGHUP)
	for {
		<-a.sc
		a.Log().Warn("Hangup signal (SIGHUP) received")
		a.performHotReload()
	}
}

func (a *app) performHotReload() {
	a.settings.HotReload = true
	defer func() { a.settings.HotReload = false }()

	activeProfile := a.Profile()

	a.Log().Info("Application hot-reload and reinitialization starts ...")
	var err error

	if err = a.initConfig(); err != nil {
		a.Log().Errorf("Unable to reload aah.conf: %v", err)
		return
	}
	a.Log().Info("Configuration files reload succeeded")

	// Set activeProfile into reloaded configuration
	a.Config().SetString("env.active", activeProfile)

	if err = a.settings.Refresh(a.Config()); err != nil {
		a.Log().Errorf("Unable to reinitialize aah application settings: %v", err)
		return
	}
	a.Log().Info("Configuration values reinitialize succeeded")

	if err = a.initLog(); err != nil {
		a.Log().Errorf("Unable to reinitialize application logger: %v", err)
		return
	}
	a.Log().Info("Logging reinitialize succeeded")

	if err = a.initI18n(); err != nil {
		a.Log().Errorf("Unable to reinitialize application i18n: %v", err)
		return
	}
	if a.Type() == "web" {
		a.Log().Info("I18n reinitialize succeeded")
	}

	if err = a.initRouter(); err != nil {
		a.Log().Errorf("Unable to reinitialize application %v", err)
		return
	}
	a.Log().Info("Router reinitialize succeeded")

	if err = a.initView(); err != nil {
		a.Log().Errorf("Unable to reinitialize application views: %v", err)
		return
	}
	if a.Type() == "web" {
		a.Log().Info("View engine reinitialize succeeded")
	}

	if err = a.initSecurity(); err != nil {
		a.Log().Errorf("Unable to reinitialize application security manager: %v", err)
		return
	}
	a.Log().Info("Security reinitialize succeeded")

	if a.settings.AccessLogEnabled {
		if err = a.initAccessLog(); err != nil {
			a.Log().Errorf("Unable to reinitialize application access log: %v", err)
			return
		}
		a.Log().Info("Access logging reinitialize succeeded")
	}

	if a.settings.DumpLogEnabled {
		if err = a.initDumpLog(); err != nil {
			a.Log().Errorf("Unable to reinitialize application dump log: %v", err)
			return
		}
		a.Log().Info("Server dump logging reinitialize succeeded")
	}

	a.Log().Info("Application hot-reload and reinitialization was successful")
}

func inferBaseDir(p string) (string, error) {
	for {
		p = filepath.Dir(p)
		if p == "/" || p == "." || len(p) == 3 {
			break
		}
		if ess.IsFileExists(filepath.Join(p, "config")) {
			return p, nil
		}
	}
	return "", errors.New("aah: config directory not found in parent directories")
}
