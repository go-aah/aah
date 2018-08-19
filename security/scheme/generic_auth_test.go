// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package scheme

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/security.v0/authc"
	"aahframework.org/security.v0/authz"
	"aahframework.org/test.v0/assert"
)

type testGenericAuthentication struct {
}

var (
	_ authc.Authenticator = (*testGenericAuthentication)(nil)
	_ authz.Authorizer    = (*testGenericAuthentication)(nil)
)

func (tg *testGenericAuthentication) Init(cfg *config.Config) error {
	return nil
}

func (tg *testGenericAuthentication) GetAuthenticationInfo(authcToken *authc.AuthenticationToken) (*authc.AuthenticationInfo, error) {
	if authcToken == nil {
		return authc.NewAuthenticationInfo(), nil
	}

	if strings.HasSuffix(authcToken.Identity, "de4cdd5e96d24708af69b2d0f1b6db14") {
		authcInfo := authc.NewAuthenticationInfo()
		authcInfo.Principals = append(authcInfo.Principals, &authc.Principal{Realm: "database", Value: "jeeva", IsPrimary: true})
		return authcInfo, nil
	} else if strings.HasSuffix(authcToken.Identity, "de4cdd5e96d24708af69b2d0f1b6db14") {
		authcInfo := authc.NewAuthenticationInfo()
		authcInfo.Principals = append(authcInfo.Principals, &authc.Principal{Realm: "database", Value: "john", IsPrimary: true})
		return authcInfo, nil
	}
	return nil, nil
}

func (tg *testGenericAuthentication) GetAuthorizationInfo(authcInfo *authc.AuthenticationInfo) *authz.AuthorizationInfo {
	return nil
}

func TestSchemeAPIAuth(t *testing.T) {
	securityAuthConfigStr := `
  security {
    auth_schemes {
      generic_auth {
        # Generic Auth Scheme
        scheme = "generic"

        # Authenticator is used to validate the subject (aka User)
        authenticator = "security/APIAuthentication"

        # Authorizer is used to get Subject authorization information,
        # such as Roles and Permissions
        authorizer = "security/APIAuthorization"

        header {
          # Default value is 'Authorization'
          identity = "X-Authorization"

          # Optional credential header
          # Typically it's not used, however in the industry people do use it
          # Default value is empty string
          #credential = "X-AuthPass"
        }
      }
    }
  }
  `

	genericAuth := GenericAuth{}
	cfg, _ := config.ParseString(securityAuthConfigStr)

	err := genericAuth.Init(cfg, "generic_auth")
	assert.Nil(t, err)
	assert.Equal(t, "generic", genericAuth.Scheme())
	assert.Equal(t, "X-Authorization", genericAuth.IdentityHeader)
	assert.Equal(t, "", genericAuth.CredentialHeader)

	// Authentication - Failure
	req, _ := http.NewRequest("GET", "http://localhost:8080/users/10010", nil)
	req.Header.Add("X-Authorization", "Bearer de4cdd5e96d24708af69b2d0f1b6db14")
	areq := ahttp.ParseRequest(req, &ahttp.Request{})
	authcToken := genericAuth.ExtractAuthenticationToken(areq)
	authcInfo, err := genericAuth.DoAuthenticate(authcToken)
	assert.NotNil(t, err)
	assert.Equal(t, errors.New("security/authc: authenticator is nil"), err)
	assert.Nil(t, authcInfo)

	ta := &testGenericAuthentication{}
	assert.Nil(t, genericAuth.SetAuthenticator(ta))
	assert.Nil(t, genericAuth.SetAuthorizer(ta))

	// Authentication - Success
	authcInfo, err = genericAuth.DoAuthenticate(authcToken)
	assert.Nil(t, err)
	assert.NotNil(t, authcInfo)
	assert.Equal(t, "jeeva", authcInfo.PrimaryPrincipal().Value)

	authzInfo := genericAuth.DoAuthorizationInfo(authcInfo)
	assert.NotNil(t, authzInfo)
	assert.False(t, authzInfo.HasRole("role1"))

}
