// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

// Package security houses all the application security implementation (Form Auth,
// Basic Auth, Token Auth, Session, CORS, CSRF, Security Headers, etc.) by aah framework.
package security

import (
	"encoding/gob"
	"errors"
	"fmt"
	"sync"

	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/security.v0-unstable/authc"
	"aahframework.org/security.v0-unstable/authz"
	"aahframework.org/security.v0-unstable/scheme"
	"aahframework.org/security.v0-unstable/session"
)

// Version is security library version no. of aah framework
const Version = "0.6"

var (
	// ErrAuthSchemeIsNil returned when given auth scheme instance is nil.
	ErrAuthSchemeIsNil = errors.New("security: auth scheme is nil")

	subjectPool = &sync.Pool{New: func() interface{} { return &Subject{} }}
)

type (
	// Manager holds aah security management and its implementation.
	Manager struct {
		SessionManager *session.Manager
		appCfg         *config.Config
		authSchemes    map[string]scheme.Schemer
	}
)

// New method creates the security manager initial values and returns it.
func New() *Manager {
	return &Manager{
		authSchemes: make(map[string]scheme.Schemer),
	}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Security methods
//___________________________________

// Init method initialize the application security configuration `security { ... }`.
// Which is mainly Session, CORS, CSRF, Security Headers, etc.
func (m *Manager) Init(appCfg *config.Config) error {
	m.appCfg = appCfg

	// Initialize Auth Schemes
	keyPrefixAuthScheme := "security.auth_schemes"
	for _, keyAuthScheme := range m.appCfg.KeysByPath(keyPrefixAuthScheme) {
		schemeName := m.appCfg.StringDefault(keyPrefixAuthScheme+"."+keyAuthScheme+".scheme", "")
		if ess.IsStrEmpty(schemeName) {
			return fmt.Errorf("security: '%v' is required", keyPrefixAuthScheme+"."+keyAuthScheme+".scheme")
		}

		authScheme := m.GetAuthScheme(keyAuthScheme)
		if authScheme == nil {
			authScheme = scheme.New(schemeName)
			if authScheme == nil {
				return fmt.Errorf("security: auth scheme '%v' not available", schemeName)
			}
			_ = m.AddAuthScheme(keyAuthScheme, authScheme)
		}

		// Initialize the auth scheme
		if err := authScheme.Init(m.appCfg, keyAuthScheme); err != nil {
			return err
		}
	}

	// Initialize session manager
	var err error
	if m.SessionManager, err = session.NewManager(m.appCfg); err != nil {
		return err
	}
	_ = m
	return nil
}

// GetAuthScheme ...
func (m *Manager) GetAuthScheme(name string) scheme.Schemer {
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

// AcquireSubject method gets the subject from pool.
func AcquireSubject() *Subject {
	return subjectPool.Get().(*Subject)
}

// ReleaseSubject method puts authenticatio info, authorization info and subject
// back to pool.
func ReleaseSubject(s *Subject) {
	if s != nil {
		authc.ReleaseAuthenticationInfo(s.AuthenticationInfo)
		authz.ReleaseAuthorizationInfo(s.AuthorizationInfo)
		session.ReleaseSession(s.Session)

		s.Reset()
		subjectPool.Put(s)
	}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

func init() {
	gob.Register(&authc.AuthenticationInfo{})
	gob.Register(&authc.Principal{})
	gob.Register(make([]authc.Principal, 0))
}
