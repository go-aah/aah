// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package authc

import (
	"errors"
	"sync"

	"aahframework.org/config.v0"
)

var (
	// ErrAuthenticatorIsNil error is returned when authenticator is nil in the auth scheme.
	ErrAuthenticatorIsNil = errors.New("security: authenticator is nil")

	// ErrAuthenticationFailed error is returned when user authentication fails;
	// such as subject password doesn't match, is-locked or is-Expired.
	ErrAuthenticationFailed = errors.New("security: authentication failed")

	authcInfoPool = &sync.Pool{New: func() interface{} {
		return &AuthenticationInfo{
			Principals: make([]*Principal, 0),
		}
	}}
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

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package methods
//___________________________________

// NewAuthenticationInfo method creates an `AuthenticationInfo` instance with zero
// values. Then using this instance you fill-in user credential, principals, locked,
// expried information.
func NewAuthenticationInfo() *AuthenticationInfo {
	return authcInfoPool.Get().(*AuthenticationInfo)
}

// ReleaseAuthenticationInfo method resets instance and puts back to pool repurpose.
func ReleaseAuthenticationInfo(authcInfo *AuthenticationInfo) {
	if authcInfo != nil {
		authcInfo.Reset()
		authcInfoPool.Put(authcInfo)
	}
}
