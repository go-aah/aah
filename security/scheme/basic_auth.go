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
	"aahframework.org/security.v0/authc"
	"aahframework.org/security.v0/authz"
)

var _ Schemer = (*BasicAuth)(nil)

type (
	// BasicAuth struct provides aah's OOTB Basic Auth scheme.
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
	b.AppConfig = cfg
	b.KeyName = keyName
	b.KeyPrefix = "security.auth_schemes." + b.KeyName
	b.Name, _ = b.AppConfig.String(b.ConfigKey("scheme"))

	b.RealmName = b.AppConfig.StringDefault(b.ConfigKey("realm_name"), "Authentication Required")
	fileRealmPath := b.AppConfig.StringDefault(b.ConfigKey("file_realm"), "")
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
			authcInfo.Principals = append(authcInfo.Principals, &authc.Principal{Realm: "Basic", Claim: "Username", Value: username, IsPrimary: true})
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

	var err error
	b.passwordEncoder, err = passwordAlgorithm(b.AppConfig, b.KeyPrefix)

	return err
}

// DoAuthenticate method calls the registered `Authenticator` with authentication token.
func (b *BasicAuth) DoAuthenticate(authcToken *authc.AuthenticationToken) (*authc.AuthenticationInfo, error) {
	log.Info(authcToken)

	var authcInfo *authc.AuthenticationInfo
	var err error
	if b.isFileRealm {
		if subject, found := b.subjectMap[authcToken.Identity]; found {
			ai := *subject.AuthcInfo
			authcInfo = &ai
		} else {
			err = authc.ErrSubjectNotExists
		}
	} else {
		if b.authenticator == nil {
			log.Warnf("%s: basic auth is not properly configured in security.conf, possibly file realm or authenticator", b.Scheme())
			return nil, authc.ErrAuthenticatorIsNil
		}

		authcInfo, err = b.authenticator.GetAuthenticationInfo(authcToken)
	}

	if err != nil {
		log.Error(err)
		return nil, err
	}

	// Compare passwords
	isPasswordOk := b.passwordEncoder.Compare(authcInfo.Credential, []byte(authcToken.Credential))
	if !isPasswordOk {
		log.Errorf("Subject [%s] credentials do not match", authcToken.Identity)
		return nil, authc.ErrAuthenticationFailed
	}

	if authcInfo.IsLocked || authcInfo.IsExpired {
		log.Errorf("Subject [%s] is locked or expired", authcToken.Identity)
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
		log.Warnf("%s: '%s' is not properly configured in security.conf", b.Scheme(), b.ConfigKey("authorizer"))
		return authz.NewAuthorizationInfo()
	}

	authzInfo := b.authorizer.GetAuthorizationInfo(authcInfo)
	if authzInfo == nil {
		authzInfo = authz.NewAuthorizationInfo()
	}

	return authzInfo
}

// ExtractAuthenticationToken method extracts the authentication token information
// from the HTTP request.
func (b *BasicAuth) ExtractAuthenticationToken(r *ahttp.Request) *authc.AuthenticationToken {
	username, password, _ := r.Unwrap().BasicAuth()
	return &authc.AuthenticationToken{
		Scheme:     b.Scheme(),
		Identity:   username,
		Credential: password,
	}
}
