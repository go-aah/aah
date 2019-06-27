// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package authc

import "fmt"

// AuthenticationToken is an account's principals and supporting credentials
// submitted by a user during an authentication attempt.
//
// The auth token is submitted to an Authenticator via the
// GetAuthenticationInfo(authToken) method to get `AuthenticationInfo` for the
// the authentication/log-in process.
//
// Common implementations of an AuthenticationToken would have username/password pairs,
// auth token, or anything else you can think of.
type AuthenticationToken struct {
	// Scheme denotes the authentication scheme. It is derived value.
	// For e.g.: form, basic, api, etc.
	Scheme string

	// Identity is an account username or principal or token.
	Identity string

	// Credential is an account or subject secret.
	Credential string

	// Values contains additional information needed for authc and or authz phase
	Values map[string]interface{}
}

// String method is stringer interface implementation.
func (a AuthenticationToken) String() string {
	return fmt.Sprintf("authenticationtoken(scheme:%s identity:%s credential:*******, values:%v)", a.Scheme, a.Identity, a.Values)
}
