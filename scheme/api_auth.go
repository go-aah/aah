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

// APIAuth struct is aah framework's ready to use API Authentication scheme.
type APIAuth struct {
	BaseAuth
	IdentityHeader   string
	CredentialHeader string
}

// Init method initializes the API authentication scheme from `security.auth_schemes`.
func (a *APIAuth) Init(cfg *config.Config, keyName string) error {
	a.appCfg = cfg
	a.keyPrefix = "security.auth_schemes." + keyName
	a.scheme = a.appCfg.StringDefault(a.keyPrefix+".scheme", "api")
	a.IdentityHeader = http.CanonicalHeaderKey(a.appCfg.StringDefault(a.keyPrefix+".header.identity", "Authorization"))
	a.CredentialHeader = a.appCfg.StringDefault(a.keyPrefix+".header.credential", "")
	return nil
}

// ExtractAuthenticationToken method extracts the authentication token information
// from the HTTP request.
func (a *APIAuth) ExtractAuthenticationToken(r *ahttp.Request) *authc.AuthenticationToken {
	return &authc.AuthenticationToken{
		Scheme:     a.scheme,
		Identity:   r.Header.Get(a.IdentityHeader),
		Credential: r.Header.Get(a.CredentialHeader),
	}
}
