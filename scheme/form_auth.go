// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package scheme

import (
	"aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/log.v0"
	"aahframework.org/security.v0-unstable/acrypto"
	"aahframework.org/security.v0-unstable/authc"
)

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

	scheme          string
	keyPrefix       string
	passwordEncoder acrypto.PasswordEncoder
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

	pencoder := f.appCfg.StringDefault(f.keyPrefix+".password_encoder.type", "bcrypt")
	var err error
	f.passwordEncoder, err = acrypto.CreatePasswordEncoder(pencoder)

	return err
}

// Scheme method return authentication scheme name.
func (f *FormAuth) Scheme() string {
	return f.scheme
}

// DoAuthenticate method calls the registered `Authenticator` with authentication token.
func (f *FormAuth) DoAuthenticate(authcToken *authc.AuthenticationToken) (*authc.AuthenticationInfo, error) {
	if f.authenticator == nil {
		log.Warn("FormAuth: authenticator is nil")
		return nil, authc.ErrAuthenticatorIsNil
	}

	log.Info(authcToken)

	// Getting authentication information
	authcInfo := f.authenticator.GetAuthenticationInfo(authcToken)
	log.Info(authcInfo)

	// Compare passwords
	isPasswordOk := f.passwordEncoder.Compare(authcInfo.Credential, []byte(authcToken.Credential))
	if isPasswordOk && !authcInfo.IsLocked && !authcInfo.IsExpired {
		// Success, return authentication info
		return authcInfo, nil
	}

	// Failed, return error
	return nil, authc.ErrAuthenticationFailed
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
