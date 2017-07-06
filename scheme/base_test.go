// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package scheme

import (
	"testing"

	"aahframework.org/config.v0"
	"aahframework.org/security.v0-unstable/authc"
	"aahframework.org/security.v0-unstable/authz"
	"aahframework.org/test.v0/assert"
)

func TestSchemeBaseAuth(t *testing.T) {
	// BaseAuth initialize and assertion
	baseAuth := BaseAuth{}
	cfg, _ := config.ParseString("")
	err := baseAuth.Init(cfg)

	assert.Nil(t, err)
	assert.NotNil(t, baseAuth)
	assert.Nil(t, baseAuth.appCfg)

	// Authenticator & Authorizer to nil
	err = baseAuth.SetAuthenticator(nil)
	assert.NotNil(t, err)
	assert.True(t, err == authc.ErrAuthenticatorIsNil)

	err = baseAuth.SetAuthorizer(nil)
	assert.NotNil(t, err)
	assert.True(t, err == authz.ErrAuthorizerIsNil)

	// Authenticator & Authorizer to non-nil
	testFormAuth := &testFormAuthentication{}
	assert.Nil(t, baseAuth.SetAuthenticator(testFormAuth))
	assert.Nil(t, baseAuth.SetAuthorizer(testFormAuth))

	authcInfo, err := baseAuth.DoAuthenticate(nil)
	assert.Nil(t, err)
	assert.NotNil(t, authcInfo)

	authzInfo := baseAuth.DoAuthorizationInfo(nil)
	assert.NotNil(t, authzInfo)

	authzInfo = baseAuth.DoAuthorizationInfo(authcInfo)
	assert.NotNil(t, authzInfo)
	assert.Nil(t, baseAuth.Init(cfg))
	assert.Equal(t, "unknown", baseAuth.Scheme())

	authcToken := baseAuth.ExtractAuthenticationToken(nil)
	assert.Nil(t, authcToken)
}
