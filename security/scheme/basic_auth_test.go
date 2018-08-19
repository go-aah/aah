// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package scheme

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/security.v0/acrypto"
	"aahframework.org/security.v0/authc"
	"aahframework.org/security.v0/authz"
	"aahframework.org/test.v0/assert"
)

func TestSchemeBasicAuthFileRealm(t *testing.T) {
	securityAuthConfigStr := `
  security {
    auth_schemes {
      basic_auth {
        # HTTP Basic Auth Scheme
        scheme = "basic"

        realm_name = "Authentication Required"

        # supplied dynamicall for test
        file_realm = "path/to/file"
      }
    }
  }
  `

	// BasicAuth initialize and assertion
	basicAuth := BasicAuth{}
	cfg, _ := config.ParseString(securityAuthConfigStr)

	_ = acrypto.InitPasswordEncoders(cfg)

	err := basicAuth.Init(cfg, "basic_auth")
	assert.NotNil(t, err)
	assert.Equal(t, "configuration does not exists: path/to/file", err.Error())

	fileRealmPathError := filepath.Join(getTestdataPath(), "basic_auth_file_realm_error.conf")
	cfg.SetString("security.auth_schemes.basic_auth.file_realm", fileRealmPathError)
	err = basicAuth.Init(cfg, "basic_auth")
	assert.NotNil(t, err)
	assert.Equal(t, "basicauth: 'test2.password' key is required", err.Error())

	fileRealmPath := filepath.Join(getTestdataPath(), "basic_auth_file_realm.conf")
	cfg.SetString("security.auth_schemes.basic_auth.file_realm", fileRealmPath)
	err = basicAuth.Init(cfg, "basic_auth")

	assert.Nil(t, err)
	assert.NotNil(t, basicAuth)
	assert.NotNil(t, basicAuth.AppConfig)
	assert.NotNil(t, basicAuth.passwordEncoder)
	assert.Equal(t, "Authentication Required", basicAuth.RealmName)
	assert.Equal(t, "basic", basicAuth.Scheme())

	// user value
	test1 := basicAuth.subjectMap["test1"]
	assert.NotNil(t, test1)
	assert.Equal(t, "test1", test1.AuthcInfo.PrimaryPrincipal().Value)
	assert.True(t, test1.AuthzInfo.HasRole("admin"))

	// user value
	test2 := basicAuth.subjectMap["test2"]
	assert.NotNil(t, test2)
	assert.Equal(t, "test2", test2.AuthcInfo.PrimaryPrincipal().Value)
	assert.False(t, test2.AuthzInfo.HasRole("admin"))

	// Authenticate - Success
	req, _ := http.NewRequest("GET", "http://localhost:8080/doc.html", nil)
	req.SetBasicAuth("test1", "welcome123")
	areq := ahttp.ParseRequest(req, &ahttp.Request{})
	authcToken := basicAuth.ExtractAuthenticationToken(areq)
	authcInfo, err := basicAuth.DoAuthenticate(authcToken)
	assert.Nil(t, err)
	assert.Equal(t, "test1", authcInfo.PrimaryPrincipal().Value)

	// Authorization
	authzInfo := basicAuth.DoAuthorizationInfo(authcInfo)
	assert.NotNil(t, authzInfo)
	assert.True(t, authzInfo.HasRole("manager"))
	assert.True(t, authzInfo.IsPermitted("newsletter:read"))

	// Authenticate - Failure
	req.SetBasicAuth("test2", "welcome@123")
	areq = ahttp.ParseRequest(req, &ahttp.Request{})
	authcToken = basicAuth.ExtractAuthenticationToken(areq)
	authcInfo, err = basicAuth.DoAuthenticate(authcToken)
	assert.NotNil(t, err)
	assert.Equal(t, errors.New("security/authc: authentication failed"), err)
	assert.Nil(t, authcInfo)

	// Authenticate - Subject not exists
	req.SetBasicAuth("test3", "welcome123")
	areq = ahttp.ParseRequest(req, &ahttp.Request{})
	authcToken = basicAuth.ExtractAuthenticationToken(areq)
	authcInfo, err = basicAuth.DoAuthenticate(authcToken)
	assert.NotNil(t, err)
	assert.Equal(t, errors.New("security/authc: subject not exists"), err)
	assert.Nil(t, authcInfo)
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
	if authcToken == nil {
		return authc.NewAuthenticationInfo(), nil
	}

	if authcToken.Identity == "test1" {
		authcInfo := authc.NewAuthenticationInfo()
		authcInfo.Principals = append(authcInfo.Principals, &authc.Principal{Realm: "database", Value: "test1", IsPrimary: true})
		authcInfo.Credential = []byte("$2y$10$2A4GsJ6SmLAMvDe8XmTam.MSkKojdobBVJfIU7GiyoM.lWt.XV3H6") // welcome123
		return authcInfo, nil
	}
	return nil, authc.ErrSubjectNotExists
}

func (tba *testBasicAuthentication) GetAuthorizationInfo(authcInfo *authc.AuthenticationInfo) *authz.AuthorizationInfo {
	if authcInfo.PrimaryPrincipal().Value == "test1" {
		return authz.NewAuthorizationInfo()
	}
	return nil
}

func TestSchemeBasicAuthCustom(t *testing.T) {
	securityAuthConfigStr := `
  security {
    auth_schemes {
      basic_auth {
        # HTTP Basic Auth Scheme
        scheme = "basic"

        realm_name = "Authentication Required"

				# Authenticator is used to validate the subject (aka User)
        authenticator = "security/BasicAuthentication"

        # Authorizer is used to get Subject authorization information,
        # such as Roles and Permissions
        authorizer = "security/BasicAuthorization"
      }
    }
  }
  `

	// BasicAuth initialize and assertion
	basicAuth := BasicAuth{}
	cfg, _ := config.ParseString(securityAuthConfigStr)

	err := basicAuth.Init(cfg, "basic_auth")
	assert.Nil(t, err)
	assert.NotNil(t, basicAuth)
	assert.NotNil(t, basicAuth.AppConfig)
	assert.NotNil(t, basicAuth.passwordEncoder)
	assert.Equal(t, "Authentication Required", basicAuth.RealmName)
	assert.Equal(t, "basic", basicAuth.Scheme())
	assert.Nil(t, basicAuth.authenticator)
	assert.Nil(t, basicAuth.authorizer)

	// Authenticate - Success
	req, _ := http.NewRequest("GET", "http://localhost:8080/doc.html", nil)
	req.SetBasicAuth("test1", "welcome123")
	areq := ahttp.ParseRequest(req, &ahttp.Request{})
	authcToken := basicAuth.ExtractAuthenticationToken(areq)
	authcInfo, err := basicAuth.DoAuthenticate(authcToken)
	assert.NotNil(t, err)
	assert.Equal(t, errors.New("security/authc: authenticator is nil"), err)
	assert.Nil(t, authcInfo)

	// Authorization
	authzInfo := basicAuth.DoAuthorizationInfo(authcInfo)
	assert.NotNil(t, authzInfo)
	assert.False(t, authzInfo.HasRole("manager"))
	assert.False(t, authzInfo.IsPermitted("newsletter:read"))

	// Custom
	tba := &testBasicAuthentication{}
	assert.Nil(t, basicAuth.SetAuthenticator(tba))
	assert.Nil(t, basicAuth.SetAuthorizer(tba))

	authcInfo, err = basicAuth.DoAuthenticate(authcToken)
	assert.Nil(t, err)
	assert.Equal(t, "test1", authcInfo.PrimaryPrincipal().Value)

	// Authorization
	authzInfo = basicAuth.DoAuthorizationInfo(authcInfo)
	assert.NotNil(t, authzInfo)
	assert.False(t, authzInfo.HasRole("manager"))
	assert.False(t, authzInfo.IsPermitted("newsletter:read"))

	authcInfo.Principals[0].Value = "john"
	authzInfo = basicAuth.DoAuthorizationInfo(authcInfo)
	assert.NotNil(t, authzInfo)
	assert.False(t, authzInfo.HasRole("manager"))
	assert.False(t, authzInfo.IsPermitted("newsletter:read"))

	authcToken.Identity = "newuser"
	authcInfo, err = basicAuth.DoAuthenticate(authcToken)
	assert.NotNil(t, err)
	assert.True(t, err == authc.ErrSubjectNotExists)
}

func getTestdataPath() string {
	wd, _ := os.Getwd()
	return filepath.Join(wd, "testdata")
}
