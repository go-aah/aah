// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
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

	go writePID(AppName(), AppBaseDir(), AppConfig())
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

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

func startUnix(address string) {
	log.Infof("aah server running on %v", address)

	sockFile := address[5:]
	if err := os.Remove(sockFile); !os.IsNotExist(err) {
		logAsFatal(err)
	}

	listener, err := net.Listen("unix", sockFile)
	logAsFatal(err)

	aahServer.Addr = address
	if err := aahServer.Serve(listener); err != nil {
		log.Error(err)
	}
}

func startHTTPS() {
	log.Infof("Let's Encypyt cert enabled: %v", appIsLetsEncrypt)
	log.Infof("aah server running on %v", aahServer.Addr)

	// assign custom TLS config if provided
	if appTLSCfg != nil {
		aahServer.TLSConfig = appTLSCfg
	}

	// Add cert, if let's encrypt enabled
	if appIsLetsEncrypt {
		if appTLSCfg == nil {
			aahServer.TLSConfig = &tls.Config{GetCertificate: appAutocertManager.GetCertificate}
		} else {
			aahServer.TLSConfig.GetCertificate = appAutocertManager.GetCertificate
		}
	}

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
