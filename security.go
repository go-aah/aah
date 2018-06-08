// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"aahframework.org/ahttp.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/security.v0"
	"aahframework.org/security.v0/acrypto"
	"aahframework.org/security.v0/anticsrf"
	"aahframework.org/security.v0/authc"
	"aahframework.org/security.v0/scheme"
	"aahframework.org/security.v0/session"
)

const (
	// KeyViewArgAuthcInfo key name is used to store `AuthenticationInfo` instance into `ViewArgs`.
	KeyViewArgAuthcInfo = "_aahAuthcInfo"

	// KeyViewArgSubject key name is used to store `Subject` instance into `ViewArgs`.
	KeyViewArgSubject = "_aahSubject"

	// KeyOAuth2Token key name is used to store OAuth2 Access Token into `aah.Context`.
	KeyOAuth2Token = "_aahOAuth2Token"

	keyAntiCSRF       = "_aahAntiCSRF"
	keyOAuth2StateKey = "_aahOAuth2State"
	keyAuthScheme     = "_aahAuthScheme"
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package methods
//______________________________________________________________________________

// AddSessionStore method allows you to add custom session store which
// implements `session.Storer` interface. The `name` parameter is used in
// aah.conf on `session.store.type = "name"`.
func AddSessionStore(name string, store session.Storer) error {
	return session.AddStore(name, store)
}

// AddPasswordAlgorithm method adds given password algorithm to encoders list.
// Implementation have to implement interface `PasswordEncoder`.
//
// Then you can use it `security.auth_schemes.*`.
func AddPasswordAlgorithm(name string, encoder acrypto.PasswordEncoder) error {
	return acrypto.AddPasswordAlgorithm(name, encoder)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// app methods
//______________________________________________________________________________

func (a *app) SecurityManager() *security.Manager {
	return a.securityMgr
}

func (a *app) SessionManager() *session.Manager {
	return a.SecurityManager().SessionManager
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// app Unexported methods
//______________________________________________________________________________

func (a *app) initSecurity() error {
	asecmgr := security.New()
	asecmgr.IsSSLEnabled = a.IsSSLEnabled()
	if err := asecmgr.Init(a.Config()); err != nil {
		return err
	}

	a.securityMgr = asecmgr
	return nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Authentication and Authorization Middleware
//______________________________________________________________________________

// AuthcAuthzMiddleware is aah Authentication and Authorization Middleware.
func AuthcAuthzMiddleware(ctx *Context, m *Middleware) {
	// Load session from request if its `stateful`.
	if ctx.a.SessionManager().IsStateful() {
		ctx.Subject().Session = ctx.a.SessionManager().GetSession(ctx.Req.Unwrap())
	}

	// If session is authenticated then populate subject and continue the request flow.
	if ctx.Subject().IsAuthenticated() {
		if key := ctx.Session().GetString(keyAuthScheme); key != "" {
			authScheme := ctx.a.SecurityManager().AuthScheme(key)
			if populateSubject(authScheme, ctx) == flowCont {
				m.Next(ctx)
				return
			}
		}
	}

	// If route auth is `anonymous` then continue the request flow
	// No authentication or authorization is required for that route.
	if ctx.route.Auth == "anonymous" {
		m.Next(ctx)
		return
	}

	// Route `auth` attribute or global `default_auth` is not defined
	if ctx.route.Auth == "" {
		// If one or more auth schemes are defined in `security.auth_schemes { ... }`
		// then aah treats that route as `403 Forbidden`.
		if len(ctx.a.SecurityManager().AuthSchemes()) != 0 {
			ctx.Log().Warnf("Auth schemes are configured in security.conf, however attribute 'auth' "+
				"or 'default_auth' is not defined in routes.conf. Let's treat it as 403 Forbidden: %s", ctx.Req.Path)
			ctx.Reply().Forbidden().Error(newError(ErrAccessDenied, http.StatusForbidden))
			return
		}

		// If auth scheme is not configured in security.conf then treat it as `anonymous`.
		ctx.Log().Tracef("Route auth scheme is not configured, so treat it as anonymous: %s", ctx.Req.Path)
		m.Next(ctx)
		return
	}

	ctx.Log().Debugf("Route auth scheme(s): %s", ctx.route.Auth)

	// Supports one or more auth scheme on route
	var result flowResult
	for _, s := range strings.Split(ctx.route.Auth, ",") {
		authScheme := ctx.a.SecurityManager().AuthScheme(strings.TrimSpace(s))
		ctx.Log().Debugf("Processing auth scheme: %s", authScheme.Key())
		switch authScheme.Scheme() {
		case "form":
			result = doFormAuth(authScheme, ctx)
		case "oauth2":
			result = doOAuth2(authScheme, ctx)
		default:
			result = doAuthScheme(authScheme, ctx)
		}

		if result == flowCont {
			break
		}
	}

	if result == flowCont {
		if result, reasons := ctx.hasAccess(); result {
			m.Next(ctx)
		} else {
			ctx.Log().Warnf("Authorization failed:%s", reason2String(reasons))
			ctx.Reply().Forbidden().Error(newErrorWithData(ErrAuthorizationFailed, http.StatusForbidden, reasons))
		}
	}
}

// doFormAuth method does Form Authentication and Authorization.
func doFormAuth(authScheme scheme.Schemer, ctx *Context) flowResult {
	formAuth := authScheme.(*scheme.FormAuth)

	// Check route is login submit URL otherwise send it login URL.
	// Since session is not authenticated.
	if formAuth.LoginSubmitURL != ctx.route.Path && ctx.Req.Method != ahttp.MethodPost {
		loginURL := formAuth.LoginURL
		if formAuth.LoginURL != ctx.Req.Path {
			loginURL = fmt.Sprintf("%s?_rt=%s", loginURL, ctx.Req.URL().String())
		}
		ctx.Reply().Redirect(loginURL)
		return flowAbort
	}

	ctx.e.publishOnPreAuthEvent(ctx)

	if doAuthentication(authScheme, ctx) == flowAbort {
		return flowAbort
	}

	populateAuthorizationInfo(authScheme, ctx)
	debugLogSubjectInfo(ctx)

	ctx.e.publishOnPostAuthEvent(ctx)

	rt := ctx.Req.Unwrap().FormValue("_rt") // redirect to requested URL
	if formAuth.IsAlwaysToDefaultTarget || ess.IsStrEmpty(rt) {
		ctx.Reply().Redirect(formAuth.DefaultTargetURL)
	} else {
		ctx.Log().Debugf("Redirecting to URL found in param '_rt': %s", rt)
		ctx.Reply().Redirect(rt)
	}

	return flowAbort
}

// doOAuth2 method does 3-legged OAuth2 authentication with provider
// and adds the Token into Context. It bit different from FormAuth,
// BasicAuth and Generic (basically it does not have support for
// interface authenticator and authorizer, since its not appliable in the
// OAuth2 flow).
func doOAuth2(authScheme scheme.Schemer, ctx *Context) flowResult {
	ctx.e.publishOnPreAuthEvent(ctx)
	oauth := authScheme.(*scheme.OAuth2)

	// OAuth2 provider login
	if ctx.Req.Path == oauth.LoginURL {
		state, authURL := oauth.ProviderAuthURL(ctx.Req)
		ctx.Session().Set(keyOAuth2StateKey, state)
		ctx.Reply().Redirect(authURL)
		return flowAbort
	}

	// OAuth2 provider callback handling
	if ctx.Req.Path == oauth.RedirectURL {
		defer ctx.Session().Del(keyOAuth2StateKey)

		// Validate OAuth2 callback
		ctx.Log().Debug(ctx.Req.URL().String())
		token, err := oauth.ValidateCallback(ctx.Session().GetString(keyOAuth2StateKey), ctx.Req)
		if err != nil {
			ctx.Log().Error(err)
			ctx.Reply().Unauthorized().Error(newError(err, http.StatusUnauthorized))
			return flowAbort
		}

		// Set successful access token into aah.Context
		ctx.Log().Info("oauth2: Token obtained from provider")
		ctx.Set(KeyOAuth2Token, token)

		if doAuthentication(authScheme, ctx) == flowAbort {
			return flowAbort
		}

		populateAuthorizationInfo(authScheme, ctx)
		debugLogSubjectInfo(ctx)

		ctx.e.publishOnPostAuthEvent(ctx)

		// Redirect to success URL
		ctx.Reply().Redirect(oauth.SuccessURL)
		return flowAbort
	}

	// typically it should not reach here
	ctx.Log().Trace("OAuth2 flow; typically it should not reach here")
	return flowAbort
}

// doAuthScheme method does generic and basic (Authentication and Authorization).
func doAuthScheme(authScheme scheme.Schemer, ctx *Context) flowResult {
	ctx.e.publishOnPreAuthEvent(ctx)

	if doAuthentication(authScheme, ctx) == flowAbort {
		return flowAbort
	}

	populateAuthorizationInfo(authScheme, ctx)
	debugLogSubjectInfo(ctx)

	ctx.e.publishOnPostAuthEvent(ctx)

	return flowCont
}

type principalProviderNoInit interface {
	Principal(keyName string, v ess.Valuer) ([]*authc.Principal, error)
}

func doAuthentication(authScheme scheme.Schemer, ctx *Context) flowResult {
	var authcInfo *authc.AuthenticationInfo
	if c, ok := authScheme.(principalProviderNoInit); ok {
		// Call Subject principals provider
		principals, err := c.Principal(authScheme.Key(), ctx)
		if err != nil {
			ctx.Log().Error(ErrUnableToGetPrincipal)
			ctx.Reply().Unauthorized().Error(newError(ErrUnableToGetPrincipal, http.StatusUnauthorized))
			return flowAbort
		}

		ctx.Log().Debugf("%s: Subject principals obtained", authScheme.Key())
		authcInfo = authc.NewAuthenticationInfo()
		authcInfo.Principals = append(authcInfo.Principals, principals...)
	} else {
		// Call Authentication Info provider
		var err error
		authcInfo, err = authScheme.DoAuthenticate(authScheme.ExtractAuthenticationToken(ctx.Req))
		if err != nil || authcInfo == nil {
			switch sa := authScheme.(type) {
			case *scheme.FormAuth:
				ctx.Log().Infof("%s: Authentication is failed, sending to login failure URL", authScheme.Key())
				redirectURL := sa.LoginFailureURL
				if rt := ctx.Req.Unwrap().FormValue("_rt"); !ess.IsStrEmpty(rt) {
					// rt => Redirect Target
					if strings.IndexByte(redirectURL, '?') > 0 {
						redirectURL += "&_rt=" + rt
					} else {
						redirectURL += "?_rt=" + rt
					}
				}
				ctx.Reply().Redirect(redirectURL)
			case *scheme.BasicAuth:
				ctx.Log().Infof("%s: Authentication is failed", authScheme.Key())
				ctx.Reply().Header(ahttp.HeaderWWWAuthenticate, `Basic realm="`+sa.RealmName+`"`)
				ctx.Reply().Unauthorized().Error(newError(ErrAuthenticationFailed, http.StatusUnauthorized))
			}

			return flowAbort
		}
	}

	populateAuthenticationInfo(authcInfo, ctx)
	ctx.Session().IsAuthenticated = true
	ctx.Session().Set(keyAuthScheme, authScheme.Key())
	ctx.Log().Infof("%s: Authentication successful", authScheme.Key())

	// Add to session its stateful
	if ctx.a.SessionManager().IsStateful() {
		ctx.Session().Set(KeyViewArgAuthcInfo, ctx.Subject().AuthenticationInfo)
	}

	if ctx.a.ViewEngine() != nil {
		// Change the Anti-CSRF token in use for a request after login for security purposes.
		ctx.Log().Info("Change Anti-CSRF secret after login authentication for security purpose")
		ctx.AddViewArg(keyAntiCSRF, ctx.a.SecurityManager().AntiCSRF.GenerateSecret())
	}

	return flowCont
}

func populateSubject(authScheme scheme.Schemer, ctx *Context) flowResult {
	if ctx.Session().IsKeyExists(KeyViewArgAuthcInfo) {
		populateAuthenticationInfo(ctx.Session().Get(KeyViewArgAuthcInfo).(*authc.AuthenticationInfo), ctx)
		populateAuthorizationInfo(authScheme, ctx)
		return flowCont
	}

	ctx.Log().Warn("It seems there is an issue with session data, possibly Authentication Info")
	ctx.Reply().Error(newError(ErrSessionAuthenticationInfo, http.StatusBadRequest))
	return flowAbort
}

func populateAuthenticationInfo(authcInfo *authc.AuthenticationInfo, ctx *Context) {
	ctx.Subject().AuthenticationInfo = authcInfo
	ctx.logger = ctx.Log().WithField("principal", ctx.Subject().PrimaryPrincipal().Value)

	// Remove the credential
	ctx.Subject().AuthenticationInfo.Credential = nil
}

func populateAuthorizationInfo(authScheme scheme.Schemer, ctx *Context) {
	ctx.Subject().AuthorizationInfo = authScheme.DoAuthorizationInfo(ctx.Subject().AuthenticationInfo)
}

func debugLogSubjectInfo(ctx *Context) {
	ctx.Log().Debug(ctx.Subject().AuthenticationInfo)
	ctx.Log().Debug(ctx.Subject().AuthorizationInfo)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Anti-CSRF Middleware
//______________________________________________________________________________

// AntiCSRFMiddleware provides feature to prevent Cross-Site Request Forgery (CSRF)
// attacks.
func AntiCSRFMiddleware(ctx *Context, m *Middleware) {
	// If Anti-CSRF is not enabled, move on.
	// It is highly recommended to enable it for web application.
	if !ctx.a.SecurityManager().AntiCSRF.Enabled || !ctx.route.IsAntiCSRFCheck || ctx.a.ViewEngine() == nil {
		ctx.a.SecurityManager().AntiCSRF.ClearCookie(ctx.Res, ctx.Req)
		m.Next(ctx)
		return
	}

	// Get cipher secret from anti-csrf cookie
	secret := ctx.a.SecurityManager().AntiCSRF.CipherSecret(ctx.Req)
	ctx.AddViewArg(keyAntiCSRF, secret)

	// HTTP Method is safe per defined in
	// https://tools.ietf.org/html/rfc7231#section-4.2.1
	if anticsrf.IsSafeHTTPMethod(ctx.Req.Method) {
		ctx.Log().Tracef("HTTP %s is safe method per RFC7231", ctx.Req.Method)
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
	if ctx.Req.Scheme == ahttp.SchemeHTTPS {
		referer, err := url.Parse(ctx.Req.Referer)
		if err != nil {
			ctx.Log().Warnf("anticsrf: Malformed referer %s", ctx.Req.Referer)
			ctx.Reply().Forbidden().Error(newError(anticsrf.ErrMalformedReferer, http.StatusForbidden))
			return
		}

		if ess.IsStrEmpty(referer.String()) {
			ctx.Log().Warnf("anticsrf: No referer %s", ctx.Req.Referer)
			ctx.Reply().Forbidden().Error(newError(anticsrf.ErrNoReferer, http.StatusForbidden))
			return
		}

		if !anticsrf.IsSameOrigin(ctx.Req.URL(), referer) {
			ctx.Log().Warnf("anticsrf: Bad referer %s", ctx.Req.Referer)
			ctx.Reply().Forbidden().Error(newError(anticsrf.ErrBadReferer, http.StatusForbidden))
			return
		}
	}

	// Get request cipher secret from HTTP header or Form
	requestSecret := ctx.a.SecurityManager().AntiCSRF.RequestCipherSecret(ctx.Req)
	if requestSecret == nil || !ctx.a.SecurityManager().AntiCSRF.IsAuthentic(secret, requestSecret) {
		ctx.Log().Warn("anticsrf: Verification failed, invalid cipher secret")
		ctx.Reply().Forbidden().Error(newError(anticsrf.ErrNoCookieFound, http.StatusForbidden))
		return
	}

	ctx.Log().Info("anticsrf: Cipher secret verification passed")
	m.Next(ctx)

	writeAntiCSRFCookie(ctx, ctx.viewArgs[keyAntiCSRF].([]byte))
}

func writeAntiCSRFCookie(ctx *Context, secret []byte) {
	if err := ctx.a.SecurityManager().AntiCSRF.SetCookie(ctx.Res, secret); err != nil {
		ctx.Log().Error("anticsrf: Unable to write cookie")
	}
}
