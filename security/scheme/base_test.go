// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package scheme

import (
	"testing"

	"aahframe.work/config"
	"aahframe.work/security/authc"
	"aahframe.work/security/authz"
	"github.com/stretchr/testify/assert"
)

func TestSchemeBaseAuth(t *testing.T) {
	// BaseAuth initialize and assertion
	baseAuth := BaseAuth{}
	cfg := config.NewEmpty()
	err := baseAuth.Init(cfg, "base")

	assert.Nil(t, err)
	assert.NotNil(t, baseAuth)
	assert.Nil(t, baseAuth.AppConfig)

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

	authcToken := &authc.AuthenticationToken{}
	authcInfo, err := baseAuth.DoAuthenticate(authcToken)
	assert.NotNil(t, err)
	assert.True(t, err == authc.ErrSubjectNotExists)
	assert.Nil(t, authcInfo)

	authcInfo, err = baseAuth.DoAuthenticate(nil)
	assert.Nil(t, err)
	assert.NotNil(t, authcInfo)

	authzInfo := baseAuth.DoAuthorizationInfo(nil)
	assert.NotNil(t, authzInfo)

	authzInfo = baseAuth.DoAuthorizationInfo(authcInfo)
	assert.NotNil(t, authzInfo)
	assert.Nil(t, baseAuth.Init(cfg, "base"))
	assert.Equal(t, "unknown", baseAuth.Scheme())

	authcToken = baseAuth.ExtractAuthenticationToken(nil)
	assert.Nil(t, authcToken)
}
