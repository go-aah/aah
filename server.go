// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"aahframework.org/log.v0"
)

var aahServer *http.Server

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

	address := AppHTTPAddress()
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
	if strings.HasPrefix(address, "unix") {
		startUnix(address)
		return
	}

	aahServer.Addr = fmt.Sprintf("%s:%s", AppHTTPAddress(), AppHTTPPort())
	log.Infof("aah server running on %v", aahServer.Addr)

	// HTTPS
	if AppIsSSLEnabled() {
		startHTTPS()
		return
	}

	// HTTP
	startHTTP()
}

func startUnix(address string) {
	log.Infof("Listening and serving HTTP on %v", address)

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
	if err := aahServer.ListenAndServeTLS(appSSLCert, appSSLKey); err != nil {
		log.Error(err)
	}
}

func startHTTP() {
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
