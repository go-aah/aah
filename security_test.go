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
	"aahframework.org/router.v0-unstable"
	"aahframework.org/security.v0-unstable"
	"aahframework.org/security.v0-unstable/authc"
	"aahframework.org/security.v0-unstable/authz"
	"aahframework.org/security.v0-unstable/scheme"
	"aahframework.org/security.v0-unstable/session"
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

	assert.Nil(t, viewArgs[keySubjectValue])

	bv1 := tmplSessionValue(viewArgs, "my-testvalue")
	assert.Nil(t, bv1)

	bv2 := tmplFlashValue(viewArgs, "my-flashvalue")
	assert.Nil(t, bv2)

	session := &session.Session{Values: make(map[string]interface{})}
	session.Set("my-testvalue", 38458473684763)
	session.SetFlash("my-flashvalue", "user not found")

	viewArgs[keySubjectValue] = &security.Subject{Session: session}
	assert.NotNil(t, viewArgs[keySubjectValue])

	v1 := tmplSessionValue(viewArgs, "my-testvalue")
	assert.Equal(t, 38458473684763, v1)

	v2 := tmplFlashValue(viewArgs, "my-flashvalue")
	assert.Equal(t, "user not found", v2)

	v3 := tmplIsAuthenticated(viewArgs)
	assert.False(t, v3)

	delete(viewArgs, keySubjectValue)
	v4 := tmplIsAuthenticated(viewArgs)
	assert.False(t, v4)
}

type testFormAuthentication struct {
}

func (tfa *testFormAuthentication) Init(cfg *config.Config) error {
	return nil
}

func (tfa *testFormAuthentication) GetAuthenticationInfo(authcToken *authc.AuthenticationToken) *authc.AuthenticationInfo {
	return testGetAuthenticationInfo()
}

func (tfa *testFormAuthentication) GetAuthorizationInfo(authcInfo *authc.AuthenticationInfo) *authz.AuthorizationInfo {
	return nil
}

func TestSecurityHandleAuthcAndAuthz(t *testing.T) {
	e := engine{}

	// anonymous
	r1 := httptest.NewRequest("GET", "http://localhost:8080/doc/v0.3/mydoc.html", nil)
	ctx1 := &Context{
		Req:   ahttp.ParseRequest(r1, &ahttp.Request{}),
		route: &router.Route{Auth: "anonymous"},
	}
	result1 := e.handleAuthcAndAuthz(ctx1)
	assert.True(t, result1 == flowCont)

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
		Req:     ahttp.ParseRequest(r2, &ahttp.Request{}),
		Res:     ahttp.GetResponseWriter(w2),
		route:   &router.Route{Auth: "form_auth"},
		subject: &security.Subject{},
		reply:   NewReply(),
	}
	result2 := e.handleAuthcAndAuthz(ctx2)
	assert.True(t, result2 == flowStop)

	// session is authenticated
	ctx2.Session().IsAuthenticated = true
	result3 := e.handleAuthcAndAuthz(ctx2)
	assert.True(t, result3 == flowCont)

	// form auth
	testFormAuth := &testFormAuthentication{}
	formAuth := AppSecurityManager().GetAuthScheme("form_auth").(*scheme.FormAuth)
	err = formAuth.SetAuthenticator(testFormAuth)
	assert.Nil(t, err)
	err = formAuth.SetAuthorizer(testFormAuth)
	assert.Nil(t, err)
	r3 := httptest.NewRequest("POST", "http://localhost:8080/login", nil)
	ctx2.Req = ahttp.ParseRequest(r3, &ahttp.Request{})
	ctx2.Session().Set(keyAuthcInfo, testGetAuthenticationInfo())
	result4 := e.handleAuthcAndAuthz(ctx2)
	assert.True(t, result4 == flowCont)

	// form auth not authenticated and no credentials
	ctx2.Session().IsAuthenticated = false
	delete(ctx2.Session().Values, keyAuthcInfo)
	result5 := e.handleAuthcAndAuthz(ctx2)
	assert.True(t, result5 == flowStop)

	// form auth not authenticated and with credentials
	r4 := httptest.NewRequest("POST", "http://localhost:8080/login", strings.NewReader("username=jeeva&password=welcome123"))
	r4.Header.Set(ahttp.HeaderContentType, "application/x-www-form-urlencoded")
	ctx2.Req = ahttp.ParseRequest(r4, &ahttp.Request{})
	result6 := e.handleAuthcAndAuthz(ctx2)
	assert.True(t, result6 == flowStop)
}

func testGetAuthenticationInfo() *authc.AuthenticationInfo {
	authcInfo := authc.NewAuthenticationInfo()
	authcInfo.Principals = append(authcInfo.Principals, &authc.Principal{Realm: "database", Value: "jeeva", IsPrimary: true})
	authcInfo.Credential = []byte("$2y$10$2A4GsJ6SmLAMvDe8XmTam.MSkKojdobBVJfIU7GiyoM.lWt.XV3H6") // welcome123
	return authcInfo
}
