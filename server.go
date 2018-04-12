// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"aahframework.org/essentials.v0"
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// app methods
//______________________________________________________________________________

func (a *app) Start() {
	defer a.aahRecover()

	if !a.initialized {
		a.Log().Fatal("aah application is not initialized, call `aah.Init` before the `aah.Start`.")
	}

	sessionMode := "stateless"
	if a.SessionManager().IsStateful() {
		sessionMode = "stateful"
	}

	a.Log().Infof("App Name: %s", a.Name())
	a.Log().Infof("App Version: %s", a.BuildInfo().Version)
	a.Log().Infof("App Build Date: %s", a.BuildInfo().Date)
	a.Log().Infof("App Profile: %s", a.Profile())
	a.Log().Infof("App TLS/SSL Enabled: %t", a.IsSSLEnabled())

	if a.viewMgr != nil {
		a.Log().Infof("App View Engine: %s", a.viewMgr.engineName)
	}

	a.Log().Infof("App Session Mode: %s", sessionMode)

	if a.webApp || a.viewMgr != nil {
		a.Log().Infof("App Anti-CSRF Protection Enabled: %t", a.SecurityManager().AntiCSRF.Enabled)
	}

	a.Log().Info("App Route Domains:")
	for _, name := range a.Router().DomainAddresses() {
		a.Log().Infof("      Host: %s, CORS Enabled: %t", name, a.Router().Domains[name].CORSEnabled)
	}

	if a.I18n() != nil {
		a.Log().Infof("App i18n Locales: %s", strings.Join(a.I18n().Locales(), ", "))
	}

	if a.Log().IsLevelDebug() {
		for event := range a.EventStore().subscribers {
			for _, c := range a.EventStore().subscribers[event] {
				a.Log().Debugf("Callback: %s, subscribed to event: %s", funcName(c.Callback), event)
			}
		}
	}

	a.Log().Infof("App Shutdown Grace Timeout: %s", a.shutdownGraceTimeStr)

	// Publish `OnStart` event
	a.EventStore().sortAndPublishSync(&Event{Name: EventOnStart})

	hl := a.Log().ToGoLogger()
	hl.SetOutput(ioutil.Discard)

	a.server = &http.Server{
		Handler:        a.engine,
		ReadTimeout:    a.httpReadTimeout,
		WriteTimeout:   a.httpWriteTimeout,
		MaxHeaderBytes: a.httpMaxHdrBytes,
		ErrorLog:       hl,
	}

	a.server.SetKeepAlivesEnabled(a.Config().BoolDefault("server.keep_alive", true))
	a.writePID()

	go a.listenForHotConfigReload()

	// Unix Socket
	if strings.HasPrefix(a.HTTPAddress(), "unix") {
		a.startUnix()
		return
	}

	a.server.Addr = fmt.Sprintf("%s:%s", a.HTTPAddress(), a.HTTPPort())

	// HTTPS
	if a.IsSSLEnabled() {
		a.startHTTPS()
		return
	}

	// HTTP
	a.startHTTP()
}

func (a *app) Shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), a.shutdownGraceTimeout)
	defer cancel()

	a.Log().Warn("aah go server graceful shutdown triggered with timeout of ", a.shutdownGraceTimeStr)
	if err := a.server.Shutdown(ctx); err != nil && err != http.ErrServerClosed {
		a.Log().Error(err)
	}

	a.shutdownRedirectServer()

	// Publish `OnShutdown` event
	a.EventStore().sortAndPublishSync(&Event{Name: EventOnShutdown})
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// app Unexported methods
//______________________________________________________________________________

func (a *app) writePID() {
	// Get the application PID
	a.pid = os.Getpid()

	pidFile := a.Config().StringDefault("pid_file", "")
	if ess.IsStrEmpty(pidFile) {
		pidFile = filepath.Join(a.BaseDir(), a.binaryFilename())
	}

	if !strings.HasSuffix(pidFile, ".pid") {
		pidFile += ".pid"
	}

	if err := ioutil.WriteFile(pidFile, []byte(strconv.Itoa(a.pid)), 0644); err != nil {
		a.Log().Error(err)
	}
}

func (a *app) startUnix() {
	sockFile := a.HTTPAddress()[5:]
	if err := os.Remove(sockFile); !os.IsNotExist(err) {
		a.Log().Fatal(err)
	}

	listener, err := net.Listen("unix", sockFile)
	if err != nil {
		a.Log().Fatal(err)
		return
	}

	a.server.Addr = a.HTTPAddress()
	a.Log().Infof("aah go server running on %v", a.server.Addr)
	if err := a.server.Serve(listener); err != nil && err != http.ErrServerClosed {
		a.Log().Error(err)
	}
}

func (a *app) startHTTPS() {
	// Assign user-defined TLS config if provided
	if a.tlsCfg == nil {
		a.server.TLSConfig = new(tls.Config)
	} else {
		a.Log().Info("Adding user provided TLS Config")
		a.server.TLSConfig = a.tlsCfg
	}

	// Add cert, if let's encrypt enabled
	if a.IsLetsEncrypt() {
		a.Log().Infof("Let's Encypyt CA Cert enabled")
		a.server.TLSConfig.GetCertificate = a.autocertMgr.GetCertificate
	} else {
		a.Log().Infof("SSLCert: %s, SSLKey: %s", a.sslCert, a.sslKey)
	}

	// Enable & Disable HTTP/2
	if a.Config().BoolDefault("server.ssl.disable_http2", false) {
		// To disable HTTP/2 is-
		//  - Don't add "h2" to TLSConfig.NextProtos
		//  - Initialize TLSNextProto with empty map
		// Otherwise Go will enable HTTP/2 by default. It's not gonna listen to you :)
		a.server.TLSNextProto = map[string]func(*http.Server, *tls.Conn, http.Handler){}
	} else {
		a.server.TLSConfig.NextProtos = append(a.server.TLSConfig.NextProtos, "h2")
	}

	// start HTTP redirect server if enabled
	go a.startHTTPRedirect()

	a.printStartupNote()
	if err := a.server.ListenAndServeTLS(a.sslCert, a.sslKey); err != nil && err != http.ErrServerClosed {
		a.Log().Error(err)
	}
}

func (a *app) startHTTP() {
	a.printStartupNote()
	if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		a.Log().Error(err)
	}
}

func (a *app) startHTTPRedirect() {
	cfg := a.Config()
	keyPrefix := "server.ssl.redirect_http"
	if !cfg.BoolDefault(keyPrefix+".enable", false) {
		return
	}

	address := a.HTTPAddress()
	toPort := a.parsePort(cfg.StringDefault("server.port", defaultHTTPPort))
	fromPort, found := cfg.String(keyPrefix + ".port")
	if !found {
		a.Log().Errorf("'%s.port' is required value, unable to start redirect server", keyPrefix)
		return
	}
	redirectCode := cfg.IntDefault(keyPrefix+".code", http.StatusTemporaryRedirect)

	a.Log().Infof("aah go redirect server running on %s:%s", address, fromPort)
	a.redirectServer = &http.Server{
		Addr: address + ":" + fromPort,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			target := "https://" + parseHost(r.Host, toPort) + r.URL.Path
			if len(r.URL.RawQuery) > 0 {
				target += "?" + r.URL.RawQuery
			}
			http.Redirect(w, r, target, redirectCode)
		}),
	}

	if err := a.redirectServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		a.Log().Error(err)
	}
}

func (a *app) shutdownRedirectServer() {
	if a.redirectServer != nil {
		_ = a.redirectServer.Close()
	}
}

func (a *app) printStartupNote() {
	port := firstNonZeroString(
		a.Config().StringDefault("server.port", defaultHTTPPort),
		a.Config().StringDefault("server.proxyport", ""))
	a.Log().Infof("aah go server running on %s:%s", a.HTTPAddress(), a.parsePort(port))
}
