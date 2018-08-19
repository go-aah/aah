// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package scheme

import (
	"fmt"

	"aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
	"aahframework.org/security.v0/acrypto"
	"aahframework.org/security.v0/authc"
	"aahframework.org/security.v0/authz"
)

var _ Schemer = (*BaseAuth)(nil)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Base Auth methods
//______________________________________________________________________________

// BaseAuth struct hold base implementation of aah framework's authentication schemes.
type BaseAuth struct {
	// Name contains name of the auth scheme.
	// For e.g.: form, basic, oauth2, generic
	Name string

	// KeyName value is auth scheme configuration KeyName.
	// For e.g: `security.auth_schemes.<keyname>`.
	KeyName string

	// KeyPrefix value is composed auth scheme configuration key.
	//
	// 	For e.g.: KeyName is 'form_auth', then KeyPrefix is
	// 		  security.auth_schemes.form_auth
	KeyPrefix string

	// AppConfig value is application configuration, its suppiled via function `Init`.
	AppConfig *config.Config

	authenticator     authc.Authenticator
	principalProvider authc.PrincipalProvider
	authorizer        authz.Authorizer
	passwordEncoder   acrypto.PasswordEncoder
}

// Init method typically implemented by extending struct.
func (b *BaseAuth) Init(appCfg *config.Config, keyName string) error {
	log.Trace("BaseAuth Init called:", keyName)
	return nil
}

// Key method returns auth scheme configuration KeyName.
// For e.g: `security.auth_schemes.<keyname>`.
func (b *BaseAuth) Key() string {
	return b.KeyName
}

// Scheme method return authentication scheme name.
func (b *BaseAuth) Scheme() string {
	if ess.IsStrEmpty(b.Name) {
		return "unknown"
	}
	return b.Name
}

// SetAuthenticator method assigns the given `Authenticator` instance to
// auth scheme.
func (b *BaseAuth) SetAuthenticator(authenticator authc.Authenticator) error {
	if authenticator == nil {
		return authc.ErrAuthenticatorIsNil
	}
	b.authenticator = authenticator
	return b.authenticator.Init(b.AppConfig)
}

// SetAuthorizer method assigns the given `Authorizer` instance to
// auth scheme.
func (b *BaseAuth) SetAuthorizer(authorizer authz.Authorizer) error {
	if authorizer == nil {
		return authz.ErrAuthorizerIsNil
	}
	b.authorizer = authorizer
	return b.authorizer.Init(b.AppConfig)
}

// SetPrincipalProvider method assigns the given `PrincipalProvider` instance to
// auth scheme.
func (b *BaseAuth) SetPrincipalProvider(principal authc.PrincipalProvider) error {
	if principal == nil {
		return authc.ErrPrincipalIsNil
	}
	b.principalProvider = principal
	return b.principalProvider.Init(b.AppConfig)
}

// DoAuthenticate method calls the registered `Authenticator` with authentication token.
func (b *BaseAuth) DoAuthenticate(authcToken *authc.AuthenticationToken) (*authc.AuthenticationInfo, error) {
	if b.authenticator == nil {
		log.Warnf("%s: '%s' is not properly configured in security.conf", b.KeyName, b.ConfigKey("authenticator"))
		return nil, authc.ErrAuthenticatorIsNil
	}

	authcInfo, err := b.authenticator.GetAuthenticationInfo(authcToken)
	if err != nil || authcInfo == nil {
		if err != nil {
			log.Error(err)
		}
		return nil, authc.ErrAuthenticationFailed
	}

	return authcInfo, nil
}

// DoAuthorizationInfo method calls registered `Authorizer` with authentication information.
func (b *BaseAuth) DoAuthorizationInfo(authcInfo *authc.AuthenticationInfo) *authz.AuthorizationInfo {
	authzInfo := b.authorizer.GetAuthorizationInfo(authcInfo)
	if authzInfo == nil {
		authzInfo = authz.NewAuthorizationInfo()
	}
	return authzInfo
}

// ExtractAuthenticationToken method typically implementated by extending struct.
func (b *BaseAuth) ExtractAuthenticationToken(r *ahttp.Request) *authc.AuthenticationToken {
	return nil
}

// ConfigKey method returns fully qualified config key name with given
// suffix key for auth scheme.
func (b *BaseAuth) ConfigKey(suffix string) string {
	return b.KeyPrefix + "." + suffix
}

// ConfigError method creates config `error` instance for errors in the
// auth scheme configuration.
func (b *BaseAuth) ConfigError(keySuffix string) error {
	return fmt.Errorf("%s: config '%s' is required", b.KeyName, b.ConfigKey(keySuffix))
}
