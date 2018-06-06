// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org/aah source code and usage is governed by a MIT style
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
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"aahframework.org/ahttp.v0"
	"aahframework.org/ainsp.v0"
	"aahframework.org/aruntime.v0"
	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/i18n.v0"
	"aahframework.org/log.v0"
	"aahframework.org/router.v0"
	"aahframework.org/security.v0"
	"aahframework.org/vfs.v0"
	"aahframework.org/ws.v0"
	"golang.org/x/crypto/acme/autocert"
)

const (
	defaultEnvProfile = "dev"
	profilePrefix     = "env."
	defaultHTTPPort   = "8080"
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
		vfs: new(vfs.VFS),
		mu:  new(sync.Mutex),
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
		mu:          new(sync.Mutex),
	}

	return aahApp
}

// app struct represents aah application.
type app struct {
	physicalPathMode       bool
	packagedMode           bool
	serverHeaderEnabled    bool
	requestIDEnabled       bool
	gzipEnabled            bool
	secureHeadersEnabled   bool
	accessLogEnabled       bool
	staticAccessLogEnabled bool
	dumpLogEnabled         bool
	initialized            bool
	hotReload              bool
	pid                    int
	httpMaxHdrBytes        int
	multipartMaxMemory     int64
	maxBodyBytes           int64
	importPath             string
	baseDir                string
	envProfile             string
	sslCert                string
	sslKey                 string
	serverHeader           string
	requestIDHeaderKey     string
	secureJSONPrefix       string
	shutdownGraceTimeStr   string
	httpReadTimeout        time.Duration
	httpWriteTimeout       time.Duration
	shutdownGraceTimeout   time.Duration
	buildInfo              *BuildInfo
	defaultContentType     *ahttp.ContentType

	cfg            *config.Config
	vfs            *vfs.VFS
	tlsCfg         *tls.Config
	he             *HTTPEngine
	wse            *ws.Engine
	server         *http.Server
	redirectServer *http.Server
	autocertMgr    *autocert.Manager
	router         *router.Router
	eventStore     *EventStore
	bindMgr        *bindManager
	i18n           *i18n.I18n
	securityMgr    *security.Manager
	viewMgr        *viewManager
	staticMgr      *staticManager
	errorMgr       *errorManager
	sc             chan os.Signal

	logger    log.Loggerer
	accessLog *accessLogger
	dumpLog   *dumpLogger

	mu *sync.Mutex
}

func (a *app) Init(importPath string) error {
	a.importPath = path.Clean(importPath)
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
		if err = a.initConfigValues(); err != nil {
			return err
		}
		if err = a.initRouter(); err != nil {
			return err
		}
		_ = log.SetLevel("debug")
	} else {
		if err = a.initPath(); err != nil {
			return err
		}
		if err = a.initConfig(); err != nil {
			return err
		}

		// publish `OnInit` server event
		a.EventStore().sortAndPublishSync(&Event{Name: EventOnInit})

		if err = a.initConfigValues(); err != nil {
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
		if a.accessLogEnabled {
			if err = a.initAccessLog(); err != nil {
				return err
			}
		}
		if a.dumpLogEnabled {
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

	a.initialized = true
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
	return a.baseDir
}

func (a *app) VirtualBaseDir() string {
	return "/app"
}

func (a *app) ImportPath() string {
	return a.importPath
}

func (a *app) HTTPAddress() string {
	return a.Config().StringDefault("server.address", "")
}

func (a *app) HTTPPort() string {
	port := firstNonZeroString(
		a.Config().StringDefault("server.proxyport", ""),
		a.Config().StringDefault("server.port", defaultHTTPPort),
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
	return a.packagedMode
}

// TODO remove pack parameter
func (a *app) SetPackaged(pack bool) {
	a.packagedMode = pack
}

func (a *app) IsEmbeddedMode() bool {
	return a.VFS().IsEmbeddedMode()
}

func (a *app) SetEmbeddedMode() {
	a.VFS().SetEmbeddedMode()
}

func (a *app) Profile() string {
	return a.envProfile
}

func (a *app) SetProfile(profile string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.Config().SetProfile(profilePrefix + profile); err != nil {
		return err
	}

	a.envProfile = profile
	return nil
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
	return a.cfg.BoolDefault("server.ssl.enable", false)
}

func (a *app) IsLetsEncryptEnabled() bool {
	return a.cfg.BoolDefault("server.ssl.lets_encrypt.enable", false)
}

func (a *app) IsWebSocketEnabled() bool {
	return a.cfg.BoolDefault("server.websocket.enable", false)
}

func (a *app) NewChildLogger(fields log.Fields) log.Loggerer {
	return a.Log().WithFields(fields)
}

func (a *app) SetTLSConfig(tlsCfg *tls.Config) {
	a.mu.Lock()
	defer a.mu.Unlock()
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

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// app Unexported methods
//______________________________________________________________________________

func (a *app) logsDir() string {
	return filepath.Join(a.BaseDir(), "logs")
}

func (a *app) initPath() error {
	defer func() {
		err := a.VFS().AddMount(a.VirtualBaseDir(), a.BaseDir())
		if err != nil && err.(*os.PathError).Err == vfs.ErrMountExists {
			// Update app-base-dir to inferred base directory
			if m, err := a.VFS().FindMount(a.VirtualBaseDir()); err == nil {
				m.Proot = a.BaseDir()
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
			a.baseDir = filepath.Dir(ep)
		} else {
			a.baseDir = filepath.Dir(filepath.Dir(ep))
		}

		a.baseDir = filepath.Clean(a.baseDir)
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

		a.baseDir = filepath.Clean(a.ImportPath())
		a.physicalPathMode = true
		return nil
	}

	// Import path mode
	a.baseDir = filepath.Join(gopath, "src", filepath.FromSlash(a.ImportPath()))
	if !ess.IsFileExists(a.BaseDir()) {
		return fmt.Errorf("import path does not exists: %s", a.ImportPath())
	}

	return nil
}

func (a *app) initConfigValues() (err error) {
	cfg := a.Config()
	a.envProfile = cfg.StringDefault("env.active", defaultEnvProfile)
	if err = a.SetProfile(a.Profile()); err != nil {
		return err
	}

	readTimeout := cfg.StringDefault("server.timeout.read", "90s")
	writeTimeout := cfg.StringDefault("server.timeout.write", "90s")
	if !isValidTimeUnit(readTimeout, "s", "m") || !isValidTimeUnit(writeTimeout, "s", "m") {
		return errors.New("'server.timeout.{read|write}' value is not a valid time unit")
	}

	if a.httpReadTimeout, err = time.ParseDuration(readTimeout); err != nil {
		return fmt.Errorf("'server.timeout.read': %s", err)
	}

	if a.httpWriteTimeout, err = time.ParseDuration(writeTimeout); err != nil {
		return fmt.Errorf("'server.timeout.write': %s", err)
	}

	maxHdrBytesStr := cfg.StringDefault("server.max_header_bytes", "1mb")
	if maxHdrBytes, er := ess.StrToBytes(maxHdrBytesStr); er == nil {
		a.httpMaxHdrBytes = int(maxHdrBytes)
	} else {
		return errors.New("'server.max_header_bytes' value is not a valid size unit")
	}

	a.sslCert = cfg.StringDefault("server.ssl.cert", "")
	a.sslKey = cfg.StringDefault("server.ssl.key", "")
	if err = a.checkSSLConfigValues(); err != nil {
		return err
	}

	if err = a.initAutoCertManager(); err != nil {
		return err
	}

	if a.Type() != "websocket" {
		maxBodySizeStr := cfg.StringDefault("request.max_body_size", "5mb")
		if a.maxBodyBytes, err = ess.StrToBytes(maxBodySizeStr); err != nil {
			return errors.New("'request.max_body_size' value is not a valid size unit")
		}

		multipartMemoryStr := cfg.StringDefault("request.multipart_size", "32mb")
		if a.multipartMaxMemory, err = ess.StrToBytes(multipartMemoryStr); err != nil {
			return errors.New("'request.multipart_size' value is not a valid size unit")
		}

		a.serverHeader = cfg.StringDefault("server.header", "")
		a.serverHeaderEnabled = !ess.IsStrEmpty(a.serverHeader)
		a.requestIDEnabled = cfg.BoolDefault("request.id.enable", true)
		a.requestIDHeaderKey = cfg.StringDefault("request.id.header", ahttp.HeaderXRequestID)
		a.secureHeadersEnabled = cfg.BoolDefault("security.http_header.enable", true)
		a.gzipEnabled = cfg.BoolDefault("render.gzip.enable", true)
		a.accessLogEnabled = cfg.BoolDefault("server.access_log.enable", false)
		a.staticAccessLogEnabled = cfg.BoolDefault("server.access_log.static_file", true)
		a.dumpLogEnabled = cfg.BoolDefault("server.dump_log.enable", false)
		a.defaultContentType = resolveDefaultContentType(a.Config().StringDefault("render.default", ""))
		if a.defaultContentType == nil {
			return errors.New("'render.default' config value is not defined")
		}

		a.secureJSONPrefix = cfg.StringDefault("render.secure_json.prefix", defaultSecureJSONPrefix)

		ahttp.GzipLevel = cfg.IntDefault("render.gzip.level", 5)
		if !(ahttp.GzipLevel >= 1 && ahttp.GzipLevel <= 9) {
			return fmt.Errorf("'render.gzip.level' is not a valid level value: %v", ahttp.GzipLevel)
		}
	}

	a.shutdownGraceTimeStr = cfg.StringDefault("server.timeout.grace_shutdown", "60s")
	if !(strings.HasSuffix(a.shutdownGraceTimeStr, "s") || strings.HasSuffix(a.shutdownGraceTimeStr, "m")) {
		log.Warn("'server.timeout.grace_shutdown' value is not a valid time unit, assigning default value 60s")
		a.shutdownGraceTimeStr = "60s"
	}
	a.shutdownGraceTimeout, _ = time.ParseDuration(a.shutdownGraceTimeStr)

	return nil
}

func (a *app) checkSSLConfigValues() error {
	if a.IsSSLEnabled() {
		if !a.IsLetsEncryptEnabled() && (ess.IsStrEmpty(a.sslCert) || ess.IsStrEmpty(a.sslKey)) {
			return errors.New("SSL config is incomplete; either enable 'server.ssl.lets_encrypt.enable' or provide 'server.ssl.cert' & 'server.ssl.key' value")
		} else if !a.IsLetsEncryptEnabled() {
			if !ess.IsFileExists(a.sslCert) {
				return fmt.Errorf("SSL cert file not found: %s", a.sslCert)
			}

			if !ess.IsFileExists(a.sslKey) {
				return fmt.Errorf("SSL key file not found: %s", a.sslKey)
			}
		}
	}

	if a.IsLetsEncryptEnabled() && !a.IsSSLEnabled() {
		return errors.New("let's encrypt enabled, however SSL 'server.ssl.enable' is not enabled for application")
	}
	return nil
}

func (a *app) initAutoCertManager() error {
	if !a.IsSSLEnabled() || !a.IsLetsEncryptEnabled() {
		return nil
	}

	cfgKeyPrefix := "server.ssl.lets_encrypt"
	hostPolicy, found := a.cfg.StringList(cfgKeyPrefix + ".host_policy")
	if !found || len(hostPolicy) == 0 {
		return errors.New("'server.ssl.lets_encrypt.host_policy' is empty, provide at least one hostname")
	}

	renewBefore := time.Duration(a.cfg.IntDefault(cfgKeyPrefix+".renew_before", 10))

	a.autocertMgr = &autocert.Manager{
		Prompt:      autocert.AcceptTOS,
		HostPolicy:  autocert.HostWhitelist(hostPolicy...),
		RenewBefore: 24 * renewBefore * time.Hour,
		ForceRSA:    a.cfg.BoolDefault(cfgKeyPrefix+".force_rsa", false),
		Email:       a.cfg.StringDefault(cfgKeyPrefix+".email", ""),
	}

	if cacheDir := a.cfg.StringDefault(cfgKeyPrefix+".cache_dir", ""); !ess.IsStrEmpty(cacheDir) {
		a.autocertMgr.Cache = autocert.DirCache(cacheDir)
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
// App - Engines
//______________________________________________________________________________

// ServeHTTP method implementation of http.Handler interface.
func (a *app) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer a.aahRecover()
	if a.he.doRedirect(w, r) {
		return
	}

	if isWebSocket(r) {
		a.wse.Handle(w, r)
		return
	}

	a.he.Handle(w, r)
}
