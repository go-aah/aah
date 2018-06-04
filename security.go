// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

// Package security houses all the application security implementation Authentication,
// Authorization, Session Management, CSRF, Security Headers, etc.) by aah framework.
package security

import (
	"encoding/gob"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/security.v0/acrypto"
	"aahframework.org/security.v0/anticsrf"
	"aahframework.org/security.v0/authc"
	"aahframework.org/security.v0/scheme"
	"aahframework.org/security.v0/session"
)

var (
	// ErrAuthSchemeIsNil returned when given auth scheme instance is nil.
	ErrAuthSchemeIsNil = errors.New("security: auth scheme is nil")

	// Bcrypt password algorithm instance for Password generate and compare.
	// By default it is enabled.
	Bcrypt acrypto.PasswordEncoder

	// Scrypt password algorithm instance for Password generate and compare.
	// Enable `scrypt` algorithm in `security.conf` otherwise it might be nil.
	Scrypt acrypto.PasswordEncoder

	// Pbkdf2 password algorithm instance for Password generate and compare.
	// Enable `pbkdf2` algorithm in `security.conf` otherwise it might be nil.
	Pbkdf2 acrypto.PasswordEncoder

	subjectPool = &sync.Pool{New: func() interface{} { return &Subject{} }}
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package methods
//___________________________________

// New method creates the security manager initial values and returns it.
func New() *Manager {
	return &Manager{
		authSchemes: make(map[string]scheme.Schemer),
	}
}

// AcquireSubject method gets the subject from pool.
func AcquireSubject() *Subject {
	return subjectPool.Get().(*Subject)
}

// ReleaseSubject method puts authenticatio info, authorization info and subject
// back to pool.
func ReleaseSubject(s *Subject) {
	if s != nil {
		session.ReleaseSession(s.Session)

		s.Reset()
		subjectPool.Put(s)
	}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Security methods
//___________________________________

type (
	// Manager holds aah security management and its implementation.
	Manager struct {
		SessionManager *session.Manager
		SecureHeaders  *SecureHeaders
		AntiCSRF       *anticsrf.AntiCSRF
		IsSSLEnabled   bool
		appCfg         *config.Config
		authSchemes    map[string]scheme.Schemer
	}

	// SecureHeaders holds the composed values of HTTP security headers
	// based on config `security.http_header.*` from `security.conf`.
	SecureHeaders struct {
		Common map[string]string

		// Applied to all HTTPS response.
		STS           string
		PKP           string
		PKPReportOnly bool

		// Applied to all HTML Content-Type
		XSSFilter     string
		CSP           string
		CSPReportOnly bool
	}
)

// Init method initialize the application security configuration `security { ... }`.
// Which is mainly Session, CSRF, Security Headers, etc.
func (m *Manager) Init(appCfg *config.Config) error {
	var err error
	m.appCfg = appCfg

	// Initializing password encoders
	if err = acrypto.InitPasswordEncoders(m.appCfg); err != nil {
		return err
	}

	// Initialize Secure Headers
	m.initializeSecureHeaders()
	Bcrypt = acrypto.PasswordAlgorithm("bcrypt")
	Scrypt = acrypto.PasswordAlgorithm("scrypt")
	Pbkdf2 = acrypto.PasswordAlgorithm("pbkdf2")

	// Initialize Anti-CSRF
	if m.AntiCSRF, err = anticsrf.New(m.appCfg); err != nil {
		return err
	}

	// Initialize Auth Schemes
	keyPrefixAuthScheme := "security.auth_schemes"
	for _, keyAuthScheme := range m.appCfg.KeysByPath(keyPrefixAuthScheme) {
		schemeName := m.appCfg.StringDefault(keyPrefixAuthScheme+"."+keyAuthScheme+".scheme", "")
		if ess.IsStrEmpty(schemeName) {
			return fmt.Errorf("security: '%v' is required", keyPrefixAuthScheme+"."+keyAuthScheme+".scheme")
		}

		authScheme := m.AuthScheme(keyAuthScheme)
		if authScheme == nil {
			authScheme = scheme.New(schemeName)
			if authScheme == nil {
				return fmt.Errorf("security: auth scheme '%v' not available", schemeName)
			}
			_ = m.AddAuthScheme(keyAuthScheme, authScheme)
		}

		// Initialize the auth scheme
		if err = authScheme.Init(m.appCfg, keyAuthScheme); err != nil {
			return err
		}
	}

	// Initialize session manager
	m.SessionManager, err = session.NewManager(m.appCfg)
	return err
}

// AuthScheme method returns the auth scheme instance for given name otherwise nil.
func (m *Manager) AuthScheme(name string) scheme.Schemer {
	if authScheme, found := m.authSchemes[name]; found {
		return authScheme
	}
	return nil
}

// AddAuthScheme method adds the given name and auth scheme to view schemes.
func (m *Manager) AddAuthScheme(name string, authScheme scheme.Schemer) error {
	if authScheme == nil {
		return ErrAuthSchemeIsNil
	}

	if _, found := m.authSchemes[name]; found {
		return fmt.Errorf("security: auth scheme name '%v' is already added", name)
	}

	m.authSchemes[name] = authScheme

	return nil
}

// AuthSchemes method returns all configured auth schemes from `security.conf`
// under `security.auth_schemes { ... }`.
func (m *Manager) AuthSchemes() map[string]scheme.Schemer {
	return m.authSchemes
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Manager Unexported methods
//___________________________________

func (m *Manager) initializeSecureHeaders() {
	keyPrefix := "security.http_header."
	if !m.appCfg.BoolDefault(keyPrefix+"enable", true) {
		return
	}

	cfg := m.appCfg
	m.SecureHeaders = new(SecureHeaders)

	// Common
	common := make(map[string]string)

	// Header: X-Frame-Options
	if xfo := cfg.StringDefault(keyPrefix+"xfo", "SAMEORIGIN"); !ess.IsStrEmpty(xfo) {
		common[ahttp.HeaderXFrameOptions] = strings.TrimSpace(xfo)
	}

	// Header: X-Content-Type-Options
	if xcto := cfg.StringDefault(keyPrefix+"xcto", "nosniff"); !ess.IsStrEmpty(xcto) {
		common[ahttp.HeaderXContentTypeOptions] = strings.TrimSpace(xcto)
	}

	// Header: Referrer-Policy
	if rp := cfg.StringDefault(keyPrefix+"rp", "no-referrer-when-downgrade"); !ess.IsStrEmpty(rp) {
		common[ahttp.HeaderReferrerPolicy] = strings.TrimSpace(rp)
	}

	// Header: X-Permitted-Cross-Domain-Policies
	if xpcdp := cfg.StringDefault(keyPrefix+"xpcdp", "master-only"); !ess.IsStrEmpty(xpcdp) {
		common[ahttp.HeaderXPermittedCrossDomainPolicies] = strings.TrimSpace(xpcdp)
	}

	// Set common headers
	m.SecureHeaders.Common = common

	// Header: X-XSS-Protection, applied to all HTML Content-Type
	m.SecureHeaders.XSSFilter = strings.TrimSpace(cfg.StringDefault(keyPrefix+"xxssp", "1; mode=block"))

	// Header: Strict-Transport-Security, applied to all HTTPS response.
	sts := "max-age=" + parseToSecondsString(
		cfg.StringDefault(keyPrefix+"sts.max_age", "720h"),
		2592000) // 30 days
	if cfg.BoolDefault(keyPrefix+"sts.include_subdomains", false) {
		sts += "; includeSubDomains"
	}
	if cfg.BoolDefault(keyPrefix+"sts.preload", false) {
		sts += "; preload"
	}
	m.SecureHeaders.STS = strings.TrimSpace(sts)

	// Header: Content-Security-Policy, to all HTML Content-Type
	if csp := cfg.StringDefault(keyPrefix+"csp.directives", ""); !ess.IsStrEmpty(csp) {
		// Add Report URI
		if reportURI := cfg.StringDefault(keyPrefix+"csp.report_uri", "false"); !ess.IsStrEmpty(reportURI) {
			csp += "; report-uri " + strings.TrimSpace(reportURI)
		}
		m.SecureHeaders.CSP = strings.TrimSpace(csp)
		m.SecureHeaders.CSPReportOnly = cfg.BoolDefault(keyPrefix+"csp.report_only", false)
	}

	// Header: Public-Key-Pins, applied to all HTTPS response.
	if pkpKeys, found := cfg.StringList(keyPrefix + "pkp.keys"); found && len(pkpKeys) > 0 {
		pkp := []string{}
		for _, key := range pkpKeys {
			pkp = append(pkp, ` pin-sha256="`+key+`"`)
		}

		// Max Age
		pkp = append(pkp, " max-age="+parseToSecondsString(
			cfg.StringDefault(keyPrefix+"pkp.max_age", "720h"), 2592000))

		// Include Subdomains
		if cfg.BoolDefault(keyPrefix+"pkp.include_subdomains", false) {
			pkp = append(pkp, " includeSubdomains")
		}

		// Add Report URI
		if reportURI := cfg.StringDefault(keyPrefix+"pkp.report_uri", ""); !ess.IsStrEmpty(reportURI) {
			pkp = append(pkp, " report-uri="+reportURI)
		}

		m.SecureHeaders.PKP = strings.TrimSpace(strings.Join(pkp, ";"))
		m.SecureHeaders.PKPReportOnly = cfg.BoolDefault(keyPrefix+"pkp.report_only", false)
	}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

func parseToSecondsString(durationStr string, defaultOnErr int64) string {
	value, err := time.ParseDuration(durationStr)
	if err != nil {
		value = time.Second * time.Duration(defaultOnErr)
	}
	return fmt.Sprintf("%v", int64(value.Seconds()))
}

func init() {
	gob.Register(&authc.AuthenticationInfo{})
	gob.Register(&authc.Principal{})
	gob.Register(make([]authc.Principal, 0))
}
