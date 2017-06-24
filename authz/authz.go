// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package authz

import (
	"aahframework.org/config.v0"
	"aahframework.org/security.v0-unstable/authc"
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
