// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"aahframe.work/ahttp"
	"aahframe.work/config"
	"aahframe.work/essentials"
	"aahframe.work/router"
	"aahframe.work/security"
	"aahframe.work/security/anticsrf"
	"aahframe.work/security/authc"
	"aahframe.work/security/authz"
	"aahframe.work/security/scheme"
	"aahframe.work/security/session"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/publicsuffix"
	"golang.org/x/oauth2"
)

func TestSecuritySessionStore(t *testing.T) {
	err := AddSessionStore("file", &session.FileStore{})
	assert.NotNil(t, err)
	assert.Equal(t, "session: store name 'file' is already added, skip it", err.Error())

	err = AddSessionStore("custom", nil)
	assert.NotNil(t, err)
	assert.Equal(t, errors.New("security/session: store value is nil"), err)
}

func TestSecuritySessionTemplateFuns(t *testing.T) {
	importPath := filepath.Join(testdataBaseDir(), "webapp1")
	ts := newTestServer(t, importPath)
	defer ts.Close()

	t.Logf("Test Server URL [Security Template funcs]: %s", ts.URL)

	err := ts.app.initSecurity()
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

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Form Auth test
//______________________________________________________________________________

type testFormAuthentication struct{}

var (
	_ authc.Authenticator = (*testFormAuthentication)(nil)
	_ authz.Authorizer    = (*testFormAuthentication)(nil)
)

func (tfa *testFormAuthentication) Init(cfg *config.Config) error { return nil }
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
	ts := newTestServer(t, importPath)
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

	err := ts.app.Config().Merge(cfg)
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
	formAuth := ts.app.SecurityManager().AuthScheme("form_auth").(*scheme.FormAuth)
	err = formAuth.SetAuthenticator(testFormAuth)
	assert.Nil(t, err)
	err = formAuth.SetAuthorizer(testFormAuth)
	assert.Nil(t, err)
	r2, _ := http.NewRequest("POST", "http://localhost:8080/login?_rt=/", nil)
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

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// OAuth2 Auth test
//______________________________________________________________________________

type testOAuth2Auth struct{}

var (
	_ authc.PrincipalProvider = (*testOAuth2Auth)(nil)
	_ authz.Authorizer        = (*testOAuth2Auth)(nil)
)

func (to *testOAuth2Auth) Init(cfg *config.Config) error { return nil }
func (to *testOAuth2Auth) Principal(keyName string, v ess.Valuer) ([]*authc.Principal, error) {
	principals := make([]*authc.Principal, 0)
	principals = append(principals,
		&authc.Principal{Realm: "TestOAuth", Claim: "Email", Value: "test@test.com", IsPrimary: true})
	return principals, nil
}
func (to *testOAuth2Auth) GetAuthorizationInfo(authcInfo *authc.AuthenticationInfo) *authz.AuthorizationInfo {
	return nil
}

func createOAuth2TestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/auth/login" {
			redirectURL := r.FormValue("redirect_uri") +
				"?code=EAACmZAkEPRWwBABp3pPRSAww7i4NS" + "&state=" + r.FormValue("state")
			http.Redirect(w, r, redirectURL, 302)
			return
		}

		if r.Method == http.MethodPost && r.URL.Path == "/auth/token" {
			if strings.HasSuffix(r.FormValue("code"), ":senderror") {
				return
			}

			// test token
			w.Header().Set("Content-Type", "application/json")
			token := &oauth2.Token{
				TokenType:    "bearer",
				AccessToken:  "EAACmZAkEPRWwBABp3pPRSAww7i4NSIbGHjwmGpR0tuqN29ZCXA2",
				RefreshToken: "SzGotMzeKoIlCLVrZApwEfo4zNA10mcsWMeViZAy2y7legE6aEZD",
				Expiry:       time.Now().UTC().AddDate(0, 0, 60),
			}
			_ = json.NewEncoder(w).Encode(token)
			return
		}
	}))
}

func TestSecurityHandleOAuth2(t *testing.T) {
	importPath := filepath.Join(testdataBaseDir(), "webapp1")
	ts := newTestServer(t, importPath)
	defer ts.Close()

	t.Logf("Test Server URL [Security OAuth2 Authc and Authz]: %s", ts.URL)

	// form auth scheme
	cfg, _ := config.ParseString(`
security {
	auth_schemes {
		local_oauth {
			scheme = "oauth2"
			client {
				id = "clientid"
				secret = "clientsecret"
				sign_key = "5a977494319cde3203fbb49711f08ad2"
				provider {
					url {
						auth = "http://localhost/auth/login"
						token = "http://localhost/auth/token"
					}
				}
			}
			principal = "security/SubjectPrincipalProvider"
			authorizer = "security/AuthorizationProvider"
		}
	}
}`)

	err := ts.app.Config().Merge(cfg)
	assert.Nil(t, err)

	// Load new security config and router
	err = ts.app.initSecurity()
	assert.Nil(t, err)
	err = ts.app.initRouter()
	assert.Nil(t, err, "router init issue")

	ots := createOAuth2TestServer()
	defer ots.Close()

	t.Logf("Local dummy mock OAuth2 server: %s", ots.URL)
	oauth := ts.app.SecurityManager().AuthScheme("local_oauth").(*scheme.OAuth2)
	assert.NotNil(t, oauth)
	oauth.Config().Endpoint = oauth2.Endpoint{
		AuthURL:  ots.URL + "/auth/login",
		TokenURL: ots.URL + "/auth/token",
	}

	cookieJar, _ := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	client := &http.Client{Jar: cookieJar}

	r1, _ := http.NewRequest(http.MethodGet, ts.URL+"/local-oauth/login", nil)
	resp, err := client.Do(r1)
	assert.Nil(t, err)
	// assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	testOAuth2 := new(testOAuth2Auth)
	assert.Nil(t, oauth.SetPrincipalProvider(testOAuth2))
	assert.Nil(t, oauth.SetAuthorizer(testOAuth2))

	resp, err = client.Do(r1)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// set auth attribute to oauth2
	r2, _ := http.NewRequest(http.MethodGet, ts.URL+"/get-json-oauth2", nil)
	domain := ts.app.Router().Lookup(ts.URL[7:])
	domain.LookupByName("get_json_oauth2").Auth = "local_oauth"
	resp, err = client.Do(r2)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.True(t, strings.HasPrefix(responseBody(resp), `{"ProductID":190398398,"ProductName":"JSONP product","Username"`))
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Basic Auth test
//______________________________________________________________________________

type testBasicAuth struct{}

var (
	_ authc.Authenticator = (*testBasicAuth)(nil)
	_ authz.Authorizer    = (*testBasicAuth)(nil)
)

func (tba *testBasicAuth) Init(cfg *config.Config) error { return nil }
func (tba *testBasicAuth) GetAuthenticationInfo(authcToken *authc.AuthenticationToken) (*authc.AuthenticationInfo, error) {
	return testGetAuthenticationInfo(), nil
}
func (tba *testBasicAuth) GetAuthorizationInfo(authcInfo *authc.AuthenticationInfo) *authz.AuthorizationInfo {
	return nil
}

func TestSecurityHandleBasicAuthcAndAuthz(t *testing.T) {
	importPath := filepath.Join(testdataBaseDir(), "webapp1")
	ts := newTestServer(t, importPath)
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

	err := ts.app.Config().Merge(cfg)
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

	testBasicAuth := &testBasicAuth{}
	basicAuth := ts.app.SecurityManager().AuthScheme("basic_auth").(*scheme.BasicAuth)
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
	ts := newTestServer(t, importPath)
	defer ts.Close()

	t.Logf("Test Server URL [Security Basic Authc and Authz]: %s", ts.URL)

	cfg, _ := config.ParseString(`
	security {
		anti_csrf {
			enable = true
		}
	}
	`)
	err := ts.app.Config().Merge(cfg)
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
	r1.Header.Set("Referer", ":host:8080")
	ctx1.Req = ahttp.AcquireRequest(r1)
	AntiCSRFMiddleware(ctx1, &Middleware{})
	assert.Equal(t, anticsrf.ErrMalformedReferer, ctx1.reply.err.Reason)

	// Bad referer
	t.Log("Bad referer")
	r1.Header.Set("Referer", "https:::")
	ctx1.Req = ahttp.AcquireRequest(r1)
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
