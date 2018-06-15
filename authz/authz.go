// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package authz

import (
	"errors"
	"fmt"

	"aahframework.org/config.v0"
	"aahframework.org/security.v0/authc"
)

var (
	// ErrAuthorizerIsNil error is return when authorizer is nil in the auth scheme.
	ErrAuthorizerIsNil = errors.New("security/authz: authorizer is nil")
)

// Authorizer interface is used to provide authorization info (roles and permissions)
// after successful authentication.
type Authorizer interface {
	// Init method gets called by aah during an application start.
	Init(appCfg *config.Config) error

	// GetAuthorizationInfo method called by auth scheme after authentication
	// successful to get Subject's (aka User) access control information
	// such as roles and permissions.
	GetAuthorizationInfo(authcInfo *authc.AuthenticationInfo) *AuthorizationInfo
}

// Reason struct used to represent authorization failed details.
type Reason struct {
	Func     string
	Expected string
	Got      string
}

// String method is Stringer interface
func (r Reason) String() string {
	return fmt.Sprintf("reason(func=%s expected=%s got=%s)", r.Func, r.Expected, r.Got)
}

// Error method is error interface
func (r Reason) Error() string {
	return fmt.Sprintf("error(func=%s expected=%s got=%s)", r.Func, r.Expected, r.Got)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

// acquirePermission method gets from pool or creates an `Permission` instance
// with zero values.
func acquirePermission() *Permission {
	return permissionPool.Get().(*Permission)
}

// releasePermission method resets and puts back to pool for repurpose.
func releasePermission(permissions ...*Permission) {
	for _, p := range permissions {
		if p != nil {
			p.Reset()
			permissionPool.Put(p)
		}
	}
}
