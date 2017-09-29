// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package scheme

import (
	"aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/log.v0"
	"aahframework.org/security.v0/authc"
)

var _ Schemer = (*FormAuth)(nil)

// FormAuth struct is aah framework's ready to use Form Authentication scheme.
type FormAuth struct {
	BaseAuth
	LoginURL                string
	LoginSubmitURL          string
	LoginFailureURL         string
	DefaultTargetURL        string
	IsAlwaysToDefaultTarget bool
	FieldIdentity           string
	FieldCredential         string
}

// Init method initializes the Form authentication scheme from `security.auth_schemes`.
func (f *FormAuth) Init(cfg *config.Config, keyName string) error {
	f.appCfg = cfg

	f.keyPrefix = "security.auth_schemes." + keyName
	f.scheme = f.appCfg.StringDefault(f.keyPrefix+".scheme", "form")

	f.LoginURL = f.appCfg.StringDefault(f.keyPrefix+".url.login", "/login.html")
	f.LoginSubmitURL = f.appCfg.StringDefault(f.keyPrefix+".url.login_submit", "/login")
	f.LoginFailureURL = f.appCfg.StringDefault(f.keyPrefix+".url.login_failure", "/login.html?error=true")
	f.DefaultTargetURL = f.appCfg.StringDefault(f.keyPrefix+".url.default_target", "/")
	f.IsAlwaysToDefaultTarget = f.appCfg.BoolDefault(f.keyPrefix+".url.always_to_default", false)
	f.FieldIdentity = f.appCfg.StringDefault(f.keyPrefix+".field.identity", "username")
	f.FieldCredential = f.appCfg.StringDefault(f.keyPrefix+".field.credential", "password")

	var err error
	f.passwordEncoder, err = passwordAlgorithm(f.appCfg, f.keyPrefix)

	return err
}

// DoAuthenticate method calls the registered `Authenticator` with authentication token.
func (f *FormAuth) DoAuthenticate(authcToken *authc.AuthenticationToken) (*authc.AuthenticationInfo, error) {
	log.Info(authcToken)
	if f.authenticator == nil {
		log.Warnf("%v: authenticator is not properly configured in security.conf", f.scheme)
		return nil, authc.ErrAuthenticatorIsNil
	}

	// Getting authentication information
	authcInfo, err := f.authenticator.GetAuthenticationInfo(authcToken)
	if err != nil || authcInfo == nil {
		if err != nil {
			log.Error(err)
		}
		return nil, authc.ErrAuthenticationFailed
	}

	// Compare passwords
	isPasswordOk := f.passwordEncoder.Compare(authcInfo.Credential, []byte(authcToken.Credential))
	if !isPasswordOk {
		log.Errorf("Subject [%s] credentials do not match", authcToken.Identity)
		return nil, authc.ErrAuthenticationFailed
	}

	if authcInfo.IsLocked || authcInfo.IsExpired {
		log.Errorf("Subject [%s] is locked or expired", authcToken.Identity)
		return nil, authc.ErrAuthenticationFailed
	}

	// Success, return authentication info
	return authcInfo, nil
}

// ExtractAuthenticationToken method extracts the authentication token information
// from the HTTP request.
func (f *FormAuth) ExtractAuthenticationToken(r *ahttp.Request) *authc.AuthenticationToken {
	_ = r.Raw.ParseForm()
	return &authc.AuthenticationToken{
		Scheme:     f.scheme,
		Identity:   r.Raw.FormValue(f.FieldIdentity),
		Credential: r.Raw.FormValue(f.FieldCredential),
	}
}
