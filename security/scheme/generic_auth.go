// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package scheme

import (
	"net/http"

	"aahframe.work/ahttp"
	"aahframe.work/config"
	"aahframe.work/security/authc"
)

var _ Schemer = (*GenericAuth)(nil)

// GenericAuth struct provides generic Auth Scheme for all custom scenario's.
type GenericAuth struct {
	BaseAuth
	IdentityHeader   string
	CredentialHeader string
}

// Init method initializes the Generic authentication scheme from `security.auth_schemes`.
func (g *GenericAuth) Init(cfg *config.Config, keyName string) error {
	g.AppConfig = cfg
	g.KeyName = keyName
	g.KeyPrefix = "security.auth_schemes." + keyName
	g.Name, _ = g.AppConfig.String(g.ConfigKey("scheme"))
	g.IdentityHeader = http.CanonicalHeaderKey(g.AppConfig.StringDefault(g.ConfigKey("header.identity"), "Authorization"))
	g.CredentialHeader = g.AppConfig.StringDefault(g.ConfigKey("header.credential"), "")
	return nil
}

type acauthenticator interface {
	ExtractAuthenticationToken(r *ahttp.Request) *authc.AuthenticationToken
}

// ExtractAuthenticationToken method extracts an authentication token information
// from the HTTP request.
func (g *GenericAuth) ExtractAuthenticationToken(r *ahttp.Request) *authc.AuthenticationToken {
	// Invoke the user provided method if exists for extracting authentication token
	if ac, found := g.authenticator.(acauthenticator); found {
		return ac.ExtractAuthenticationToken(r)
	}

	return &authc.AuthenticationToken{
		Scheme:     g.Scheme(),
		Identity:   r.Header.Get(g.IdentityHeader),
		Credential: r.Header.Get(g.CredentialHeader),
	}
}
