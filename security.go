// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"fmt"
	"net/http"

	"aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
	"aahframework.org/security.v0-unstable"
	"aahframework.org/security.v0-unstable/authc"
	"aahframework.org/security.v0-unstable/scheme"
	"aahframework.org/security.v0-unstable/session"
)

const (
	// KeyViewArgAuthcInfo key name is used to store `AuthenticationInfo` instance into `ViewArgs`.
	KeyViewArgAuthcInfo = "_aahAuthcInfo"

	// KeyViewArgSubject key name is used to store `Subject` instance into `ViewArgs`.
	KeyViewArgSubject = "_aahSubject"
)

var appSecurityManager = security.New()

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package methods
//___________________________________

// AppSecurityManager method returns the application security instance,
// which manages the Session, CORS, CSRF, Security Headers, etc.
func AppSecurityManager() *security.Manager {
	return appSecurityManager
}

// AppSessionManager method returns the application session manager.
// By default session is stateless.
func AppSessionManager() *session.Manager {
	return AppSecurityManager().SessionManager
}

// AddSessionStore method allows you to add custom session store which
// implements `session.Storer` interface. The `name` parameter is used in
// aah.conf on `session.store.type = "name"`.
func AddSessionStore(name string, store session.Storer) error {
	return session.AddStore(name, store)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Authentication and Authorization methods
//__________________________________________

func (e engine) handleAuthcAndAuthz(ctx *Context) flowResult {
	// If route auth is `anonymous` then continue the request flow
	// No authentication or authorization is required for that route.
	if ctx.route.Auth == "anonymous" {
		log.Debugf("Route auth is anonymous: %v", ctx.Req.Path)
		return flowCont
	}

	authScheme := AppSecurityManager().GetAuthScheme(ctx.route.Auth)
	if authScheme == nil {
		// If one or more auth schemes are defined in `security.auth_schemes { ... }`
		// and routes `auth` attribute is not defined then framework treats that route as `403 Forbidden`.
		if AppSecurityManager().IsAuthSchemesConfigured() {
			log.Warnf("Auth schemes are configured in security.conf, however attribute 'auth' or 'default_auth' is not defined in routes.conf, so treat it as 403 forbidden: %v", ctx.Req.Path)
			writeErrorInfo(ctx, http.StatusForbidden, "Forbidden")
			e.writeReply(ctx)
			return flowStop
		}

		// If auth scheme is not configured in security.conf then treat it as `anonymous`.
		log.Tracef("Route auth scheme is not configured, so treat it as anonymous: %v", ctx.Req.Path)
		return flowCont
	}

	log.Debugf("Route auth scheme: %s", authScheme.Scheme())
	switch authScheme.Scheme() {
	case "form":
		return e.doFormAuthcAndAuthz(authScheme, ctx)
	default:
		return e.doAuthcAndAuthz(authScheme, ctx)
	}
}

// doFormAuthcAndAuthz method does Form Authentication and Authorization.
func (e *engine) doFormAuthcAndAuthz(ascheme scheme.Schemer, ctx *Context) flowResult {
	formAuth := ascheme.(*scheme.FormAuth)

	// In Form authentication check session is already authentication if yes
	// then continue the request flow immediately.
	if ctx.Subject().IsAuthenticated() {
		if ctx.Session().IsKeyExists(KeyViewArgAuthcInfo) {
			ctx.Subject().AuthenticationInfo = ctx.Session().Get(KeyViewArgAuthcInfo).(*authc.AuthenticationInfo)
			ctx.Subject().AuthorizationInfo = formAuth.DoAuthorizationInfo(ctx.Subject().AuthenticationInfo)
		} else {
			log.Warn("It seems there is an issue with session data of AuthenticationInfo")
		}

		return flowCont
	}

	// Check route is login submit URL otherwise send it login URL.
	// Since session is not authenticated.
	if formAuth.LoginSubmitURL != ctx.route.Path && ctx.Req.Method != ahttp.MethodPost {
		loginURL := formAuth.LoginURL
		if formAuth.LoginURL != ctx.Req.Path {
			loginURL = fmt.Sprintf("%s?_rt=%s", loginURL, ctx.Req.Raw.RequestURI)
		}
		ctx.Reply().Redirect(loginURL)
		e.writeReply(ctx)
		return flowStop
	}

	publishOnPreAuthEvent(ctx)

	// Do Authentication
	authcInfo, err := formAuth.DoAuthenticate(formAuth.ExtractAuthenticationToken(ctx.Req))
	if err != nil || authcInfo == nil {
		log.Info("Authentication is failed, sending to login failure URL")

		redirectURL := formAuth.LoginFailureURL
		redirectTarget := ctx.Req.Raw.FormValue("_rt")
		if !ess.IsStrEmpty(redirectTarget) {
			redirectURL = redirectURL + "&_rt=" + redirectTarget
		}

		ctx.Reply().Redirect(redirectURL)
		e.writeReply(ctx)
		return flowStop
	}

	log.Info("Authentication successful")

	ctx.Subject().AuthenticationInfo = authcInfo
	ctx.Subject().AuthorizationInfo = formAuth.DoAuthorizationInfo(authcInfo)
	ctx.Session().IsAuthenticated = true

	// Remove the credential
	ctx.Subject().AuthenticationInfo.Credential = nil
	ctx.Session().Set(KeyViewArgAuthcInfo, ctx.Subject().AuthenticationInfo)

	publishOnPostAuthEvent(ctx)

	rt := ctx.Req.Raw.FormValue("_rt")
	if formAuth.IsAlwaysToDefaultTarget || ess.IsStrEmpty(rt) {
		ctx.Reply().Redirect(formAuth.DefaultTargetURL)
	} else {
		log.Debugf("Redirect to URL found: %v", rt)
		ctx.Reply().Redirect(rt)
	}

	e.writeReply(ctx)
	return flowStop
}

// doAuthcAndAuthz method does Authentication and Authorization.
func (e *engine) doAuthcAndAuthz(ascheme scheme.Schemer, ctx *Context) flowResult {
	publishOnPreAuthEvent(ctx)

	// Do Authentication
	authcInfo, err := ascheme.DoAuthenticate(ascheme.ExtractAuthenticationToken(ctx.Req))
	if err != nil || authcInfo == nil {
		log.Info("Authentication is failed")

		if ascheme.Scheme() == "basic" {
			basicAuth := ascheme.(*scheme.BasicAuth)
			ctx.Reply().Header(ahttp.HeaderWWWAuthenticate, `Basic realm="`+basicAuth.RealmName+`"`)
		}

		writeErrorInfo(ctx, http.StatusUnauthorized, "Unauthorized")
		e.writeReply(ctx)
		return flowStop
	}

	log.Info("Authentication successful")

	ctx.Subject().AuthenticationInfo = authcInfo
	ctx.Subject().AuthorizationInfo = ascheme.DoAuthorizationInfo(authcInfo)
	ctx.Session().IsAuthenticated = true

	// Remove the credential
	ctx.Subject().AuthenticationInfo.Credential = nil

	publishOnPostAuthEvent(ctx)

	return flowCont
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

func initSecurity(appCfg *config.Config) error {
	if err := appSecurityManager.Init(appCfg); err != nil {
		return err
	}

	// Based on aah server SSL configuration `http.Cookie.Secure` value is set, even
	// though it's true in aah.conf at `security.session.secure = true`.
	if AppSessionManager() != nil {
		AppSessionManager().Options.Secure = AppIsSSLEnabled()
	}

	return nil
}

func isFormAuthLoginRoute(ctx *Context) bool {
	authScheme := AppSecurityManager().GetAuthScheme(ctx.route.Auth)
	if authScheme != nil && authScheme.Scheme() == "form" {
		return authScheme.(*scheme.FormAuth).LoginSubmitURL == ctx.route.Path
	}
	return false
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Template methods
//___________________________________

// tmplSessionValue method returns session value for the given key. If session
// object unavailable this method returns nil.
func tmplSessionValue(viewArgs map[string]interface{}, key string) interface{} {
	if sub := getSubjectFromViewArgs(viewArgs); sub != nil {
		if sub.Session != nil {
			value := sub.Session.Get(key)
			return sanatizeValue(value)
		}
	}
	return nil
}

// tmplFlashValue method returns session value for the given key. If session
// object unavailable this method returns nil.
func tmplFlashValue(viewArgs map[string]interface{}, key string) interface{} {
	if sub := getSubjectFromViewArgs(viewArgs); sub != nil {
		if sub.Session != nil {
			return sanatizeValue(sub.Session.GetFlash(key))
		}
	}
	return nil
}

// tmplIsAuthenticated method returns the value of `Session.IsAuthenticated`.
func tmplIsAuthenticated(viewArgs map[string]interface{}) bool {
	if sub := getSubjectFromViewArgs(viewArgs); sub != nil {
		if sub.Session != nil {
			return sub.Session.IsAuthenticated
		}
	}
	return false
}

// tmplHasRole method returns the value of `Subject.HasRole`.
func tmplHasRole(viewArgs map[string]interface{}, role string) bool {
	if sub := getSubjectFromViewArgs(viewArgs); sub != nil {
		return sub.HasRole(role)
	}
	return false
}

// tmplHasAllRoles method returns the value of `Subject.HasAllRoles`.
func tmplHasAllRoles(viewArgs map[string]interface{}, roles ...string) bool {
	if sub := getSubjectFromViewArgs(viewArgs); sub != nil {
		return sub.HasAllRoles(roles...)
	}
	return false
}

// tmplHasAnyRole method returns the value of `Subject.HasAnyRole`.
func tmplHasAnyRole(viewArgs map[string]interface{}, roles ...string) bool {
	if sub := getSubjectFromViewArgs(viewArgs); sub != nil {
		return sub.HasAnyRole(roles...)
	}
	return false
}

// tmplIsPermitted method returns the value of `Subject.IsPermitted`.
func tmplIsPermitted(viewArgs map[string]interface{}, permission string) bool {
	if sub := getSubjectFromViewArgs(viewArgs); sub != nil {
		return sub.IsPermitted(permission)
	}
	return false
}

// tmplIsPermittedAll method returns the value of `Subject.IsPermittedAll`.
func tmplIsPermittedAll(viewArgs map[string]interface{}, permissions ...string) bool {
	if sub := getSubjectFromViewArgs(viewArgs); sub != nil {
		return sub.IsPermittedAll(permissions...)
	}
	return false
}

func getSubjectFromViewArgs(viewArgs map[string]interface{}) *security.Subject {
	if sv, found := viewArgs[KeyViewArgSubject]; found {
		return sv.(*security.Subject)
	}
	return nil
}
