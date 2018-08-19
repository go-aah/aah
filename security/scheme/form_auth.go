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

// FormAuth struct provides aah's OOTB Form Auth scheme.
type FormAuth struct {
	BaseAuth
	IsAlwaysToDefaultTarget bool
	LoginURL                string
	LoginSubmitURL          string
	LoginFailureURL         string
	DefaultTargetURL        string
	FieldIdentity           string
	FieldCredential         string
}

// Init method initializes the Form Auth scheme from `security.auth_schemes`.
func (f *FormAuth) Init(cfg *config.Config, keyName string) error {
	f.AppConfig = cfg
	f.KeyName = keyName
	f.KeyPrefix = "security.auth_schemes." + f.KeyName
	f.Name, _ = f.AppConfig.String(f.ConfigKey("scheme"))

	f.LoginURL = f.AppConfig.StringDefault(f.ConfigKey("url.login"), "/login.html")
	f.LoginSubmitURL = f.AppConfig.StringDefault(f.ConfigKey("url.login_submit"), "/login")
	f.LoginFailureURL = f.AppConfig.StringDefault(f.ConfigKey("url.login_failure"), "/login.html?error=true")
	f.DefaultTargetURL = f.AppConfig.StringDefault(f.ConfigKey("url.default_target"), "/")
	f.IsAlwaysToDefaultTarget = f.AppConfig.BoolDefault(f.ConfigKey("url.always_to_default"), false)
	f.FieldIdentity = f.AppConfig.StringDefault(f.ConfigKey("field.identity"), "username")
	f.FieldCredential = f.AppConfig.StringDefault(f.ConfigKey("field.credential"), "password")

	var err error
	f.passwordEncoder, err = passwordAlgorithm(f.AppConfig, f.KeyPrefix)
	return err
}

// DoAuthenticate method calls the registered `Authenticator` with authentication token.
func (f *FormAuth) DoAuthenticate(authcToken *authc.AuthenticationToken) (*authc.AuthenticationInfo, error) {
	log.Info(authcToken)
	if f.authenticator == nil {
		log.Warnf("%s: '%s' is not properly configured in security.conf", f.KeyName, f.ConfigKey("authenticator"))
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
		log.Errorf("%s: subject [%s] credentials do not match", f.KeyName, authcToken.Identity)
		return nil, authc.ErrAuthenticationFailed
	}

	if authcInfo.IsLocked || authcInfo.IsExpired {
		log.Errorf("%s: subject [%s] is locked or expired", f.KeyName, authcToken.Identity)
		return nil, authc.ErrAuthenticationFailed
	}

	// Success, return authentication info
	return authcInfo, nil
}

// ExtractAuthenticationToken method extracts the authentication token information
// from the HTTP request.
func (f *FormAuth) ExtractAuthenticationToken(r *ahttp.Request) *authc.AuthenticationToken {
	return &authc.AuthenticationToken{
		Scheme:     f.Scheme(),
		Identity:   r.FormValue(f.FieldIdentity),
		Credential: r.FormValue(f.FieldCredential),
	}
}
