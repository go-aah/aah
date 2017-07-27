// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package scheme

import (
	"strings"

	"aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/security.v0-unstable/authc"
	"aahframework.org/security.v0-unstable/authz"
)

type (
	// Schemer interface is implemented by aah framework's authentication scheme.
	Schemer interface {
		// Init method gets called by framework during an application start.
		Init(appCfg *config.Config, keyName string) error

		// Scheme method returns the auth scheme name. For e.g.: form, basic, generic, etc.
		Scheme() string

		// SetAuthenticator method is used to set user provided Authentication implementation.
		SetAuthenticator(authenticator authc.Authenticator) error

		// SetAuthorizer method is used to set user provided Authorization implementation.
		SetAuthorizer(authorizer authz.Authorizer) error

		// DoAuthenticate method called by SecurityManager to get Subject authentication
		// information.
		DoAuthenticate(authcToken *authc.AuthenticationToken) (*authc.AuthenticationInfo, error)

		// DoAuthorizationInfo method called by SecurityManager to get Subject authorization
		// information.
		DoAuthorizationInfo(authcInfo *authc.AuthenticationInfo) *authz.AuthorizationInfo

		// ExtractAuthenticationToken method called by SecurityManager to extract identity details
		// from the HTTP request.
		ExtractAuthenticationToken(r *ahttp.Request) *authc.AuthenticationToken
	}
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package methods
//___________________________________

// New method creates the auth scheme instance for given type.
func New(authSchemeType string) Schemer {
	switch strings.ToLower(authSchemeType) {
	case "form":
		return &FormAuth{}
	case "basic":
		return &BasicAuth{}
	case "generic":
		return &GenericAuth{}
	}
	return nil
}
