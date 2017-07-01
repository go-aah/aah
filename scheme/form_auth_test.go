// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package scheme

import (
	"bytes"
	"net/http"
	"strings"
	"testing"

	"aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/security.v0-unstable/authc"
	"aahframework.org/security.v0-unstable/authz"
	"aahframework.org/test.v0/assert"
)

type testFormAuthentication struct {
}

func (tfa *testFormAuthentication) Init(cfg *config.Config) error {
	return nil
}

func (tfa *testFormAuthentication) GetAuthenticationInfo(authcToken *authc.AuthenticationToken) *authc.AuthenticationInfo {
	authcInfo := authc.NewAuthenticationInfo()
	authcInfo.Principals = append(authcInfo.Principals, &authc.Principal{Realm: "database", Value: "jeeva", IsPrimary: true})
	authcInfo.Credential = []byte("$2y$10$2A4GsJ6SmLAMvDe8XmTam.MSkKojdobBVJfIU7GiyoM.lWt.XV3H6") // welcome123
	return authcInfo
}

func (tfa *testFormAuthentication) GetAuthorizationInfo(authcInfo *authc.AuthenticationInfo) *authz.AuthorizationInfo {
	return nil
}

func TestSchemeFormAuth(t *testing.T) {
	securityAuthConfigStr := `
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
  `

	// FormAuth initialize and assertion
	formAuth := FormAuth{}
	cfg, _ := config.ParseString(securityAuthConfigStr)
	err := formAuth.Init(cfg, "form_auth")

	assert.Nil(t, err)
	assert.NotNil(t, formAuth)
	assert.NotNil(t, formAuth.appCfg)
	assert.NotNil(t, formAuth.passwordEncoder)
	assert.Equal(t, "/login.html", formAuth.LoginURL)
	assert.Equal(t, "/login", formAuth.LoginSubmitURL)
	assert.Equal(t, "/login.html?error=true", formAuth.LoginFailureURL)
	assert.Equal(t, "/", formAuth.DefaultTargetURL)
	assert.Equal(t, "username", formAuth.FieldIdentity)
	assert.Equal(t, "password", formAuth.FieldCredential)
	assert.Equal(t, "form", formAuth.Scheme())
	assert.Equal(t, "security.auth_schemes.form_auth", formAuth.keyPrefix)

	// Extract AuthenticationToken
	req, _ := http.NewRequest("POST", "http://localhost:8080/login", strings.NewReader("username=jeeva&password=welcome123"))
	req.Header.Set(ahttp.HeaderContentType, "application/x-www-form-urlencoded")
	areq := ahttp.ParseRequest(req, &ahttp.Request{})
	authcToken := formAuth.ExtractAuthenticationToken(areq)
	assert.NotNil(t, authcToken)
	assert.Equal(t, "form", authcToken.Scheme)
	assert.Equal(t, "jeeva", authcToken.Identity)
	assert.Equal(t, "welcome123", authcToken.Credential)

	// Do Authentication
	authcInfo, err := formAuth.DoAuthenticate(authcToken)
	assert.NotNil(t, err)
	assert.Equal(t, "security: authenticator is nil", err.Error())
	assert.Nil(t, authcInfo)

	testFormAuth := &testFormAuthentication{}
	err = formAuth.SetAuthenticator(testFormAuth)
	assert.Nil(t, err)

	// Valid Authentication
	authcInfo, err = formAuth.DoAuthenticate(authcToken)
	assert.Nil(t, err)
	assert.NotNil(t, authcInfo)
	assert.Equal(t, "jeeva", authcInfo.PrimaryPrincipal().Value)
	assert.Equal(t, "database", authcInfo.PrimaryPrincipal().Realm)
	assert.True(t, bytes.Equal([]byte("$2y$10$2A4GsJ6SmLAMvDe8XmTam.MSkKojdobBVJfIU7GiyoM.lWt.XV3H6"), authcInfo.Credential))

	// Incorrect Credentials
	authcToken.Credential = "welcome@123"
	authcInfo, err = formAuth.DoAuthenticate(authcToken)
	assert.NotNil(t, err)
	assert.Nil(t, authcInfo)
	assert.True(t, err == authc.ErrAuthenticationFailed)
}
