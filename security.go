// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

// Package security houses all the application security implementation (Session,
// Basic Auth, Token Auth, CORS, CSRF, Security Headers, etc.) by aah framework.
package security

import (
	"fmt"

	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
	"aahframework.org/security.v0/session"
)

// Version is aah framework security library version no.
const Version = "0.5"

// Security is holds the security management implementation.
type Security struct {
	SessionManager *session.Manager
	configPath     string
	config         *config.Config
	appCfg         *config.Config
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Global methods
//___________________________________

// New method initialize the application security configuration `security { ... }`.
// Which is mainly Session, CORS, CSRF, Security Headers, etc.
func New(configPath string, appCfg *config.Config) (*Security, error) {
	if !ess.IsFileExists(configPath) {
		return nil, fmt.Errorf("security: configuration does not exists: %v", configPath)
	}

	var err error
	s := &Security{configPath: configPath, appCfg: appCfg}
	if s.config, err = config.LoadFile(s.configPath); err != nil {
		return nil, err
	}

	isSessionConfigExists := s.config.IsExists("security.session")
	log.Debugf("Session config exists: %v", isSessionConfigExists)
	if s.SessionManager, err = session.NewManager(s.config); err != nil {
		return nil, err
	}

	return s, nil
}
