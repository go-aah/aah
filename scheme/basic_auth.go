// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package scheme

import (
	"fmt"

	"aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
	"aahframework.org/security.v0-unstable/acrypto"
	"aahframework.org/security.v0-unstable/authc"
	"aahframework.org/security.v0-unstable/authz"
)

type (
	// BasicAuth struct is aah framework's ready to use Basic Authentication scheme.
	BasicAuth struct {
		BaseAuth
		RealmName string

		isFileRealm bool
		subjectMap  map[string]*basicSubjectInfo
	}

	basicSubjectInfo struct {
		AuthcInfo *authc.AuthenticationInfo
		AuthzInfo *authz.AuthorizationInfo
	}
)

// Init method initializes the Basic authentication scheme from `security.auth_schemes`.
func (b *BasicAuth) Init(cfg *config.Config, keyName string) error {
	b.appCfg = cfg

	b.keyPrefix = "security.auth_schemes." + keyName
	b.scheme = b.appCfg.StringDefault(b.keyPrefix+".scheme", "basic")

	b.RealmName = b.appCfg.StringDefault(b.keyPrefix+".realm_name", "Authentication Required")

	fileRealmPath := b.appCfg.StringDefault(b.keyPrefix+".file_realm", "")
	b.isFileRealm = !ess.IsStrEmpty(fileRealmPath)

	// Basic auth configured to use file based user source
	if b.isFileRealm {
		fileRealmCfg, err := config.LoadFile(fileRealmPath)
		if err != nil {
			return err
		}
		b.subjectMap = make(map[string]*basicSubjectInfo)

		for _, username := range fileRealmCfg.Keys() {
			password := fileRealmCfg.StringDefault(username+".password", "")
			if ess.IsStrEmpty(password) {
				return fmt.Errorf("basicauth: '%v' key is required", username+".password")
			}

			authcInfo := authc.NewAuthenticationInfo()
			authcInfo.Principals = append(authcInfo.Principals, &authc.Principal{Value: username, IsPrimary: true})
			authcInfo.Credential = []byte(password)

			authzInfo := authz.NewAuthorizationInfo()
			if roles, found := fileRealmCfg.StringList(username + ".roles"); found {
				authzInfo.AddRole(roles...)
			}

			if permissions, found := fileRealmCfg.StringList(username + ".permissions"); found {
				authzInfo.AddPermissionString(permissions...)
			}

			b.subjectMap[username] = &basicSubjectInfo{AuthcInfo: authcInfo, AuthzInfo: authzInfo}
		}
	}

	pencoder := b.appCfg.StringDefault(b.keyPrefix+".password_encoder.type", "bcrypt")
	var err error
	b.passwordEncoder, err = acrypto.CreatePasswordEncoder(pencoder)

	return err
}

// Scheme method return authentication scheme name.
func (b *BasicAuth) Scheme() string {
	return b.scheme
}

// DoAuthenticate method calls the registered `Authenticator` with authentication token.
func (b *BasicAuth) DoAuthenticate(authcToken *authc.AuthenticationToken) (*authc.AuthenticationInfo, error) {
	log.Info(authcToken)

	var authcInfo *authc.AuthenticationInfo
	if b.isFileRealm {
		if subject, found := b.subjectMap[authcToken.Identity]; found {
			authcInfo = subject.AuthcInfo
		}
	} else {
		if b.authenticator == nil {
			log.Warn("BasicAuth: authenticator is nil")
			return nil, authc.ErrAuthenticatorIsNil
		}

		authcInfo = b.authenticator.GetAuthenticationInfo(authcToken)
	}

	if authcInfo == nil {
		log.Error("Subject not found")
		return nil, authc.ErrAuthenticationFailed
	}

	log.Info(authcInfo)

	// Compare passwords
	isPasswordOk := b.passwordEncoder.Compare(authcInfo.Credential, []byte(authcToken.Credential))
	if !isPasswordOk {
		log.Error("Subject credentials do not match")
		return nil, authc.ErrAuthenticationFailed
	}

	return authcInfo, nil
}

// DoAuthorizationInfo method calls registered `Authorizer` with authentication information.
func (b *BasicAuth) DoAuthorizationInfo(authcInfo *authc.AuthenticationInfo) *authz.AuthorizationInfo {
	if b.isFileRealm {
		return b.subjectMap[authcInfo.PrimaryPrincipal().Value].AuthzInfo
	}

	if b.authorizer == nil {
		log.Warn("BasicAuth: authorizer is nil")
		return authz.NewAuthorizationInfo()
	}

	return b.authorizer.GetAuthorizationInfo(authcInfo)
}

// ExtractAuthenticationToken method extracts the authentication token information
// from the HTTP request.
func (b *BasicAuth) ExtractAuthenticationToken(r *ahttp.Request) *authc.AuthenticationToken {
	username, password, _ := r.Raw.BasicAuth()
	return &authc.AuthenticationToken{
		Scheme:     b.scheme,
		Identity:   username,
		Credential: password,
	}
}
