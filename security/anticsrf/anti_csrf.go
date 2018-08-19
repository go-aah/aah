// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package anticsrf

import (
	"crypto/subtle"
	"errors"
	"net/http"

	"aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/security.v0/cookie"
)

// Anti-CSRF errors
var (
	ErrNoReferer        = errors.New("security/anticsrf: no referer")
	ErrMalformedReferer = errors.New("security/anticsrf: malformed referer")
	ErrBadReferer       = errors.New("security/anticsrf: bad referer")
	ErrNoCookieFound    = errors.New("security/anticsrf: no cookie found")
)

// AntiCSRF struct hold the implementation of Anti CSRF (aka XSRF) protection.
type AntiCSRF struct {
	Enabled       bool
	cfg           *config.Config
	cookieMgr     *cookie.Manager
	secretLength  int
	cookieName    string
	headerName    string
	formFieldName string
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package methods
//___________________________________

// New method initializes the Anti-CSRF based on security configuration.
func New(cfg *config.Config) (*AntiCSRF, error) {
	keyPrefix := "security.anti_csrf"
	if !cfg.IsExists(keyPrefix) {
		return &AntiCSRF{Enabled: false}, nil
	}

	c := &AntiCSRF{cfg: cfg}
	c.Enabled = c.cfg.BoolDefault(keyPrefix+".enable", true)
	c.secretLength = c.cfg.IntDefault(keyPrefix+".secret_length", 32)
	c.headerName = c.cfg.StringDefault(keyPrefix+".header_name", "X-Anti-CSRF-Token")
	c.formFieldName = c.cfg.StringDefault(keyPrefix+".form_field_name", "anti_csrf_token")

	// Anit CSRF cookie options
	c.cookieName = c.cfg.StringDefault(keyPrefix+".prefix", "aah") + "_anti_csrf"
	opts := &cookie.Options{
		Name:     c.cookieName,
		Domain:   c.cfg.StringDefault(keyPrefix+".domain", ""),
		Path:     c.cfg.StringDefault(keyPrefix+".path", "/"),
		HTTPOnly: true,
		Secure:   c.cfg.BoolDefault("server.ssl.enable", false),
		SameSite: c.cfg.StringDefault(keyPrefix+".same_site", "Lax"),
	}

	// Anti-CSRF cookie TTL, default is 24 hours
	var err error
	ttl := c.cfg.StringDefault(keyPrefix+".ttl", "24h")
	if opts.MaxAge, err = toSeconds(ttl); err != nil {
		return nil, err
	}

	if c.cookieMgr, err = cookie.NewManager(opts,
		c.cfg.StringDefault(keyPrefix+".sign_key", ""),
		c.cfg.StringDefault(keyPrefix+".enc_key", "")); err != nil {
		return nil, err
	}

	return c, nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// AntiCSRF methods
//___________________________________

// GenerateSecret method generates new secure secret by configured length.
func (ac *AntiCSRF) GenerateSecret() []byte {
	return ess.GenerateSecureRandomKey(ac.secretLength)
}

// CipherSecret method returns the Anti-CSRF secert from the cookie if not available
// generates new secret.
func (ac *AntiCSRF) CipherSecret(r *ahttp.Request) []byte {
	if ac.cookieMgr == nil {
		return ac.GenerateSecret()
	}

	cookie, err := r.Cookie(ac.cookieMgr.Options.Name)
	if err != nil {
		return ac.GenerateSecret()
	}

	chiperSecret, err := ac.cookieMgr.Decode(cookie.Value)
	if err != nil {
		return ac.GenerateSecret()
	}

	return chiperSecret
}

// RequestCipherSecret method returns aah request secret (aka anti-csrf token)
// from the request. The order of secret retrival is HTTP Header,
// Form (Regular and Multipart).
func (ac *AntiCSRF) RequestCipherSecret(r *ahttp.Request) []byte {
	token := r.Header.Get(ac.headerName)
	if ess.IsStrEmpty(token) {
		token = r.FormValue(ac.formFieldName)
	}

	tokenBytes, err := ess.DecodeBase64([]byte(token))
	if err != nil || len(tokenBytes) != ac.secretLength*2 {
		return nil
	}

	return ac.unsaltCipherToken(tokenBytes)
}

// IsAuthentic method compares the given secret and request secret.
func (ac *AntiCSRF) IsAuthentic(secret, requestSecret []byte) bool {
	return subtle.ConstantTimeCompare(secret, requestSecret) == 1
}

// SaltCipherSecret method returns salted chiper secret.
func (ac *AntiCSRF) SaltCipherSecret(secret []byte) string {
	salt := ess.GenerateSecureRandomKey(ac.secretLength)
	return string(ess.EncodeToBase64(append(salt, xorBytes(salt, secret)...)))
}

// SetCookie method write/refresh the Anti-CSRF cookie value and expriy.
func (ac *AntiCSRF) SetCookie(w http.ResponseWriter, secret []byte) error {
	if len(secret) == 0 || ac.cookieMgr == nil {
		return nil
	}

	s := make([]byte, len(secret))
	copy(s, secret)
	value, err := ac.cookieMgr.Encode(s)
	if err != nil {
		return err
	}

	w.Header().Add(ahttp.HeaderVary, ahttp.HeaderCookie)
	ac.cookieMgr.Write(w, value)
	return nil
}

// ClearCookie method is to clear Anti-CSRF cookie when disabled.
func (ac *AntiCSRF) ClearCookie(w http.ResponseWriter, r *ahttp.Request) {
	if !ac.Enabled || ac.cookieMgr == nil {
		return
	}

	if _, err := r.Cookie(ac.cookieMgr.Options.Name); err == nil {
		opts := *ac.cookieMgr.Options
		opts.MaxAge = -1
		http.SetCookie(w, cookie.NewWithOptions("", &opts))
	}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// AntiCSRF Unexported methods
//_________________________________________

func (ac *AntiCSRF) unsaltCipherToken(token []byte) []byte {
	salt := token[:ac.secretLength]
	secret := token[ac.secretLength:]
	return xorBytes(salt, secret)
}
