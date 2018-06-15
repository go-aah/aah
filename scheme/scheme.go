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

// Schemer interface is used to create new Auth Scheme for aah framework.
type Schemer interface {
	// Init method gets called by aah during an application start.
	//
	// `keyName` is value of security auth scheme key.
	// 		For e.g.:
	// 			security.auth_schemes.<keyname>
	Init(appCfg *config.Config, keyName string) error

	// Key method returns auth scheme configuration KeyName.
	// For e.g: `security.auth_schemes.<keyname>`.
	Key() string

	// Scheme method returns auth scheme name. For e.g.: form, basic, oauth2, generic, etc.
	Scheme() string

	// DoAuthenticate method called by aah SecurityManager to get Subject authentication
	// information.
	DoAuthenticate(authcToken *authc.AuthenticationToken) (*authc.AuthenticationInfo, error)

	// DoAuthorizationInfo method called by aah SecurityManager to get
	// Subject's authorization information if successful authentication.
	DoAuthorizationInfo(authcInfo *authc.AuthenticationInfo) *authz.AuthorizationInfo

	// ExtractAuthenticationToken method called by aah SecurityManager to
	// extract identity details from the HTTP request.
	ExtractAuthenticationToken(r *ahttp.Request) *authc.AuthenticationToken
}

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
	case "oauth2":
		return &OAuth2{}
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
