// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package scheme

import (
	"net/http"
	"strings"
	"testing"

	ahttp "aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/security.v0-unstable/authc"
	"aahframework.org/security.v0-unstable/authz"
	"aahframework.org/test.v0/assert"
)

type testAPIAuthentication struct {
}

func (ta *testAPIAuthentication) Init(cfg *config.Config) error {
	return nil
}

func (ta *testAPIAuthentication) GetAuthenticationInfo(authcToken *authc.AuthenticationToken) *authc.AuthenticationInfo {
	if authcToken == nil {
		return authc.NewAuthenticationInfo()
	}

	if strings.HasSuffix(authcToken.Identity, "de4cdd5e96d24708af69b2d0f1b6db14") {
		authcInfo := authc.NewAuthenticationInfo()
		authcInfo.Principals = append(authcInfo.Principals, &authc.Principal{Realm: "database", Value: "jeeva", IsPrimary: true})
		return authcInfo
	} else if strings.HasSuffix(authcToken.Identity, "de4cdd5e96d24708af69b2d0f1b6db14") {
		authcInfo := authc.NewAuthenticationInfo()
		authcInfo.Principals = append(authcInfo.Principals, &authc.Principal{Realm: "database", Value: "john", IsPrimary: true})
		return authcInfo
	}
	return nil
}

func (ta *testAPIAuthentication) GetAuthorizationInfo(authcInfo *authc.AuthenticationInfo) *authz.AuthorizationInfo {
	return nil
}

func TestSchemeAPIAuth(t *testing.T) {
	securityAuthConfigStr := `
  security {
    auth_schemes {
      api_auth {
        # REST API Auth Scheme
        scheme = "api"

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

	apiAuth := APIAuth{}
	cfg, _ := config.ParseString(securityAuthConfigStr)

	err := apiAuth.Init(cfg, "api_auth")
	assert.Nil(t, err)
	assert.Equal(t, "api", apiAuth.Scheme())
	assert.Equal(t, "X-Authorization", apiAuth.IdentityHeader)
	assert.Equal(t, "", apiAuth.CredentialHeader)

	// Authentication - Failure
	req, _ := http.NewRequest("GET", "http://localhost:8080/users/10010", nil)
	req.Header.Add("X-Authorization", "Bearer de4cdd5e96d24708af69b2d0f1b6db14")
	areq := ahttp.ParseRequest(req, &ahttp.Request{})
	authcToken := apiAuth.ExtractAuthenticationToken(areq)
	authcInfo, err := apiAuth.DoAuthenticate(authcToken)
	assert.NotNil(t, err)
	assert.Equal(t, "security: authenticator is nil", err.Error())
	assert.Nil(t, authcInfo)

	ta := &testAPIAuthentication{}
	assert.Nil(t, apiAuth.SetAuthenticator(ta))
	assert.Nil(t, apiAuth.SetAuthorizer(ta))

	// Authentication - Success
	authcInfo, err = apiAuth.DoAuthenticate(authcToken)
	assert.Nil(t, err)
	assert.NotNil(t, authcInfo)
	assert.Equal(t, "jeeva", authcInfo.PrimaryPrincipal().Value)

	authzInfo := apiAuth.DoAuthorizationInfo(authcInfo)
	assert.NotNil(t, authzInfo)
	assert.False(t, authzInfo.HasRole("role1"))

}
