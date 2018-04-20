// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"crypto/tls"
	"html/template"

	"aahframework.org/ainsp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/i18n.v0"
	"aahframework.org/log.v0"
	"aahframework.org/router.v0"
	"aahframework.org/security.v0"
	"aahframework.org/security.v0/session"
	"aahframework.org/view.v0"
)

var defaultApp = newApp()

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// App Info methods
//______________________________________________________________________________

// AppName method returns aah application name from app config `name` otherwise
// app name of the base directory.
func AppName() string {
	return defaultApp.Name()
}

// AppInstanceName method returns aah application instane name from app config
// `instance_name` otherwise empty string.
func AppInstanceName() string {
	return defaultApp.InstanceName()
}

// AppDesc method returns aah application friendly description from app config
// otherwise empty string.
func AppDesc() string {
	return defaultApp.Desc()
}

// AppProfile returns aah application configuration profile name
// For e.g.: dev, prod, etc. Default is `dev`
func AppProfile() string {
	return defaultApp.Profile()
}

// AppBaseDir method returns the application base or binary current directory
// 	For e.g.:
// 		$GOPATH/src/github.com/user/myproject
// 		<app/binary/path/base/directory>
func AppBaseDir() string {
	return defaultApp.BaseDir()
}

// AppImportPath method returns the application Go import path.
func AppImportPath() string {
	return defaultApp.ImportPath()
}

// AppHTTPAddress method returns aah application HTTP address otherwise empty string
func AppHTTPAddress() string {
	return defaultApp.HTTPAddress()
}

// AppHTTPPort method returns aah application HTTP port number based on `server.port`
// value. Possible outcomes are user-defined port, `80`, `443` and `8080`.
func AppHTTPPort() string {
	return defaultApp.HTTPPort()
}

// AppBuildInfo method return user application version no.
func AppBuildInfo() *BuildInfo {
	return defaultApp.BuildInfo()
}

// AllAppProfiles method returns all the aah application environment profile names.
func AllAppProfiles() []string {
	return defaultApp.AllProfiles()
}

// AppIsSSLEnabled method returns true if aah application is enabled with SSL
// otherwise false.
func AppIsSSLEnabled() bool {
	return defaultApp.IsSSLEnabled()
}

// AppSSLCert method returns SSL cert filpath if its configured in aah.conf
// otherwise empty string.
func AppSSLCert() string {
	return defaultApp.sslCert
}

// AppSSLKey method returns SSL key filepath if its configured in aah.conf
// otherwise empty string.
func AppSSLKey() string {
	return defaultApp.sslKey
}

// SetAppProfile method sets given profile as current aah application profile.
//		For Example:
//
//		aah.SetAppProfile("prod")
func SetAppProfile(profile string) error {
	return defaultApp.SetProfile(profile)
}

// SetAppBuildInfo method sets the user application build info into aah instance.
func SetAppBuildInfo(bi *BuildInfo) {
	defaultApp.SetBuildInfo(bi)
}

// SetAppPackaged method sets the info of binary is packaged or not.
func SetAppPackaged(pack bool) {
	defaultApp.SetPackaged(pack)
}

// NewChildLogger method create a child logger from aah application default logger.
func NewChildLogger(fields log.Fields) log.Loggerer {
	return defaultApp.NewChildLogger(fields)
}

// Init method initializes `aah` application, if anything goes wrong during
// initialize process, it will log it as fatal msg and exit.
func Init(importPath string) error {
	return defaultApp.Init(importPath)
}

// AppLog method return the aah application logger instance.
func AppLog() log.Loggerer {
	return defaultApp.Log()
}

// AppDefaultI18nLang method returns aah application i18n default language if
// configured other framework defaults to "en".
func AppDefaultI18nLang() string {
	return defaultApp.DefaultI18nLang()
}

// AppI18n method returns aah application I18n store instance.
func AppI18n() *i18n.I18n {
	return defaultApp.I18n()
}

// AppI18nLocales returns all the loaded locales from i18n store
func AppI18nLocales() []string {
	if defaultApp.I18n() == nil {
		return []string{}
	}
	return defaultApp.I18n().Locales()
}

// AddServerTLSConfig method can be used for custom TLS config for aah server.
//
// DEPRECATED: Use method `aah.SetTLSConfig` instead. Planned to be
// removed in `v1.0.0` release.
func AddServerTLSConfig(tlsCfg *tls.Config) {
	// DEPRECATED, planned to be removed in v1.0
	defaultApp.Log().Warn("DEPRECATED: Method 'AddServerTLSConfig' deprecated in v0.9, use method 'SetTLSConfig' instead. Deprecated method will not break your functionality, its good to update to new method.")

	SetTLSConfig(tlsCfg)
}

// SetTLSConfig method is used to set custom TLS config for aah server.
// Note: if `server.ssl.lets_encrypt.enable=true` then framework sets the
// `GetCertificate` from autocert manager.
//
// Use `aah.OnInit` or `func init() {...}` to assign your custom TLS Config.
func SetTLSConfig(tlsCfg *tls.Config) {
	defaultApp.SetTLSConfig(tlsCfg)
}

// AppIsWebSocketEnabled method returns true if WebSocket enabled otherwise false.
// func AppIsWebSocketEnabled() bool {
// 	return defaultApp.IsWebSocketEnabled()
// }

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// App module instance methods
//______________________________________________________________________________

// AppConfig method returns aah application configuration instance.
func AppConfig() *config.Config {
	return defaultApp.Config()
}

// AppRouter method returns aah application router instance.
func AppRouter() *router.Router {
	return defaultApp.Router()
}

// AppViewEngine method returns aah application view Engine instance.
func AppViewEngine() view.Enginer {
	return defaultApp.ViewEngine()
}

// AppSecurityManager method returns the application security instance,
// which manages the Session, CORS, CSRF, Security Headers, etc.
func AppSecurityManager() *security.Manager {
	return defaultApp.SecurityManager()
}

// AppSessionManager method returns the application session manager.
// By default session is stateless.
func AppSessionManager() *session.Manager {
	return defaultApp.SessionManager()
}

// AppEventStore method returns aah application event store.
func AppEventStore() *EventStore {
	return defaultApp.EventStore()
}

// AddController method adds given controller into controller registory.
func AddController(c interface{}, methods []*ainsp.Method) {
	defaultApp.AddController(c, methods)
}

// AddWebSocket method adds given WebSocket into WebSocket registry.
func AddWebSocket(w interface{}, methods []*ainsp.Method) {
	defaultApp.AddWebSocket(w, methods)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// App Start and Shutdown methods
//______________________________________________________________________________

// Start method starts the Go HTTP server based on aah config "server.*".
func Start() {
	defaultApp.Start()
}

// Shutdown method allows aah server to shutdown gracefully with given timeout
// in seconds. It's invoked on OS signal `SIGINT` and `SIGTERM`.
//
// Method performs:
//    - Graceful server shutdown with timeout by `server.timeout.grace_shutdown`
//    - Publishes `OnShutdown` event
//    - Exits program with code 0
func Shutdown() {
	defaultApp.Shutdown()
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// App Error and middlewares
//______________________________________________________________________________

// SetErrorHandler method is used to register custom centralized application
// error handler. If custom handler is not then default error handler takes place.
func SetErrorHandler(handlerFunc ErrorHandlerFunc) {
	defaultApp.errorMgr.SetHandler(handlerFunc)
}

// Middlewares method adds given middleware into middleware stack
func Middlewares(middlewares ...MiddlewareFunc) {
	defaultApp.he.Middlewares(middlewares...)
}

// AddLoggerHook method adds given logger into aah application default logger.
func AddLoggerHook(name string, hook log.HookFunc) error {
	return defaultApp.AddLoggerHook(name, hook)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// App View methods
//______________________________________________________________________________

// AddTemplateFunc method adds template func map into view engine.
func AddTemplateFunc(funcs template.FuncMap) {
	defaultApp.AddTemplateFunc(funcs)
}

// AddViewEngine method adds the given name and view engine to view store.
func AddViewEngine(name string, engine view.Enginer) error {
	return defaultApp.AddViewEngine(name, engine)
}

// SetMinifier method sets the given minifier func into aah framework.
// Note: currently minifier is called only for HTML contentType.
func SetMinifier(fn MinifierFunc) {
	defaultApp.SetMinifier(fn)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// app event methods
//______________________________________________________________________________

// OnInit method is to subscribe to aah application `OnInit` event. `OnInit`
// event published right after the aah application configuration `aah.conf`
// initialized.
func OnInit(ecb EventCallbackFunc, priority ...int) {
	defaultApp.OnInit(ecb, priority...)
}

// OnStart method is to subscribe to aah application `OnStart` event. `OnStart`
// event pubished right before the aah server listen and serving request.
func OnStart(ecb EventCallbackFunc, priority ...int) {
	defaultApp.OnStart(ecb, priority...)
}

// OnShutdown method is to subscribe to aah application `OnShutdown` event.
// `OnShutdown` event pubished right before the aah server is stopped Listening
// and serving request.
func OnShutdown(ecb EventCallbackFunc, priority ...int) {
	defaultApp.OnShutdown(ecb, priority...)
}

// OnRequest method is to subscribe to aah server `OnRequest` extension point.
// `OnRequest` called for every incoming request.
//
// The `aah.Context` object passed to the extension functions is decorated with
// the `ctx.SetURL()` and `ctx.SetMethod()` methods. Calls to these methods will
// impact how the request is routed and can be used for rewrite rules.
//
// Route is not yet populated/evaluated at this point.
func OnRequest(sef EventCallbackFunc) {
	defaultApp.OnRequest(sef)
}

// OnPreReply method is to subscribe to aah server `OnPreReply` extension point.
// `OnPreReply` called for every reply from aah server.
//
// Except when
//   1) `Reply().Done()`,
//   2) `Reply().Redirect(...)` is called.
// Refer `aah.Reply.Done()` godoc for more info.
func OnPreReply(sef EventCallbackFunc) {
	defaultApp.OnPreReply(sef)
}

// OnAfterReply method is to subscribe to aah server `OnAfterReply` extension
// point. `OnAfterReply` called for every reply from aah server.
//
// Except when
//   1) `Reply().Done()`,
//   2) `Reply().Redirect(...)` is called.
// Refer `aah.Reply.Done()` godoc for more info.
func OnAfterReply(sef EventCallbackFunc) {
	defaultApp.OnAfterReply(sef)
}

// OnPreAuth method is to subscribe to aah application `OnPreAuth` event.
// `OnPreAuth` event pubished right before the aah server is authenticates &
// authorizes an incoming request.
func OnPreAuth(sef EventCallbackFunc) {
	defaultApp.OnPreAuth(sef)
}

// OnPostAuth method is to subscribe to aah application `OnPreAuth` event.
// `OnPostAuth` event pubished right after the aah server is authenticates &
// authorizes an incoming request.
func OnPostAuth(sef EventCallbackFunc) {
	defaultApp.OnPostAuth(sef)
}

// PublishEvent method publishes events to subscribed callbacks asynchronously.
// It means each subscribed callback executed via goroutine.
func PublishEvent(eventName string, data interface{}) {
	defaultApp.PublishEvent(eventName, data)
}

// PublishEventSync method publishes events to subscribed callbacks
// synchronously.
func PublishEventSync(eventName string, data interface{}) {
	defaultApp.PublishEventSync(eventName, data)
}

// SubscribeEvent method is to subscribe to new or existing event.
func SubscribeEvent(eventName string, ec EventCallback) {
	defaultApp.SubscribeEvent(eventName, ec)
}

// SubscribeEventFunc method is to subscribe to new or existing event
// by `EventCallbackFunc`.
func SubscribeEventFunc(eventName string, ecf EventCallbackFunc) {
	defaultApp.SubscribeEventFunc(eventName, ecf)
}

// SubscribeEventf method is to subscribe to new or existing event
// by `EventCallbackFunc`.
//
// DEPRECATED: use `SubscribeEventFunc` instead. Planned to be removed in
// `v1.0.0` release.
func SubscribeEventf(eventName string, ecf EventCallbackFunc) {
	defaultApp.SubscribeEventf(eventName, ecf)
}

// UnsubscribeEvent method is to unsubscribe by event name and `EventCallback`
// from app event store.
func UnsubscribeEvent(eventName string, ec EventCallback) {
	defaultApp.UnsubscribeEvent(eventName, ec)
}

// UnsubscribeEventFunc method is to unsubscribe by event name and
// `EventCallbackFunc` from app event store.
func UnsubscribeEventFunc(eventName string, ecf EventCallbackFunc) {
	defaultApp.UnsubscribeEventFunc(eventName, ecf)
}

// UnsubscribeEventf method is to unsubscribe by event name and
// `EventCallbackFunc` from app event store.
//
// DEPRECATED: use `UnsubscribeEventFunc` instead. Planned to be removed in
// `v1.0.0` release.
func UnsubscribeEventf(eventName string, ecf EventCallbackFunc) {
	defaultApp.UnsubscribeEventf(eventName, ecf)
}
