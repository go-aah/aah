// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package scheme

import (
	"aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
	"aahframework.org/security.v0-unstable/acrypto"
	"aahframework.org/security.v0-unstable/authc"
	"aahframework.org/security.v0-unstable/authz"
)

var _ Schemer = (*BaseAuth)(nil)

// BaseAuth struct hold base implementation of aah framework's authentication schemes.
type BaseAuth struct {
	authenticator   authc.Authenticator
	authorizer      authz.Authorizer
	appCfg          *config.Config
	scheme          string
	keyPrefix       string
	passwordEncoder acrypto.PasswordEncoder
}

// Init method typically implemented by extending struct.
func (b *BaseAuth) Init(cfg *config.Config, keyName string) error {
	log.Trace("BaseAuth Init called:", keyName)
	return nil
}

// Scheme method return authentication scheme name.
func (b *BaseAuth) Scheme() string {
	if ess.IsStrEmpty(b.scheme) {
		return "unknown"
	}
	return b.scheme
}

// SetAuthenticator method assigns the given `Authenticator` instance to
// authentication scheme.
func (b *BaseAuth) SetAuthenticator(authenticator authc.Authenticator) error {
	if authenticator == nil {
		return authc.ErrAuthenticatorIsNil
	}

	b.authenticator = authenticator
	return b.authenticator.Init(b.appCfg)
}

// SetAuthorizer method assigns the given `Authorizer` instance to
// authentication scheme.
func (b *BaseAuth) SetAuthorizer(authorizer authz.Authorizer) error {
	if authorizer == nil {
		return authz.ErrAuthorizerIsNil
	}

	b.authorizer = authorizer
	return b.authorizer.Init(b.appCfg)
}

// DoAuthenticate method calls the registered `Authenticator` with authentication token.
func (b *BaseAuth) DoAuthenticate(authcToken *authc.AuthenticationToken) (*authc.AuthenticationInfo, error) {
	log.Info(authcToken)

	if b.authenticator == nil {
		log.Warnf("%v: authenticator is nil", b.scheme)
		return nil, authc.ErrAuthenticatorIsNil
	}

	authcInfo, err := b.authenticator.GetAuthenticationInfo(authcToken)
	if err != nil || authcInfo == nil {
		log.Error(err)
		return nil, authc.ErrAuthenticationFailed
	}

	log.Info(authcInfo)

	return authcInfo, nil
}

// DoAuthorizationInfo method calls registered `Authorizer` with authentication information.
func (b *BaseAuth) DoAuthorizationInfo(authcInfo *authc.AuthenticationInfo) *authz.AuthorizationInfo {
	authzInfo := b.authorizer.GetAuthorizationInfo(authcInfo)
	if authzInfo == nil {
		authzInfo = authz.NewAuthorizationInfo()
	}
	log.Info(authzInfo)
	return authzInfo
}

// ExtractAuthenticationToken method typically implementated by extending struct.
func (b *BaseAuth) ExtractAuthenticationToken(r *ahttp.Request) *authc.AuthenticationToken {
	log.Error("security: ExtractAuthenticationToken is not implementated by auth scheme struct")
	return nil
}
