// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
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
}

// String method is stringer interface implementation.
func (a AuthenticationToken) String() string {
	return fmt.Sprintf("authenticationtoken(scheme:%s identity:%s credential:*******)", a.Scheme, a.Identity)
}
