// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package scheme

import (
	"bytes"
	"errors"
	"net/http"
	"strings"
	"testing"

	"aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/security.v0/acrypto"
	"aahframework.org/security.v0/authc"
	"aahframework.org/security.v0/authz"
	"aahframework.org/test.v0/assert"
)

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
	if authcToken == nil {
		return authc.NewAuthenticationInfo(), nil
	}

	if authcToken.Identity == "jeeva" {
		authcInfo := authc.NewAuthenticationInfo()
		authcInfo.Principals = append(authcInfo.Principals, &authc.Principal{Realm: "database", Value: "jeeva", IsPrimary: true})
		authcInfo.Credential = []byte("$2y$10$2A4GsJ6SmLAMvDe8XmTam.MSkKojdobBVJfIU7GiyoM.lWt.XV3H6") // welcome123
		return authcInfo, nil
	} else if authcToken.Identity == "john" {
		authcInfo := authc.NewAuthenticationInfo()
		authcInfo.Principals = append(authcInfo.Principals, &authc.Principal{Realm: "database", Value: "john", IsPrimary: true})
		authcInfo.Credential = []byte("$2y$10$2A4GsJ6SmLAMvDe8XmTam.MSkKojdobBVJfIU7GiyoM.lWt.XV3H6") // welcome123
		authcInfo.IsLocked = true
		return authcInfo, nil
	}
	return nil, authc.ErrSubjectNotExists
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

				password_encoder = "bcrypt"
      }
    }
  }
  `

	// FormAuth initialize and assertion
	formAuth := FormAuth{}
	cfg, _ := config.ParseString(securityAuthConfigStr)

	_ = acrypto.InitPasswordEncoders(cfg)

	err := formAuth.Init(cfg, "form_auth")

	assert.Nil(t, err)
	assert.NotNil(t, formAuth)
	assert.NotNil(t, formAuth.AppConfig)
	assert.NotNil(t, formAuth.passwordEncoder)
	assert.Equal(t, "/login.html", formAuth.LoginURL)
	assert.Equal(t, "/login", formAuth.LoginSubmitURL)
	assert.Equal(t, "/login.html?error=true", formAuth.LoginFailureURL)
	assert.Equal(t, "/", formAuth.DefaultTargetURL)
	assert.Equal(t, "username", formAuth.FieldIdentity)
	assert.Equal(t, "password", formAuth.FieldCredential)
	assert.Equal(t, "form", formAuth.Scheme())
	assert.Equal(t, "security.auth_schemes.form_auth", formAuth.KeyPrefix)

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
	assert.Equal(t, errors.New("security/authc: authenticator is nil"), err)
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

	// Correct Credentials but account is locked
	authcToken.Credential = "welcome123"
	authcToken.Identity = "john"
	authcInfo, err = formAuth.DoAuthenticate(authcToken)
	assert.NotNil(t, err)
	assert.Nil(t, authcInfo)
	assert.True(t, err == authc.ErrAuthenticationFailed)

	authcToken.Identity = "newuser"
	authcInfo, err = formAuth.DoAuthenticate(authcToken)
	assert.NotNil(t, err)
	assert.True(t, err == authc.ErrAuthenticationFailed)
}

func TestSchemeEnablePasswordAlgorithm(t *testing.T) {
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

				password_encoder = "scrypt"
      }
    }
  }
  `

	// FormAuth initialize and assertion
	formAuth := FormAuth{}
	cfg, _ := config.ParseString(securityAuthConfigStr)

	_ = acrypto.InitPasswordEncoders(cfg)

	err := formAuth.Init(cfg, "form_auth")
	assert.True(t, strings.HasPrefix(err.Error(), "'scrypt' password algorithm is not enabled"))
}
