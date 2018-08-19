// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package authc

import (
	"errors"

	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
)

var (
	// ErrAuthenticatorIsNil error is returned when given authenticator is nil.
	ErrAuthenticatorIsNil = errors.New("security/authc: authenticator is nil")

	//ErrPrincipalIsNil error is returned when given principal provider is nil.
	ErrPrincipalIsNil = errors.New("security/authc: principal provider is nil")

	// ErrAuthenticationFailed error is returned when user authentication fails;
	// such as subject password doesn't match, is-locked or is-expired.
	ErrAuthenticationFailed = errors.New("security/authc: authentication failed")

	// ErrSubjectNotExists error is returned when Subject is not exists in the application
	// datasource.
	ErrSubjectNotExists = errors.New("security/authc: subject not exists")
)

// Authenticator interface is used to provide authentication information of application
// during a login.
type Authenticator interface {
	// Init method gets called by aah during an application start.
	Init(appCfg *config.Config) error

	// GetAuthenticationInfo method called by auth scheme to get subject's authentication
	// info for given authentication token.
	GetAuthenticationInfo(authcToken *AuthenticationToken) (*AuthenticationInfo, error)
}

// PrincipalProvider interface is implemented to provide Subject's principals
// where authentication is done third party, for e.g. OAuth2, etc.
type PrincipalProvider interface {
	// Init method gets called by aah during an application start.
	Init(appCfg *config.Config) error

	// Principal method called by auth scheme to get Principals.
	//
	// 	For e.g: keyName is the auth scheme configuration KeyName.
	// 		 security.auth_schemes.<keyname>
	Principal(keyName string, v ess.Valuer) ([]*Principal, error)
}
