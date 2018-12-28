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
	"html/template"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"

	"aahframe.work/ahttp"
	"aahframe.work/ainsp"
	"aahframe.work/aruntime"
	"aahframe.work/aruntime/diagnosis"
	"aahframe.work/cache"
	"aahframe.work/config"
	"aahframe.work/console"
	"aahframe.work/essentials"
	"aahframe.work/i18n"
	"aahframe.work/internal/settings"
	"aahframe.work/log"
	"aahframe.work/router"
	"aahframe.work/security"
	"aahframe.work/security/acrypto"
	"aahframe.work/security/session"
	"aahframe.work/valpar"
	"aahframe.work/vfs"
	"aahframe.work/view"
	"aahframe.work/ws"
	"github.com/go-aah/forge"
	"gopkg.in/go-playground/validator.v9"
)

// BuildInfo holds the aah application build information; such as BinaryName,
// Version and Date.
type BuildInfo struct {
	BinaryName string
	Version    string
	Timestamp  string
	AahVersion string // introduced in v0.12.0
	GoVersion  string // introduced in v0.12.0
}

var defaultApp = newApp()

// App method returns the aah application instance.
func App() *Application {
	return defaultApp
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// aah application instance
//______________________________________________________________________________

func newApp() *Application {
	aahApp := &Application{
		RWMutex: sync.RWMutex{},
		cli:     console.NewApp(),
		vfs:     new(vfs.VFS),
		settings: &settings.Settings{
			VirtualBaseDir: "/app",
		},
		cacheMgr: cache.NewManager(),
	}
	aahApp.cli.Commands = make([]console.Command, 0)

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

	aahApp.logger, _ = log.New(config.NewEmpty())

	return aahApp
}

// Application struct represents aah application.
type Application struct {
	sync.RWMutex
	buildInfo      *BuildInfo
	settings       *settings.Settings
	cli            *console.Application
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
	diagnosis      *diagnosis.Diagnosis
}

// InitForCLI method is for purpose aah CLI tool. IT IS NOT FOR AAH USER.
// Introduced in v0.12.0 release.
func (a *Application) InitForCLI(importPath string) error {
	a.settings.ImportPath = path.Clean(importPath)
	_ = a.Log().(*log.Logger).SetLevel("warn")
	var err error
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
	_ = a.Log().(*log.Logger).SetLevel("debug")
	return nil
}

// Name method returns aah application name from app config `name` otherwise
// app name of the base directory.
func (a *Application) Name() string {
	if a.BuildInfo() == nil {
		return a.Config().StringDefault("name", path.Base(a.ImportPath()))
	}
	return a.Config().StringDefault("name", a.BuildInfo().BinaryName)
}

// InstanceName method returns aah application instane name from app config
// `instance_name` otherwise empty string.
//
// Value of `instance_name` from `aah.conf`.
func (a *Application) InstanceName() string {
	return a.Config().StringDefault("instance_name", "")
}

// Type method returns aah application type info e.g.: web, api, websocket.
//
// Value of `type` from `aah.conf`.
func (a *Application) Type() string {
	return a.Config().StringDefault("type", "")
}

// Desc method returns aah application friendly description from app config
// otherwise empty string.
//
// Value of `desc` from `aah.conf`.
func (a *Application) Desc() string {
	return a.Config().StringDefault("desc", "")
}

// Copyrights method returns application copyrights info from configuration.
//
// Value of `copyrights` from `aah.conf`.
func (a *Application) Copyrights() string {
	return a.Config().StringDefault("copyrights", "© aah framework")
}

// BaseDir method returns the application base or binary's base directory
// 	For e.g.:
// 		$GOPATH/src/github.com/user/myproject
// 		<path/to/the/aah/myproject>
// 		<app/binary/path/base/directory>
func (a *Application) BaseDir() string {
	return a.settings.BaseDir
}

// VirtualBaseDir method returns "/app". In `v0.11.0` Virtual FileSystem (VFS)
// introduced in aah to provide single binary build packaging and provides
// seamless experience of Read-Only access to application directory and its sub-tree
// across OS platforms via `aah.App().VFS()`.
func (a *Application) VirtualBaseDir() string {
	return a.settings.VirtualBaseDir
}

// ImportPath method returns the application Go import path.
func (a *Application) ImportPath() string {
	return a.settings.ImportPath
}

// HTTPAddress method returns aah application HTTP address otherwise empty string
//
// Value of `server.address` from `aah.conf`.
func (a *Application) HTTPAddress() string {
	return a.Config().StringDefault("server.address", "")
}

// HTTPPort method returns aah application HTTP port number based on `server.port`
// value. Possible outcomes are user-defined port, `80`, `443` and `8080`.
func (a *Application) HTTPPort() string {
	port := firstNonZeroString(
		a.Config().StringDefault("server.proxyport", ""),
		a.Config().StringDefault("server.port", settings.DefaultHTTPPort),
	)
	return a.parsePort(port)
}

// BuildInfo method return user application version no.
func (a *Application) BuildInfo() *BuildInfo {
	return a.buildInfo
}

// SetBuildInfo method sets the user application build info into aah instance.
func (a *Application) SetBuildInfo(bi *BuildInfo) {
	a.buildInfo = bi
}

// IsPackaged method returns true when application built for deployment.
func (a *Application) IsPackaged() bool {
	return a.settings.PackagedMode
}

// SetPackaged method sets the info of binary is packaged or not.
//
// It is used by framework during application startup. IT'S NOT FOR AAH USER(S).
func (a *Application) SetPackaged(pack bool) {
	a.settings.PackagedMode = pack
}

// EnvProfile returns active environment profile name of aah application.
// For e.g.: dev, prod, etc. Default is `dev`.
//
// Value of `env.active` from `aah.conf`.
func (a *Application) EnvProfile() string {
	a.RLock()
	defer a.RUnlock()
	return a.settings.EnvProfile
}

// IsEnvProfile method returns true if given environment profile match with active
// environment in aah application otherwise false.
func (a *Application) IsEnvProfile(envProfile string) bool {
	return a.EnvProfile() == envProfile
}

// EnvProfiles method returns all available environment profile names from aah
// application.
func (a *Application) EnvProfiles() []string {
	var profiles []string

	for _, v := range a.Config().KeysByPath("env") {
		if v == "default" {
			continue
		}
		profiles = append(profiles, v)
	}

	return profiles
}

// IsSSLEnabled method returns true if aah application is enabled with SSL
// otherwise false.
func (a *Application) IsSSLEnabled() bool {
	return a.settings.SSLEnabled
}

// IsLetsEncryptEnabled method returns true if aah application is enabled with
// Let's Encrypt certs otherwise false.
func (a *Application) IsLetsEncryptEnabled() bool {
	return a.settings.LetsEncryptEnabled
}

// IsWebSocketEnabled method returns to true if aah application enabled with
// WebSocket feature.
//
// Value of `server.websocket.enable` from `aah.conf`.
func (a *Application) IsWebSocketEnabled() bool {
	return a.cfg.BoolDefault("server.websocket.enable", false)
}

// NewChildLogger method create a child logger from aah application default logger.
func (a *Application) NewChildLogger(fields log.Fields) log.Loggerer {
	return a.Log().WithFields(fields)
}

// SetTLSConfig method is used to set custom TLS config for aah server.
// Note: if `server.ssl.lets_encrypt.enable=true` then framework sets the
// `GetCertificate` from autocert manager.
//
// Use `aah.OnInit` or `func init() {...}` to assign your custom TLS Config.
func (a *Application) SetTLSConfig(tlsCfg *tls.Config) {
	a.tlsCfg = tlsCfg
}

// HTTPEngine method returns aah HTTP engine.
func (a *Application) HTTPEngine() *HTTPEngine {
	return a.he
}

// WSEngine method returns aah WebSocket engine.
//
// Note: It could be nil if WebSocket is not enabled.
func (a *Application) WSEngine() *ws.Engine {
	if a.wse == nil {
		a.Log().Warn("It seems WebSocket is not enabled, set 'server.websocket.enable' to true." +
			" Refer to https://docs.aahframework.org/websocket.html")
	}
	return a.wse
}

// VFS method returns aah Virtual FileSystem instance.
func (a *Application) VFS() *vfs.VFS {
	return a.vfs
}

// CacheManager returns aah application cache manager.
func (a *Application) CacheManager() *cache.Manager {
	return a.cacheMgr
}

// EventStore method returns aah application event store.
func (a *Application) EventStore() *EventStore {
	return a.eventStore
}

// Router method returns aah application router instance.
func (a *Application) Router() *router.Router {
	return a.router
}

// SecurityManager method returns the application security instance,
// which manages the Session, CORS, CSRF, Security Headers, etc.
func (a *Application) SecurityManager() *security.Manager {
	return a.securityMgr
}

// SessionManager method returns the application session manager.
// By default session is stateless.
func (a *Application) SessionManager() *session.Manager {
	return a.SecurityManager().SessionManager
}

// ViewEngine method returns aah application view Engine instance.
func (a *Application) ViewEngine() view.Enginer {
	if a.viewMgr == nil {
		return nil
	}
	return a.viewMgr.engine
}

// Validator method return the default validator of aah framework.
//
// Refer to https://godoc.org/gopkg.in/go-playground/validator.v9 for detailed
// documentation.
func (a *Application) Validator() *validator.Validate {
	return valpar.Validator()
}

// SetMinifier method sets the given minifier func into aah framework.
// Note: currently minifier is called only for HTML contentType.
func (a *Application) SetMinifier(fn MinifierFunc) {
	if a.viewMgr == nil {
		a.viewMgr = &viewManager{a: a}
	}

	if a.viewMgr.minifier != nil {
		a.Log().Warnf("Changing Minifier from: '%s'  to '%s'",
			ess.GetFunctionInfo(a.viewMgr.minifier).QualifiedName, ess.GetFunctionInfo(fn).QualifiedName)
	}
	a.viewMgr.minifier = fn
}

// SetErrorHandler method is used to register custom centralized application
// error handler. If custom handler is not then default error handler takes place.
func (a *Application) SetErrorHandler(handlerFunc ErrorHandlerFunc) {
	a.errorMgr.SetHandler(handlerFunc)
}

// AddController method adds given controller into controller registory.
func (a *Application) AddController(c interface{}, methods []*ainsp.Method) {
	a.HTTPEngine().registry.Add(c, methods)
}

// AddWebSocket method adds given WebSocket into WebSocket registry.
func (a *Application) AddWebSocket(w interface{}, methods []*ainsp.Method) {
	a.WSEngine().AddWebSocket(w, methods)
}

// AddTemplateFunc method adds template func map into view engine.
func (a *Application) AddTemplateFunc(funcs template.FuncMap) {
	view.AddTemplateFunc(funcs)
}

// AddViewEngine method adds the given name and view engine to view store.
func (a *Application) AddViewEngine(name string, engine view.Enginer) error {
	return view.AddEngine(name, engine)
}

// AddSessionStore method allows you to add custom session store which
// implements `session.Storer` interface. Then configure `name` parameter in the
// configfuration as `session.store.type = "name"`.
func (a *Application) AddSessionStore(name string, store session.Storer) error {
	return session.AddStore(name, store)
}

// AddPasswordAlgorithm method adds given password algorithm to encoders list.
// Implementation have to implement interface `PasswordEncoder`.
//
// Then you can use it in the configuration `security.auth_schemes.*`.
func (a *Application) AddPasswordAlgorithm(name string, encoder acrypto.PasswordEncoder) error {
	return acrypto.AddPasswordAlgorithm(name, encoder)
}

// AddValueParser method adds given custom value parser for the `reflect.Type`
func (a *Application) AddValueParser(typ reflect.Type, parser valpar.Parser) error {
	return valpar.AddValueParser(typ, parser)
}

// AddCommand method adds the aah application CLI commands. Introduced in v0.12.0 release
// aah application binary fully compliant using module console and POSIX flags.
func (a *Application) AddCommand(cmds ...console.Command) error {
	for _, cmd := range cmds {
		name := strings.ToLower(cmd.Name)
		if name == "run" || name == "vfs" || name == "help" {
			return fmt.Errorf("aah: reserved command name '%s' cannot be used", name)
		}
		for _, c := range a.cli.Commands {
			if c.Name == name {
				return fmt.Errorf("aah: command name '%s' already exists", name)
			}
		}
		a.cli.Commands = append(a.cli.Commands, cmd)
	}
	return nil
}

// Validate method is to validate struct via underneath validator.
//
// Returns:
//
//  - For validation errors: returns `validator.ValidationErrors` and nil
//
//  - For invalid input: returns nil, error (invalid input such as nil, non-struct, etc.)
//
//  - For no validation errors: nil, nil
func (a *Application) Validate(s interface{}) (validator.ValidationErrors, error) {
	return valpar.Validate(s)
}

// ValidateValue method is to validate individual value on demand.
//
// Returns -
//
//  - true: validation passed
//
//  - false: validation failed
//
// For example:
//
// 	i := 15
// 	result := valpar.ValidateValue(i, "gt=1,lt=10")
//
// 	emailAddress := "sample@sample"
// 	result := valpar.ValidateValue(emailAddress, "email")
//
// 	numbers := []int{23, 67, 87, 23, 90}
// 	result := valpar.ValidateValue(numbers, "unique")
func (a *Application) ValidateValue(v interface{}, rules string) bool {
	return valpar.ValidateValue(v, rules)
}

// Run method initializes `aah` application and runs the given command.
// If anything goes wrong during an initialize process, it would return an error.
func (a *Application) Run(args []string) error {
	var err error
	a.settings.SetImportPath(args) // only needed for development via CLI
	if err = a.initPath(); err != nil {
		return err
	}
	if err = a.initConfig(); err != nil {
		return err
	}
	a.initCli()
	return a.cli.Run(args)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// app Unexported methods
//______________________________________________________________________________

func (a *Application) logsDir() string {
	return filepath.Join(a.BaseDir(), "logs")
}

func (a *Application) initPath() error {
	defer func() {
		if err := a.VFS().AddMount(a.VirtualBaseDir(), a.BaseDir()); err != nil {
			if perr, ok := err.(*os.PathError); ok && perr == vfs.ErrMountExists {
				// Update app-base-dir to inferred base directory
				if m, err := a.VFS().FindMount(a.VirtualBaseDir()); err == nil {
					m.Proot = a.BaseDir()
				}
			}
		}
		forge.RegisterFS(&aahVFS{fs: a.VFS()})
	}()

	// Application is packaged, it means built via `aah build`
	if a.IsPackaged() {
		ep, err := os.Executable()
		if err != nil {
			return err
		}

		if a.VFS().IsEmbeddedMode() {
			a.settings.BaseDir = filepath.Dir(ep)
		} else if a.settings.BaseDir, err = inferBaseDir(ep); err != nil {
			return err
		}

		a.settings.BaseDir = filepath.Clean(a.settings.BaseDir)
		return nil
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

	if ess.IsFileExists("go.mod") || ess.IsFileExists("aah.project") {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		a.settings.BaseDir = cwd
		return nil
	}

	// If not packaged, get the GOPATH
	gopath, err := ess.GoPath()
	if err != nil {
		return err
	}

	// Import path mode
	a.settings.BaseDir = filepath.Join(gopath, "src", filepath.FromSlash(a.ImportPath()))
	if !ess.IsFileExists(a.BaseDir()) {
		return fmt.Errorf("import path does not exists: %s", a.ImportPath())
	}

	return nil
}

func (a *Application) initApp() error {
	var err error
	for event := range a.EventStore().subscribers {
		a.EventStore().sortEventSubscribers(event)
	}
	a.EventStore().PublishSync(&Event{Name: EventOnInit}) // publish `OnInit` server event
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
	if err := a.CacheManager().InitProviders(a.Config(), a.Log()); err != nil {
		return err
	}
	a.settings.Initialized = true
	return nil
}

func (a *Application) binaryFilename() string {
	if a.buildInfo == nil {
		return ""
	}
	return ess.StripExt(a.BuildInfo().BinaryName)
}

func (a *Application) parsePort(port string) string {
	if !ess.IsStrEmpty(port) {
		return port
	}

	if a.IsSSLEnabled() {
		return "443"
	}

	return "80"
}

func (a *Application) aahRecover() {
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

// Config method returns aah application configuration instance.
func (a *Application) Config() *config.Config {
	return a.cfg
}

func (a *Application) initConfig() error {
	cfg, err := config.LoadFile(path.Join(a.VirtualBaseDir(), "config", "aah.conf"))
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
func (a *Application) Log() log.Loggerer {
	return a.logger
}

// AddLoggerHook method adds given logger into aah application default logger.
func (a *Application) AddLoggerHook(name string, hook log.HookFunc) error {
	return a.Log().(*log.Logger).AddHook(name, hook)
}

func (a *Application) initLog() error {
	if !a.Config().IsExists("log") {
		a.Log().Warn("Section 'log { ... }' configuration does not exists, initializing app logger with default values.")
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

// I18n method returns aah application I18n store instance.
func (a *Application) I18n() *i18n.I18n {
	return a.i18n
}

// DefaultI18nLang method returns application i18n default language if
// configured otherwise framework defaults to "en".
func (a *Application) DefaultI18nLang() string {
	return a.Config().StringDefault("i18n.default", "en")
}

func (a *Application) initI18n() error {
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
func (a *Application) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

func (a *Application) listenForHotReload() {
	if !a.settings.HotReloadEnabled || a.IsEnvProfile(settings.DefaultEnvProfile) || !a.IsPackaged() {
		return
	}
	if runtime.GOOS == "windows" && (a.settings.HotReloadSignalStr == "SIGUSR1" ||
		a.settings.HotReloadSignalStr == "SIGUSR2") {
		a.Log().Warn("OS Windows does not support signal SIGUSR1/SIGUSR2 let's fallback to default SIGHUP")
	}
	a.sc = make(chan os.Signal, 1)
	signal.Notify(a.sc, a.settings.HotReloadSignal())
	for {
		<-a.sc
		a.Log().Warnf("Hangup signal (%s) received", a.settings.HotReloadSignalStr)
		a.performHotReload()
	}
}

func (a *Application) performHotReload() {
	a.settings.HotReload = true
	defer func() { a.settings.HotReload = false }()

	activeProfile := a.EnvProfile()

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
	a.EventStore().PublishSync(&Event{Name: EventOnConfigHotReload})
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

type aahVFS struct {
	fs *vfs.VFS
}

func (f aahVFS) Open(filename string) (io.Reader, error) {
	return f.fs.Open(checkToSlash(filename))
}

func (f aahVFS) Glob(pattern string) (matches []string, err error) {
	return f.fs.Glob(checkToSlash(pattern))
}

func checkToSlash(value string) string {
	if strings.HasPrefix(value, "\\app\\") {
		return filepath.ToSlash(value)
	}
	return value
}
