// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/acme/autocert"

	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
)

var (
	aahServer          *http.Server
	appEngine          *engine
	appTLSCfg          *tls.Config
	appAutocertManager *autocert.Manager
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package methods
//___________________________________

// AddServerTLSConfig method can be used for custom TLS config for aah server.
//
// DEPRECATED: Use method `aah.SetTLSConfig` instead. Planned to be removed in `v1.0` release.
func AddServerTLSConfig(tlsCfg *tls.Config) {
	// DEPRECATED, planned to be removed in v1.0
	log.Warn("DEPRECATED: Method 'AddServerTLSConfig' deprecated in v0.9, use method 'SetTLSConfig' instead. Deprecated method will not break your functionality, its good to update to new method.")

	SetTLSConfig(tlsCfg)
}

// SetTLSConfig method is used to set custom TLS config for aah server.
// Note: if `server.ssl.lets_encrypt.enable=true` then framework sets the
// `GetCertificate` from autocert manager.
//
// Use `aah.OnInit` or `func init() {...}` to assign your custom TLS Config.
func SetTLSConfig(tlsCfg *tls.Config) {
	appTLSCfg = tlsCfg
}

// Start method starts the Go HTTP server based on aah config "server.*".
func Start() {
	defer aahRecover()

	if !appInitialized {
		log.Fatal("aah application is not initialized, call `aah.Init` before the `aah.Start`.")
	}

	sessionMode := "stateless"
	if AppSessionManager().IsStateful() {
		sessionMode = "stateful"
	}

	log.Infof("App Name: %v", AppName())
	log.Infof("App Version: %v", AppBuildInfo().Version)
	log.Infof("App Build Date: %v", AppBuildInfo().Date)
	log.Infof("App Profile: %v", AppProfile())
	log.Infof("App TLS/SSL Enabled: %v", AppIsSSLEnabled())
	log.Infof("App Session Mode: %v", sessionMode)
	log.Infof("App Anti-CSRF Protection Enabled: %v", AppSecurityManager().AntiCSRF.Enabled)

	if log.IsLevelDebug() {
		log.Debugf("App Route Domains: %v", strings.Join(AppRouter().DomainAddresses(), ", "))
		if AppI18n() != nil {
			log.Debugf("App i18n Locales: %v", strings.Join(AppI18n().Locales(), ", "))
		}

		for event := range AppEventStore().subscribers {
			for _, c := range AppEventStore().subscribers[event] {
				log.Debugf("Callback: %s, subscribed to event: %s", funcName(c.Callback), event)
			}
		}
	}

	// Publish `OnStart` event
	AppEventStore().sortAndPublishSync(&Event{Name: EventOnStart})

	appEngine = newEngine(AppConfig())
	aahServer = &http.Server{
		Handler:        appEngine,
		ReadTimeout:    appHTTPReadTimeout,
		WriteTimeout:   appHTTPWriteTimeout,
		MaxHeaderBytes: appHTTPMaxHdrBytes,
		ErrorLog:       log.ToGoLogger(),
	}

	aahServer.SetKeepAlivesEnabled(AppConfig().BoolDefault("server.keep_alive", true))

	go writePID(AppConfig(), getBinaryFileName(), AppBaseDir())

	// Unix Socket
	if strings.HasPrefix(AppHTTPAddress(), "unix") {
		startUnix(AppHTTPAddress())
		return
	}

	aahServer.Addr = fmt.Sprintf("%s:%s", AppHTTPAddress(), AppHTTPPort())

	// HTTPS
	if AppIsSSLEnabled() {
		startHTTPS()
		return
	}

	// HTTP
	startHTTP()
}

// Shutdown method allows aah server to shutdown gracefully with given timeoout
// in seconds. It's invoked on OS signal `SIGINT` and `SIGTERM`.
//
// Method performs:
//    - Graceful server shutdown with timeout by `server.timeout.grace_shutdown`
//    - Publishes `OnShutdown` event
//    - Exits program with code 0
func Shutdown() {
	graceTime := AppConfig().StringDefault("server.timeout.grace_shutdown", "60s")
	if !(strings.HasSuffix(graceTime, "s") || strings.HasSuffix(graceTime, "m")) {
		log.Warn("'server.timeout.grace_shutdown' value is not a valid time unit, assigning default")
		graceTime = "60s"
	}

	graceTimeout, _ := time.ParseDuration(graceTime)
	ctx, cancel := context.WithTimeout(context.Background(), graceTimeout)
	defer cancel()

	log.Trace("aah go server shutdown with timeout: ", graceTime)
	if err := aahServer.Shutdown(ctx); err != nil && err != http.ErrServerClosed {
		log.Error(err)
	}

	// Publish `OnShutdown` event
	AppEventStore().sortAndPublishSync(&Event{Name: EventOnShutdown})
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

func startUnix(address string) {
	sockFile := address[5:]
	if err := os.Remove(sockFile); !os.IsNotExist(err) {
		logAsFatal(err)
	}

	listener, err := net.Listen("unix", sockFile)
	logAsFatal(err)

	aahServer.Addr = address
	log.Infof("aah go server running on %v", aahServer.Addr)
	if err := aahServer.Serve(listener); err != nil && err != http.ErrServerClosed {
		log.Error(err)
	}
}

func startHTTPS() {
	// Assign user-defined TLS config if provided
	if appTLSCfg == nil {
		aahServer.TLSConfig = new(tls.Config)
	} else {
		log.Info("Adding user provided TLS Config")
		aahServer.TLSConfig = appTLSCfg
	}

	// Add cert, if let's encrypt enabled
	if appIsLetsEncrypt {
		log.Infof("Let's Encypyt CA Cert enabled")
		aahServer.TLSConfig.GetCertificate = appAutocertManager.GetCertificate
	} else {
		log.Infof("SSLCert: %v, SSLKey: %v", appSSLCert, appSSLKey)
	}

	// Enable & Disable HTTP/2
	if AppConfig().BoolDefault("server.ssl.disable_http2", false) {
		// To disable HTTP/2 is-
		//  - Don't add "h2" to TLSConfig.NextProtos
		//  - Initialize TLSNextProto with empty map
		// Otherwise Go will enable HTTP/2 by default. It's not gonna listen to you :)
		aahServer.TLSNextProto = map[string]func(*http.Server, *tls.Conn, http.Handler){}
	} else {
		aahServer.TLSConfig.NextProtos = append(aahServer.TLSConfig.NextProtos, "h2")
	}

	// start HTTP redirect server if enabled
	go startHTTPRedirect(AppConfig())

	printStartupNote()
	if err := aahServer.ListenAndServeTLS(appSSLCert, appSSLKey); err != nil && err != http.ErrServerClosed {
		log.Error(err)
	}
}

func startHTTP() {
	printStartupNote()
	if err := aahServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error(err)
	}
}

func startHTTPRedirect(cfg *config.Config) {
	keyPrefix := "server.ssl.redirect_http"
	if !cfg.BoolDefault(keyPrefix+".enable", false) {
		return
	}

	address := cfg.StringDefault("server.address", "")
	toPort := parsePort(cfg.StringDefault("server.port", appDefaultHTTPPort))
	fromPort, found := cfg.String(keyPrefix + ".port")
	if !found {
		log.Errorf("'%s.port' is required value, unable to start redirect server", keyPrefix)
		return
	}
	redirectCode := cfg.IntDefault(keyPrefix+".code", http.StatusTemporaryRedirect)

	log.Infof("aah go redirect server running on %s:%s", address, fromPort)
	if err := http.ListenAndServe(address+":"+fromPort, http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			target := "https://" + parseHost(r.Host, toPort) + r.URL.Path
			if len(r.URL.RawQuery) > 0 {
				target += "?" + r.URL.RawQuery
			}
			http.Redirect(w, r, target, redirectCode)
		})); err != nil && err != http.ErrServerClosed {
		log.Error(err)
	}
}

func initAutoCertManager(cfg *config.Config) error {
	if !AppIsSSLEnabled() || !appIsLetsEncrypt {
		return nil
	}

	hostPolicy, found := cfg.StringList("server.ssl.lets_encrypt.host_policy")
	if !found || len(hostPolicy) == 0 {
		return errors.New("'server.ssl.lets_encrypt.host_policy' is empty, provide at least one hostname")
	}

	renewBefore := time.Duration(cfg.IntDefault("server.ssl.lets_encrypt.renew_before", 10))

	appAutocertManager = &autocert.Manager{
		Prompt:      autocert.AcceptTOS,
		HostPolicy:  autocert.HostWhitelist(hostPolicy...),
		RenewBefore: 24 * renewBefore * time.Hour,
		ForceRSA:    cfg.BoolDefault("server.ssl.lets_encrypt.force_rsa", false),
		Email:       cfg.StringDefault("server.ssl.lets_encrypt.email", ""),
	}

	cacheDir := cfg.StringDefault("server.ssl.lets_encrypt.cache_dir", "")
	if !ess.IsStrEmpty(cacheDir) {
		appAutocertManager.Cache = autocert.DirCache(cacheDir)
	}

	return nil
}

func printStartupNote() {
	port := firstNonZeroString(AppConfig().StringDefault("server.port", appDefaultHTTPPort), AppConfig().StringDefault("server.proxyport", ""))
	log.Infof("aah go server running on %s:%s", AppHTTPAddress(), parsePort(port))
}
