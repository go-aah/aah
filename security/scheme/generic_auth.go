// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package scheme

import (
	"net/http"

	"aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/security.v0/authc"
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

// ExtractAuthenticationToken method extracts the authentication token information
// from the HTTP request.
func (g *GenericAuth) ExtractAuthenticationToken(r *ahttp.Request) *authc.AuthenticationToken {
	return &authc.AuthenticationToken{
		Scheme:     g.Scheme(),
		Identity:   r.Header.Get(g.IdentityHeader),
		Credential: r.Header.Get(g.CredentialHeader),
	}
}
