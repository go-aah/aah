// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package scheme

import (
	"net/http"

	"aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/security.v0-unstable/authc"
)

var _ Schemer = (*GenericAuth)(nil)

// GenericAuth struct is aah framework's ready to use Generic Authentication scheme
// Could be used all custom scenario's.
type GenericAuth struct {
	BaseAuth
	IdentityHeader   string
	CredentialHeader string
}

// Init method initializes the Generic authentication scheme from `security.auth_schemes`.
func (g *GenericAuth) Init(cfg *config.Config, keyName string) error {
	g.appCfg = cfg
	g.keyPrefix = "security.auth_schemes." + keyName
	g.scheme = g.appCfg.StringDefault(g.keyPrefix+".scheme", "generic")
	g.IdentityHeader = http.CanonicalHeaderKey(g.appCfg.StringDefault(g.keyPrefix+".header.identity", "Authorization"))
	g.CredentialHeader = g.appCfg.StringDefault(g.keyPrefix+".header.credential", "")
	return nil
}

// ExtractAuthenticationToken method extracts the authentication token information
// from the HTTP request.
func (g *GenericAuth) ExtractAuthenticationToken(r *ahttp.Request) *authc.AuthenticationToken {
	return &authc.AuthenticationToken{
		Scheme:     g.scheme,
		Identity:   r.Header.Get(g.IdentityHeader),
		Credential: r.Header.Get(g.CredentialHeader),
	}
}
