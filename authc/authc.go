// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package authc

import (
	"errors"

	"aahframework.org/config.v0"
)

var (
	// ErrAuthenticatorIsNil error is returned when authenticator is nil in the auth scheme.
	ErrAuthenticatorIsNil = errors.New("security: authenticator is nil")

	// ErrAuthenticationFailed error is returned when user authentication fails;
	// such as subject password doesn't match, is-locked or is-Expired.
	ErrAuthenticationFailed = errors.New("security: authentication failed")
)

// Authenticator interface is implemented by user application to provide
// authentication information during authentication process.
type Authenticator interface {
	// Init method gets called by framework during an application start.
	Init(cfg *config.Config) error

	// GetAuthenticationInfo method gets called when authentication happens for
	// user provided credentials.
	GetAuthenticationInfo(authcToken *AuthenticationToken) *AuthenticationInfo
}
