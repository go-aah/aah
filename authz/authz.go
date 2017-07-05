// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package authz

import (
	"errors"
	"sync"

	"aahframework.org/config.v0"
	"aahframework.org/security.v0-unstable/authc"
)

var (
	// ErrAuthorizerIsNil error is return when authorizer is nil in the auth scheme.
	ErrAuthorizerIsNil = errors.New("security: authorizer is nil")

	authzInfoPool = &sync.Pool{New: func() interface{} {
		return &AuthorizationInfo{
			roles:       make(parts, 0),
			permissions: make([]*Permission, 0),
		}
	}}
)

// Authorizer interface is gets implemented by user application to provide Subject's
// (aka 'application user') access control information.
type Authorizer interface {
	// Init method gets called by framework during an application start.
	Init(cfg *config.Config) error

	// GetAuthorizationInfo method gets called after authentication is successful
	// to get Subject's aka User access control information such as roles and permissions.
	GetAuthorizationInfo(authcInfo *authc.AuthenticationInfo) *AuthorizationInfo
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Global methods
//___________________________________

// NewAuthorizationInfo method creates an `AuthorizationInfo` instance with zero
// values. Use the returned instance to add roles and permissions for the Subject
// (aka User).
func NewAuthorizationInfo() *AuthorizationInfo {
	return authzInfoPool.Get().(*AuthorizationInfo)
}

// ReleaseAuthorizationInfo method resets and puts back to pool for repurpose.
func ReleaseAuthorizationInfo(authzInfo *AuthorizationInfo) {
	if authzInfo != nil {
		authzInfo.Reset()
		authzInfoPool.Put(authzInfo)
	}
}
