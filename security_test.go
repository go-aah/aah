// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"net/http/httptest"
	"strings"
	"testing"

	"aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/log.v0"
	"aahframework.org/router.v0"
	"aahframework.org/security.v0"
	"aahframework.org/security.v0/anticsrf"
	"aahframework.org/security.v0/authc"
	"aahframework.org/security.v0/authz"
	"aahframework.org/security.v0/scheme"
	"aahframework.org/security.v0/session"
	"aahframework.org/test.v0/assert"
)

func TestSecuritySessionStore(t *testing.T) {
	err := AddSessionStore("file", &session.FileStore{})
	assert.NotNil(t, err)
	assert.Equal(t, "session: store name 'file' is already added, skip it", err.Error())

	err = AddSessionStore("custom", nil)
	assert.NotNil(t, err)
	assert.Equal(t, "session: store value is nil", err.Error())
}

func TestSecuritySessionTemplateFuns(t *testing.T) {
	viewArgs := make(map[string]interface{})

	assert.Nil(t, viewArgs[KeyViewArgSubject])

	bv1 := tmplSessionValue(viewArgs, "my-testvalue")
	assert.Nil(t, bv1)

	bv2 := tmplFlashValue(viewArgs, "my-flashvalue")
	assert.Nil(t, bv2)

	session := &session.Session{Values: make(map[string]interface{})}
	session.Set("my-testvalue", 38458473684763)
	session.SetFlash("my-flashvalue", "user not found")

	assert.False(t, tmplHasRole(viewArgs, "role1"))
	assert.False(t, tmplHasAllRoles(viewArgs, "role1", "role2", "role3"))
	assert.False(t, tmplHasAnyRole(viewArgs, "role1", "role2", "role3"))
	assert.False(t, tmplIsPermitted(viewArgs, "*"))
	assert.False(t, tmplIsPermittedAll(viewArgs, "news:read,write", "manage:*"))

	viewArgs[KeyViewArgSubject] = &security.Subject{
		Session:            session,
		AuthenticationInfo: authc.NewAuthenticationInfo(),
		AuthorizationInfo:  authz.NewAuthorizationInfo(),
	}
	assert.NotNil(t, viewArgs[KeyViewArgSubject])

	v1 := tmplSessionValue(viewArgs, "my-testvalue")
	assert.Equal(t, 38458473684763, v1)

	v2 := tmplFlashValue(viewArgs, "my-flashvalue")
	assert.Equal(t, "user not found", v2)

	v3 := tmplIsAuthenticated(viewArgs)
	assert.False(t, v3)

	assert.False(t, tmplHasRole(viewArgs, "role1"))
	assert.False(t, tmplHasAllRoles(viewArgs, "role1", "role2", "role3"))
	assert.False(t, tmplHasAnyRole(viewArgs, "role1", "role2", "role3"))
	assert.False(t, tmplIsPermitted(viewArgs, "*"))
	assert.False(t, tmplIsPermittedAll(viewArgs, "news:read,write", "manage:*"))

	delete(viewArgs, KeyViewArgSubject)
	v4 := tmplIsAuthenticated(viewArgs)
	assert.False(t, v4)
}

type testFormAuthentication struct {
}

var (
	_ authc.Authenticator = (*testFormAuthentication)(nil)
	_ authz.Authorizer    = (*testFormAuthentication)(nil)
)

func (tfa *testFormAuthentication) Init(cfg *config.Config) error {
	return nil
}

func (tfa *testFormAuthentication) GetAuthenticationInfo(authcToken *authc.AuthenticationToken) (*authc.AuthenticationInfo, error) {
	return testGetAuthenticationInfo(), nil
}

func (tfa *testFormAuthentication) GetAuthorizationInfo(authcInfo *authc.AuthenticationInfo) *authz.AuthorizationInfo {
	return nil
}

func testGetAuthenticationInfo() *authc.AuthenticationInfo {
	authcInfo := authc.NewAuthenticationInfo()
	authcInfo.Principals = append(authcInfo.Principals, &authc.Principal{Realm: "database", Value: "jeeva", IsPrimary: true})
	authcInfo.Credential = []byte("$2y$10$2A4GsJ6SmLAMvDe8XmTam.MSkKojdobBVJfIU7GiyoM.lWt.XV3H6") // welcome123
	return authcInfo
}

func TestSecurityHandleFormAuthcAndAuthz(t *testing.T) {
	// anonymous
	r1 := httptest.NewRequest("GET", "http://localhost:8080/doc/v0.3/mydoc.html", nil)
	ctx1 := &Context{
		Req:      ahttp.ParseRequest(r1, &ahttp.Request{}),
		route:    &router.Route{Auth: "anonymous"},
		subject:  security.AcquireSubject(),
		viewArgs: make(map[string]interface{}),
	}
	authcAndAuthzMiddleware(ctx1, &Middleware{})

	// form auth scheme
	cfg, _ := config.ParseString(`
		security {
		  auth_schemes {
		    # HTTP Form Auth Scheme
		    form_auth {
		      scheme = "form"

		      # Authenticator is used to validate the subject (aka User)
		      authenticator = "security/Authentication"

		      # Authorizer is used to get Subject authorization information,
		      # such as Roles and Permissions
		      authorizer = "security/Authorization"
		    }
		  }
		}
	`)
	err := initSecurity(cfg)
	assert.Nil(t, err)
	r2 := httptest.NewRequest("GET", "http://localhost:8080/doc/v0.3/mydoc.html", nil)
	w2 := httptest.NewRecorder()
	ctx2 := &Context{
		Req:      ahttp.ParseRequest(r2, &ahttp.Request{}),
		Res:      ahttp.GetResponseWriter(w2),
		route:    &router.Route{Auth: "form_auth"},
		subject:  security.AcquireSubject(),
		viewArgs: make(map[string]interface{}),
		reply:    NewReply(),
	}
	authcAndAuthzMiddleware(ctx2, &Middleware{})

	// session is authenticated
	ctx2.Session().IsAuthenticated = true
	authcAndAuthzMiddleware(ctx2, &Middleware{})

	// form auth
	testFormAuth := &testFormAuthentication{}
	formAuth := AppSecurityManager().GetAuthScheme("form_auth").(*scheme.FormAuth)
	err = formAuth.SetAuthenticator(testFormAuth)
	assert.Nil(t, err)
	err = formAuth.SetAuthorizer(testFormAuth)
	assert.Nil(t, err)
	r3 := httptest.NewRequest("POST", "http://localhost:8080/login", nil)
	ctx2.Req = ahttp.ParseRequest(r3, &ahttp.Request{})
	ctx2.Session().Set(KeyViewArgAuthcInfo, testGetAuthenticationInfo())
	authcAndAuthzMiddleware(ctx2, &Middleware{})

	// form auth not authenticated and no credentials
	ctx2.Session().IsAuthenticated = false
	delete(ctx2.Session().Values, KeyViewArgAuthcInfo)
	authcAndAuthzMiddleware(ctx2, &Middleware{})

	// form auth not authenticated and with credentials
	r4 := httptest.NewRequest("POST", "http://localhost:8080/login", strings.NewReader("username=jeeva&password=welcome123"))
	r4.Header.Set(ahttp.HeaderContentType, "application/x-www-form-urlencoded")
	ctx2.Req = ahttp.ParseRequest(r4, &ahttp.Request{})
	authcAndAuthzMiddleware(ctx2, &Middleware{})
}

type testBasicAuthentication struct {
}

var (
	_ authc.Authenticator = (*testBasicAuthentication)(nil)
	_ authz.Authorizer    = (*testBasicAuthentication)(nil)
)

func (tba *testBasicAuthentication) Init(cfg *config.Config) error {
	return nil
}

func (tba *testBasicAuthentication) GetAuthenticationInfo(authcToken *authc.AuthenticationToken) (*authc.AuthenticationInfo, error) {
	return testGetAuthenticationInfo(), nil
}

func (tba *testBasicAuthentication) GetAuthorizationInfo(authcInfo *authc.AuthenticationInfo) *authz.AuthorizationInfo {
	return nil
}

func TestSecurityHandleBasicAuthcAndAuthz(t *testing.T) {
	// basic auth scheme
	cfg, _ := config.ParseString(`
		security {
		  auth_schemes {
		    # HTTP Basic Auth Scheme
		    basic_auth {
		      scheme = "basic"
		      authenticator = "security/Authentication"
		      authorizer = "security/Authorization"
		    }
		  }
		}
	`)
	err := initSecurity(cfg)
	assert.Nil(t, err)
	r1 := httptest.NewRequest("GET", "http://localhost:8080/doc/v0.3/mydoc.html", nil)
	w1 := httptest.NewRecorder()
	ctx1 := &Context{
		Req:      ahttp.ParseRequest(r1, &ahttp.Request{}),
		Res:      ahttp.GetResponseWriter(w1),
		route:    &router.Route{Auth: "basic_auth"},
		viewArgs: make(map[string]interface{}),
		subject:  security.AcquireSubject(),
		reply:    NewReply(),
	}
	authcAndAuthzMiddleware(ctx1, &Middleware{})

	testBasicAuth := &testBasicAuthentication{}
	basicAuth := AppSecurityManager().GetAuthScheme("basic_auth").(*scheme.BasicAuth)
	err = basicAuth.SetAuthenticator(testBasicAuth)
	assert.Nil(t, err)
	err = basicAuth.SetAuthorizer(testBasicAuth)
	assert.Nil(t, err)
	r2 := httptest.NewRequest("GET", "http://localhost:8080/doc/v0.3/mydoc.html", nil)
	ctx1.Req = ahttp.ParseRequest(r2, &ahttp.Request{})
	authcAndAuthzMiddleware(ctx1, &Middleware{})

	r3 := httptest.NewRequest("GET", "http://localhost:8080/doc/v0.3/mydoc.html", nil)
	r3.SetBasicAuth("jeeva", "welcome123")
	ctx1.Req = ahttp.ParseRequest(r3, &ahttp.Request{})
	authcAndAuthzMiddleware(ctx1, &Middleware{})
}

func TestSecurityAntiCSRF(t *testing.T) {
	cfg, _ := config.ParseString("")
	err := initSecurity(cfg)
	assert.Nil(t, err)
	appLogger, _ = log.New(cfg)

	r1 := httptest.NewRequest("POST", "https://localhost:8080/login", strings.NewReader("username=jeeva&password=welcome123"))
	r1.Header.Set(ahttp.HeaderContentType, "application/x-www-form-urlencoded")
	w1 := httptest.NewRecorder()
	ctx1 := &Context{
		Req:      ahttp.AcquireRequest(r1),
		Res:      ahttp.GetResponseWriter(w1),
		viewArgs: make(map[string]interface{}),
		reply:    NewReply(),
		subject:  security.AcquireSubject(),
		route:    &router.Route{IsAntiCSRFCheck: true},
	}

	ctx1.viewArgs[keyAntiCSRFSecret] = AppSecurityManager().AntiCSRF.GenerateSecret()

	// Anti-CSRF request
	ctx1.Req.Scheme = "http"
	antiCSRFMiddleware(ctx1, &Middleware{})
	assert.Equal(t, anticsrf.ErrNoCookieFound, ctx1.reply.err.Reason)
	ctx1.Req.Scheme = "https"

	// No referer
	antiCSRFMiddleware(ctx1, &Middleware{})
	assert.Equal(t, anticsrf.ErrNoReferer, ctx1.reply.err.Reason)

	// https: malformed URL
	ctx1.Req.Referer = ":host:8080"
	antiCSRFMiddleware(ctx1, &Middleware{})
	assert.Equal(t, anticsrf.ErrMalformedReferer, ctx1.reply.err.Reason)

	// Bad referer
	ctx1.Req.Referer = "https:::"
	antiCSRFMiddleware(ctx1, &Middleware{})
	assert.Equal(t, anticsrf.ErrBadReferer, ctx1.reply.err.Reason)

	// Template funcs
	result := tmplAntiCSRFToken(ctx1.viewArgs)
	assert.NotNil(t, result)
	AppSecurityManager().AntiCSRF.Enabled = false
	assert.Equal(t, "", tmplAntiCSRFToken(ctx1.viewArgs))
	antiCSRFMiddleware(ctx1, &Middleware{})
	AppSecurityManager().AntiCSRF.Enabled = true

	// Password Encoder
	err = AddPasswordAlgorithm("mypass", nil)
	assert.NotNil(t, err)
}
