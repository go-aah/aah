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
	"os/signal"
	"strings"
	"syscall"
	"time"

	"golang.org/x/crypto/acme/autocert"

	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
)

var (
	aahServer          *http.Server
	appTLSCfg          *tls.Config
	appAutocertManager *autocert.Manager
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Global methods
//___________________________________

// AddServerTLSConfig method can be used for custom TLS config for aah server.
// Note: if `server.ssl.lets_encrypt.enable=true` then framework sets the
// `GetCertificate` from autocert manager.
//
// Use `aah.OnInit` or `func init() {...}` to assign your custom TLSConfig.
func AddServerTLSConfig(tlsCfg *tls.Config) {
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
	log.Debugf("App i18n Locales: %v", strings.Join(AppI18n().Locales(), ", "))
	log.Debugf("App Route Domains: %v", strings.Join(AppRouter().DomainAddresses(), ", "))

	// Publish `OnStart` event
	AppEventStore().sortAndPublishSync(&Event{Name: EventOnStart})

	appEngine = newEngine(AppConfig())
	aahServer = &http.Server{
		Handler:        appEngine,
		ReadTimeout:    appHTTPReadTimeout,
		WriteTimeout:   appHTTPWriteTimeout,
		MaxHeaderBytes: appHTTPMaxHdrBytes,
	}

	aahServer.SetKeepAlivesEnabled(AppConfig().BoolDefault("server.keep_alive", true))

	go writePID(getBinaryFileName(), AppBaseDir())
	go listenSignals()

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
	if err := aahServer.Shutdown(ctx); err != nil && err != http.ErrServerClosed {
		log.Error(err)
	}

	// Publish `OnShutdown` event
	AppEventStore().sortAndPublishSync(&Event{Name: EventOnShutdown})

	// Exit normally
	cancel()
	log.Infof("'%v' application stopped", AppName())
	os.Exit(0)
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
	log.Infof("aah server running on %v", aahServer.Addr)
	if err := aahServer.Serve(listener); err != nil {
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

	log.Infof("aah server running on %v", aahServer.Addr)
	if err := aahServer.ListenAndServeTLS(appSSLCert, appSSLKey); err != nil {
		log.Error(err)
	}
}

func startHTTP() {
	log.Infof("aah server running on %v", aahServer.Addr)
	if err := aahServer.ListenAndServe(); err != nil {
		log.Error(err)
	}
}

// listenSignals method listens to OS signals for aah server Shutdown.
func listenSignals() {
	sc := make(chan os.Signal, 2)
	signal.Notify(sc, os.Interrupt, syscall.SIGTERM)
	go func() {
		switch <-sc {
		case os.Interrupt:
			log.Warn("Interrupt signal received")
		case syscall.SIGTERM:
			log.Warn("Termination signal received")
		}
		Shutdown()
	}()
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
