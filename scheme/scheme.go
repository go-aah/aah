// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package scheme

import (
	"fmt"
	"strings"

	"aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/log.v0"
	"aahframework.org/security.v0/acrypto"
	"aahframework.org/security.v0/authc"
	"aahframework.org/security.v0/authz"
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

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

func passwordAlgorithm(cfg *config.Config, keyPrefix string) (acrypto.PasswordEncoder, error) {
	var passAlg string
	pe, _ := cfg.Get(keyPrefix + ".password_encoder")
	if _, ok := pe.(string); !ok {
		passAlg = cfg.StringDefault(keyPrefix+".password_encoder.type", "bcrypt")

		// DEPRECATED, to be removed in v1.0
		log.Warnf("DEPRECATED: Config '%s.password_encoder.type' is deprecated in v0.9, use '%s.password_encoder = \"%s\"' instead. Deprecated config will not break your functionality, its good to update to latest config.", keyPrefix, keyPrefix, passAlg)
	} else {
		passAlg = cfg.StringDefault(keyPrefix+".password_encoder", "bcrypt")
	}

	passwordEncoder := acrypto.PasswordAlgorithm(passAlg)
	if passwordEncoder == nil {
		return nil, fmt.Errorf("'%s' password algorithm is not enabled, please refer to https://docs.aahframework.org/password-encoders.html", passAlg)
	}
	return passwordEncoder, nil
}
