// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"fmt"
	"net/http"
	"net/url"

	"aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0-unstable"
	"aahframework.org/security.v0-unstable"
	"aahframework.org/security.v0-unstable/acrypto"
	"aahframework.org/security.v0-unstable/anticsrf"
	"aahframework.org/security.v0-unstable/authc"
	"aahframework.org/security.v0-unstable/scheme"
	"aahframework.org/security.v0-unstable/session"
)

const (
	// KeyViewArgAuthcInfo key name is used to store `AuthenticationInfo` instance into `ViewArgs`.
	KeyViewArgAuthcInfo = "_aahAuthcInfo"

	// KeyViewArgSubject key name is used to store `Subject` instance into `ViewArgs`.
	KeyViewArgSubject = "_aahSubject"

	keyAntiCSRFSecret = "_AntiCSRFSecret"
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

// AddPasswordAlgorithm method adds given password algorithm to encoders list.
// Implementation have to implement interface `PasswordEncoder`.
//
/// Then you can use it `security.auth_schemes.*`.
func AddPasswordAlgorithm(name string, encoder acrypto.PasswordEncoder) error {
	return acrypto.AddPasswordAlgorithm(name, encoder)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Authentication and Authorization Middleware
//_____________________________________________

func authcAndAuthzMiddleware(ctx *Context, m *Middleware) {
	// If route auth is `anonymous` then continue the request flow
	// No authentication or authorization is required for that route.
	if ctx.route.Auth == "anonymous" {
		ctx.Log().Debugf("Route auth is anonymous: %v", ctx.Req.Path)
		m.Next(ctx)
		return
	}

	authScheme := AppSecurityManager().GetAuthScheme(ctx.route.Auth)
	if authScheme == nil {
		// If one or more auth schemes are defined in `security.auth_schemes { ... }`
		// and routes `auth` attribute is not defined then framework treats that route as `403 Forbidden`.
		if AppSecurityManager().IsAuthSchemesConfigured() {
			ctx.Log().Warnf("Auth schemes are configured in security.conf, however attribute 'auth' or 'default_auth' is not defined in routes.conf, so treat it as 403 forbidden: %v", ctx.Req.Path)
			ctx.Reply().Error(&Error{
				Reason:  ErrAccessDenied,
				Code:    http.StatusForbidden,
				Message: http.StatusText(http.StatusForbidden),
			})
			return
		}

		// If auth scheme is not configured in security.conf then treat it as `anonymous`.
		ctx.Log().Tracef("Route auth scheme is not configured, so treat it as anonymous: %v", ctx.Req.Path)
		m.Next(ctx)
		return
	}

	ctx.Log().Debugf("Route auth scheme: %s", authScheme.Scheme())
	var result flowResult
	switch authScheme.Scheme() {
	case "form":
		result = doFormAuthcAndAuthz(authScheme, ctx)
	default:
		result = doAuthcAndAuthz(authScheme, ctx)
	}

	if result == flowCont {
		m.Next(ctx)
	}
}

// doFormAuthcAndAuthz method does Form Authentication and Authorization.
func doFormAuthcAndAuthz(ascheme scheme.Schemer, ctx *Context) flowResult {
	formAuth := ascheme.(*scheme.FormAuth)

	// In Form authentication check session is already authentication if yes
	// then continue the request flow immediately.
	if ctx.Subject().IsAuthenticated() {
		if ctx.Session().IsKeyExists(KeyViewArgAuthcInfo) {
			ctx.Subject().AuthenticationInfo = ctx.Session().Get(KeyViewArgAuthcInfo).(*authc.AuthenticationInfo)
			ctx.Subject().AuthorizationInfo = formAuth.DoAuthorizationInfo(ctx.Subject().AuthenticationInfo)
		} else {
			ctx.Log().Warn("It seems there is an issue with session data - AuthenticationInfo")
		}

		return flowCont
	}

	// Check route is login submit URL otherwise send it login URL.
	// Since session is not authenticated.
	if formAuth.LoginSubmitURL != ctx.route.Path && ctx.Req.Method != ahttp.MethodPost {
		loginURL := formAuth.LoginURL
		if formAuth.LoginURL != ctx.Req.Path {
			loginURL = fmt.Sprintf("%s?_rt=%s", loginURL, ctx.Req.Unwrap().RequestURI)
		}
		ctx.Reply().Redirect(loginURL)
		return flowStop
	}

	publishOnPreAuthEvent(ctx)

	// Do Authentication
	authcInfo, err := formAuth.DoAuthenticate(formAuth.ExtractAuthenticationToken(ctx.Req))
	if err != nil || authcInfo == nil {
		ctx.Log().Info("Authentication is failed, sending to login failure URL")

		redirectURL := formAuth.LoginFailureURL
		redirectTarget := ctx.Req.Unwrap().FormValue("_rt")
		if !ess.IsStrEmpty(redirectTarget) {
			redirectURL = redirectURL + "&_rt=" + redirectTarget
		}

		ctx.Reply().Redirect(redirectURL)
		return flowStop
	}

	ctx.Log().Info("Authentication successful")
	ctx.Subject().AuthenticationInfo = authcInfo
	ctx.Subject().AuthorizationInfo = formAuth.DoAuthorizationInfo(authcInfo)
	ctx.Session().IsAuthenticated = true

	ctx.Log().Debug(ctx.Subject().AuthenticationInfo)
	ctx.Log().Debug(ctx.Subject().AuthorizationInfo)

	// Change the Anti-CSRF token in use for a request after login for security purposes.
	ctx.Log().Debug("Change Anti-CSRF secret after login for security purpose")
	ctx.AddViewArg(keyAntiCSRFSecret, AppSecurityManager().AntiCSRF.GenerateSecret())

	// Remove the credential
	ctx.Subject().AuthenticationInfo.Credential = nil
	ctx.Session().Set(KeyViewArgAuthcInfo, ctx.Subject().AuthenticationInfo)

	publishOnPostAuthEvent(ctx)

	rt := ctx.Req.Unwrap().FormValue("_rt")
	if formAuth.IsAlwaysToDefaultTarget || ess.IsStrEmpty(rt) {
		ctx.Reply().Redirect(formAuth.DefaultTargetURL)
	} else {
		ctx.Log().Debugf("Redirect to URL found: %v", rt)
		ctx.Reply().Redirect(rt)
	}

	return flowStop
}

// doAuthcAndAuthz method does Authentication and Authorization.
func doAuthcAndAuthz(ascheme scheme.Schemer, ctx *Context) flowResult {
	publishOnPreAuthEvent(ctx)

	// Do Authentication
	authcInfo, err := ascheme.DoAuthenticate(ascheme.ExtractAuthenticationToken(ctx.Req))
	if err != nil || authcInfo == nil {
		ctx.Log().Info("Authentication is failed")

		if ascheme.Scheme() == "basic" {
			basicAuth := ascheme.(*scheme.BasicAuth)
			ctx.Reply().Header(ahttp.HeaderWWWAuthenticate, `Basic realm="`+basicAuth.RealmName+`"`)
		}

		ctx.Reply().Error(&Error{
			Reason:  ErrAuthenticationFailed,
			Code:    http.StatusUnauthorized,
			Message: http.StatusText(http.StatusUnauthorized),
		})
		return flowStop
	}

	ctx.Log().Info("Authentication successful")
	ctx.Subject().AuthenticationInfo = authcInfo
	ctx.Subject().AuthorizationInfo = ascheme.DoAuthorizationInfo(authcInfo)
	ctx.Session().IsAuthenticated = true

	ctx.Log().Debug(ctx.Subject().AuthenticationInfo)
	ctx.Log().Debug(ctx.Subject().AuthorizationInfo)

	// Remove the credential
	ctx.Subject().AuthenticationInfo.Credential = nil

	publishOnPostAuthEvent(ctx)

	return flowCont
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Anti-CSRF Middleware
//___________________________________

func antiCSRFMiddleware(ctx *Context, m *Middleware) {
	// If Anti-CSRF is not enabled, move on.
	// It is highly recommended to enable for web application.
	if !AppSecurityManager().AntiCSRF.Enabled {
		ctx.Log().Trace("Anti CSRF protection is not enabled, clear the cookie if present.")
		AppSecurityManager().AntiCSRF.ClearCookie(ctx.Res, ctx.Req)
		m.Next(ctx)
		return
	}

	// Get cipher secret from anti-csrf cookie
	secret := AppSecurityManager().AntiCSRF.CipherSecret(ctx.Req)
	ctx.AddViewArg(keyAntiCSRFSecret, secret)

	// HTTP Method is safe per defined in
	// https://tools.ietf.org/html/rfc7231#section-4.2.1
	if anticsrf.IsSafeHTTPMethod(ctx.Req.Method) {
		ctx.Log().Tracef("HTTP method[%s] safe method per RFC7231", ctx.Req.Method)
		m.Next(ctx)
		writeAntiCSRFCookie(ctx, secret)
		return
	}

	// Below comment graciously borrowed from django
	// Suppose user visits http://example.com/
	// An active network attacker (man-in-the-middle, MITM) sends a
	// POST form that targets https://example.com/detonate-bomb/ and
	// submits it via JavaScript.
	//
	// The attacker will need to provide a CSRF cookie and token, but
	// that's no problem for a MITM and the session-independent
	// secret we're using. So the MITM can circumvent the CSRF
	// protection. This is true for any HTTP connection, but anyone
	// using HTTPS expects better! For this reason, for
	// https://example.com/ we need additional protection that treats
	// http://example.com/ as completely untrusted. Under HTTPS,
	// Barth et al. found that the Referer header is missing for
	// same-domain requests in only about 0.2% of cases or less, so
	// we can use strict Referer checking.
	if ctx.Req.Scheme == "https" {
		referer, err := url.Parse(ctx.Req.Referer)
		if err != nil {
			ctx.Log().Warnf("Anti-CSRF: malformed referer %s", ctx.Req.Referer)
			ctx.Reply().Error(&Error{
				Reason:  anticsrf.ErrMalformedReferer,
				Code:    http.StatusForbidden,
				Message: http.StatusText(http.StatusForbidden),
			})
			return
		}

		if ess.IsStrEmpty(referer.String()) {
			ctx.Log().Warnf("Anti-CSRF: no referer %s", ctx.Req.Referer)
			ctx.Reply().Error(&Error{
				Reason:  anticsrf.ErrNoReferer,
				Code:    http.StatusForbidden,
				Message: http.StatusText(http.StatusForbidden),
			})
			return
		}

		if !anticsrf.IsSameOrigin(ctx.Req.Unwrap().URL, referer) {
			ctx.Log().Warnf("Anti-CSRF: bad referer %s", ctx.Req.Referer)
			ctx.Reply().Error(&Error{
				Reason:  anticsrf.ErrBadReferer,
				Code:    http.StatusForbidden,
				Message: http.StatusText(http.StatusForbidden),
			})
			return
		}
	}

	// Get request cipher secret from HTTP header or Form
	requestSecret := AppSecurityManager().AntiCSRF.RequestCipherSecret(ctx.Req)
	if requestSecret == nil || !AppSecurityManager().AntiCSRF.IsAuthentic(secret, requestSecret) {
		ctx.Log().Warn("Anti-CSRF: verification failed, invalid cipher secret")
		ctx.Reply().Error(&Error{
			Reason:  anticsrf.ErrNoCookieFound,
			Code:    http.StatusForbidden,
			Message: http.StatusText(http.StatusForbidden),
		})
		return
	}
	ctx.Log().Trace("Anti-CSRF cipher secret verification passed")

	m.Next(ctx)

	writeAntiCSRFCookie(ctx, ctx.viewArgs[keyAntiCSRFSecret].([]byte))
}

func writeAntiCSRFCookie(ctx *Context, secret []byte) {
	if err := AppSecurityManager().AntiCSRF.SetCookie(ctx.Res, secret); err != nil {
		ctx.Log().Error("Unable to write Anti-CSRF cookie")
	}
	ctx.Res.Header().Add(ahttp.HeaderVary, ahttp.HeaderCookie)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

func initSecurity(appCfg *config.Config) error {
	appSecurityManager.IsSSLEnabled = AppIsSSLEnabled()
	return appSecurityManager.Init(appCfg)
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

// tmplAntiCSRFToken method returns the salted Anti-CSRF secret for the view,
// if enabled otherwise empty string.
func tmplAntiCSRFToken(viewArgs map[string]interface{}) string {
	if AppSecurityManager().AntiCSRF.Enabled {
		return AppSecurityManager().AntiCSRF.SaltCipherSecret(viewArgs[keyAntiCSRFSecret].([]byte))
	}
	return ""
}

func getSubjectFromViewArgs(viewArgs map[string]interface{}) *security.Subject {
	if sv, found := viewArgs[KeyViewArgSubject]; found {
		return sv.(*security.Subject)
	}
	return nil
}
