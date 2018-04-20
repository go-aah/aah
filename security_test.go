// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
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
	importPath := filepath.Join(testdataBaseDir(), "webapp1")
	ts, err := newTestServer(t, importPath)
	assert.Nil(t, err)
	defer ts.Close()

	t.Logf("Test Server URL [Security Template funcs]: %s", ts.URL)

	err = ts.app.initSecurity()
	assert.Nil(t, err)

	err = ts.app.initView()
	assert.Nil(t, err)

	vm := ts.app.viewMgr

	viewArgs := make(map[string]interface{})

	assert.Nil(t, viewArgs[KeyViewArgSubject])

	bv1 := vm.tmplSessionValue(viewArgs, "my-testvalue")
	assert.Nil(t, bv1)

	bv2 := vm.tmplFlashValue(viewArgs, "my-flashvalue")
	assert.Nil(t, bv2)

	session := &session.Session{Values: make(map[string]interface{})}
	session.Set("my-testvalue", 38458473684763)
	session.SetFlash("my-flashvalue", "user not found")

	assert.False(t, vm.tmplHasRole(viewArgs, "role1"))
	assert.False(t, vm.tmplHasAllRoles(viewArgs, "role1", "role2", "role3"))
	assert.False(t, vm.tmplHasAnyRole(viewArgs, "role1", "role2", "role3"))
	assert.False(t, vm.tmplIsPermitted(viewArgs, "*"))
	assert.False(t, vm.tmplIsPermittedAll(viewArgs, "news:read,write", "manage:*"))

	viewArgs[KeyViewArgSubject] = &security.Subject{
		Session:            session,
		AuthenticationInfo: authc.NewAuthenticationInfo(),
		AuthorizationInfo:  authz.NewAuthorizationInfo(),
	}
	assert.NotNil(t, viewArgs[KeyViewArgSubject])

	v1 := vm.tmplSessionValue(viewArgs, "my-testvalue")
	assert.Equal(t, 38458473684763, v1)

	v2 := vm.tmplFlashValue(viewArgs, "my-flashvalue")
	assert.Equal(t, "user not found", v2)

	v3 := vm.tmplIsAuthenticated(viewArgs)
	assert.False(t, v3)

	assert.False(t, vm.tmplHasRole(viewArgs, "role1"))
	assert.False(t, vm.tmplHasAllRoles(viewArgs, "role1", "role2", "role3"))
	assert.False(t, vm.tmplHasAnyRole(viewArgs, "role1", "role2", "role3"))
	assert.False(t, vm.tmplIsPermitted(viewArgs, "*"))
	assert.False(t, vm.tmplIsPermittedAll(viewArgs, "news:read,write", "manage:*"))

	delete(viewArgs, KeyViewArgSubject)
	v4 := vm.tmplIsAuthenticated(viewArgs)
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
	importPath := filepath.Join(testdataBaseDir(), "webapp1")
	ts, err := newTestServer(t, importPath)
	assert.Nil(t, err)
	defer ts.Close()

	t.Logf("Test Server URL [Security Form Authc and Authz]: %s", ts.URL)

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

	err = ts.app.Config().Merge(cfg)
	assert.Nil(t, err)

	err = ts.app.initSecurity()
	assert.Nil(t, err)

	r1 := httptest.NewRequest("GET", "http://localhost:8080/doc/v0.3/mydoc.html", nil)
	ctx := ts.app.he.newContext()
	ctx.Req = ahttp.AcquireRequest(r1)
	ctx.route = &router.Route{Auth: "form_auth"}
	AuthcAuthzMiddleware(ctx, &Middleware{})

	// session is authenticated
	t.Log("session is authenticated")
	ctx.Session().IsAuthenticated = true
	AuthcAuthzMiddleware(ctx, &Middleware{})

	// form auth
	t.Log("form auth")
	testFormAuth := &testFormAuthentication{}
	formAuth := ts.app.SecurityManager().GetAuthScheme("form_auth").(*scheme.FormAuth)
	err = formAuth.SetAuthenticator(testFormAuth)
	assert.Nil(t, err)
	err = formAuth.SetAuthorizer(testFormAuth)
	assert.Nil(t, err)
	r2, _ := http.NewRequest("POST", "http://localhost:8080/login", nil)
	ctx.Req = ahttp.AcquireRequest(r2)
	ctx.Session().Set(KeyViewArgAuthcInfo, testGetAuthenticationInfo())
	AuthcAuthzMiddleware(ctx, &Middleware{})

	// form auth not authenticated and no credentials
	t.Log("form auth not authenticated and no credentials")
	ctx.Session().IsAuthenticated = false
	delete(ctx.Session().Values, KeyViewArgAuthcInfo)
	AuthcAuthzMiddleware(ctx, &Middleware{})

	// form auth not authenticated and with credentials
	t.Log("form auth not authenticated and with credentials")
	r3 := httptest.NewRequest("POST", "http://localhost:8080/login", strings.NewReader("username=jeeva&password=welcome123"))
	r3.Header.Set(ahttp.HeaderContentType, "application/x-www-form-urlencoded")
	ctx.Req = ahttp.AcquireRequest(r3)
	AuthcAuthzMiddleware(ctx, &Middleware{})
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
	importPath := filepath.Join(testdataBaseDir(), "webapp1")
	ts, err := newTestServer(t, importPath)
	assert.Nil(t, err)
	defer ts.Close()

	t.Logf("Test Server URL [Security Basic Authc and Authz]: %s", ts.URL)

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

	err = ts.app.Config().Merge(cfg)
	assert.Nil(t, err)

	err = ts.app.initSecurity()
	assert.Nil(t, err)

	r1, err := http.NewRequest(ahttp.MethodGet, "http://localhost:8080/doc/v0.3/mydoc.html", nil)
	assert.Nil(t, err)
	ctx1 := ts.app.he.newContext()
	ctx1.Req = ahttp.AcquireRequest(r1)
	ctx1.Res = ahttp.AcquireResponseWriter(httptest.NewRecorder())
	ctx1.route = &router.Route{Auth: "basic_auth"}
	AuthcAuthzMiddleware(ctx1, &Middleware{})

	testBasicAuth := &testBasicAuthentication{}
	basicAuth := ts.app.SecurityManager().GetAuthScheme("basic_auth").(*scheme.BasicAuth)
	err = basicAuth.SetAuthenticator(testBasicAuth)
	assert.Nil(t, err)
	err = basicAuth.SetAuthorizer(testBasicAuth)
	assert.Nil(t, err)
	r2, err := http.NewRequest(ahttp.MethodGet, "http://localhost:8080/doc/v0.3/mydoc.html", nil)
	assert.Nil(t, err)
	ctx1.Req = ahttp.AcquireRequest(r2)
	AuthcAuthzMiddleware(ctx1, &Middleware{})

	r3, err := http.NewRequest(ahttp.MethodGet, "http://localhost:8080/doc/v0.3/mydoc.html", nil)
	assert.Nil(t, err)
	r3.SetBasicAuth("jeeva", "welcome123")
	ctx1.Req = ahttp.AcquireRequest(r3)
	AuthcAuthzMiddleware(ctx1, &Middleware{})
}

func TestSecurityAntiCSRF(t *testing.T) {
	importPath := filepath.Join(testdataBaseDir(), "webapp1")
	ts, err := newTestServer(t, importPath)
	assert.Nil(t, err)
	defer ts.Close()

	t.Logf("Test Server URL [Security Basic Authc and Authz]: %s", ts.URL)

	cfg, _ := config.ParseString(`
	security {
		anti_csrf {
			enable = true
		}
	}
	`)
	err = ts.app.Config().Merge(cfg)
	assert.Nil(t, err)

	err = ts.app.initView()
	assert.Nil(t, err)

	err = ts.app.initSecurity()
	assert.Nil(t, err)

	r1 := httptest.NewRequest("POST", "https://localhost:8080/login", strings.NewReader("username=jeeva&password=welcome123"))
	r1.Header.Set(ahttp.HeaderContentType, "application/x-www-form-urlencoded")
	w1 := httptest.NewRecorder()
	ctx1 := newContext(w1, r1)
	ctx1.a = ts.app
	ctx1.route = &router.Route{IsAntiCSRFCheck: true}
	ctx1.AddViewArg(keyAntiCSRF, ts.app.SecurityManager().AntiCSRF.GenerateSecret())

	// Anti-CSRF request
	t.Log("Anti-CSRF request")
	ctx1.Req.Scheme = "http"
	AntiCSRFMiddleware(ctx1, &Middleware{})
	assert.Equal(t, anticsrf.ErrNoCookieFound, ctx1.reply.err.Reason)
	ctx1.Req.Scheme = "https"

	// No referer
	t.Log("No referer")
	AntiCSRFMiddleware(ctx1, &Middleware{})
	assert.Equal(t, anticsrf.ErrNoReferer, ctx1.reply.err.Reason)

	// https: malformed URL
	t.Log("https: malformed URL")
	ctx1.Req.Referer = ":host:8080"
	AntiCSRFMiddleware(ctx1, &Middleware{})
	assert.Equal(t, anticsrf.ErrMalformedReferer, ctx1.reply.err.Reason)

	// Bad referer
	t.Log("Bad referer")
	ctx1.Req.Referer = "https:::"
	AntiCSRFMiddleware(ctx1, &Middleware{})
	assert.Equal(t, anticsrf.ErrBadReferer, ctx1.reply.err.Reason)

	// Template funcs
	t.Log("Template funcs")
	result := ts.app.viewMgr.tmplAntiCSRFToken(ctx1.viewArgs)
	assert.NotNil(t, result)
	ts.app.SecurityManager().AntiCSRF.Enabled = false
	assert.Equal(t, "", ts.app.viewMgr.tmplAntiCSRFToken(ctx1.viewArgs))
	AntiCSRFMiddleware(ctx1, &Middleware{})
	ts.app.SecurityManager().AntiCSRF.Enabled = true

	// Password Encoder
	t.Log("Password Encoder")
	err = AddPasswordAlgorithm("mypass", nil)
	assert.NotNil(t, err)
}
