// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

// Package session provides HTTP state management library for aah framework.
// Default store is `Cookie` and framework provides `FileStore` and extensible
// `session.Storer` interface. Using store interface you can write any key-value
// Database, NoSQL Database, and RDBMS for storing encoded session data.
//
// Features:
//  - Extensible session store interface
//  - Signed session data
//  - Encrypted session data
//
// Non-cookie store session data is maintained via store interface. Only Session ID
// is transmitted over the wire in the Cookie. Please refer `session.FileStore` for
// sample, its very easy.
//
// If you would like to store custom types in session then Register your custom
// types using `gob.Register(...)`.
//
// Secure cookie code is inspired from Gorilla secure cookie library.
//
// Know more: https://www.owasp.org/index.php/Session_Management_Cheat_Sheet
package session

import (
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
	"aahframework.org/security.v0/cookie"
)

var (
	// ErrSessionStoreIsNil returned when suppiled store is nil.
	ErrSessionStoreIsNil = errors.New("security/session: store value is nil")

	registerStores = make(map[string]Storer)
	sessionPool    = sync.Pool{New: func() interface{} { return &Session{Values: make(map[string]interface{})} }}
)

// Storer is interface for implementing pluggable storage implementation.
type Storer interface {
	Init(appCfg *config.Config) error
	Read(id string) string
	Save(id, value string) error
	Delete(id string) error
	IsExists(id string) bool
	Cleanup(m *Manager)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package methods
//___________________________________

// AddStore method allows you to add user created session store
// for aah framework application.
func AddStore(name string, store Storer) error {
	if store == nil {
		return ErrSessionStoreIsNil
	}

	if _, found := registerStores[name]; found {
		return fmt.Errorf("session: store name '%v' is already added, skip it", name)
	}

	registerStores[name] = store
	return nil
}

// NewManager method initializes the session manager and store based on
// configuration from aah.conf section `session { ... }`.
func NewManager(appCfg *config.Config) (*Manager, error) {
	var err error
	m := &Manager{cfg: appCfg}
	keyPrefix := "security.session"

	// Session mode
	m.mode = m.cfg.StringDefault(keyPrefix+".mode", "stateless")

	// Store
	m.storeName = m.cfg.StringDefault(keyPrefix+".store.type", "cookie")
	if m.storeName != "cookie" {
		store, found := registerStores[m.storeName]
		if !found {
			return nil, fmt.Errorf("session: store name '%v' not exists", m.storeName)
		}
		m.store = store
		if err = m.store.Init(m.cfg); err != nil {
			return nil, err
		}
	}

	m.idLength = m.cfg.IntDefault(keyPrefix+".id_length", 32)

	// Cookie Options
	opts := &cookie.Options{
		Name:     m.cfg.StringDefault(keyPrefix+".prefix", "aah") + "_session",
		Domain:   m.cfg.StringDefault(keyPrefix+".domain", ""),
		Path:     m.cfg.StringDefault(keyPrefix+".path", "/"),
		HTTPOnly: m.cfg.BoolDefault(keyPrefix+".http_only", true),
		// Based on aah server SSL configuration `http.Cookie.Secure` value is set
		Secure: m.cfg.BoolDefault("server.ssl.enable", false),
	}

	// TTL value
	if opts.MaxAge, err = toSeconds(m.cfg.StringDefault(keyPrefix+".ttl", "0m")); err != nil {
		return nil, err
	}

	m.cookieMgr, err = cookie.NewManager(opts,
		m.cfg.StringDefault(keyPrefix+".sign_key", ""),
		m.cfg.StringDefault(keyPrefix+".enc_key", ""))
	if err != nil {
		return nil, err
	}

	// Cleanup
	if m.cleanupInterval, err = toSeconds(m.cfg.StringDefault(keyPrefix+".cleanup_interval", "30m")); err != nil {
		return nil, err
	}

	// Schedule cleanup
	if !m.IsCookieStore() {
		time.AfterFunc(time.Duration(m.cleanupInterval)*time.Second, func() {
			log.Infof("Running expired session cleanup at %v", time.Now())
			m.store.Cleanup(m)
		})
	}

	return m, nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Session Manager
//___________________________________

// Manager is a session manager to manage sessions.
type Manager struct {
	cfg             *config.Config
	mode            string
	store           Storer
	storeName       string
	cookieMgr       *cookie.Manager
	idLength        int
	cleanupInterval int64
}

// NewSession method creates a new session for the request.
func (m *Manager) NewSession() *Session {
	s := sessionPool.Get().(*Session)
	s.ID = ess.SecureRandomString(m.idLength)
	s.IsNew = true
	t := time.Now()
	s.CreatedTime = &t
	return s
}

// GetSession method returns the session for given request instance otherwise
// it returns nil.
func (m *Manager) GetSession(r *http.Request) *Session {
	scookie, err := r.Cookie(m.cookieMgr.Options.Name)
	if err == http.ErrNoCookie {
		return nil
	}

	encodedStr := scookie.Value
	if !m.IsCookieStore() {
		if id, er := m.DecodeToString(encodedStr); er == nil {
			encodedStr = m.store.Read(id)
		} else {
			log.Error(err)
			return nil
		}
	}

	if ess.IsStrEmpty(encodedStr) {
		return nil
	}

	session, err := m.DecodeToSession(encodedStr)
	if err != nil {
		log.Error(err)

		// clean expried session
		if err == cookie.ErrCookieTimestampIsExpired && !m.IsCookieStore() {
			if id, err := m.DecodeToString(scookie.Value); err == nil {
				log.Debugf("Cleaning expried session: %s", id)
				_ = m.store.Delete(id)
			}
		}
		return nil
	}

	session.IsNew = false
	return session
}

// SaveSession method saves the given session into store.
// Add writes the cookie into response.
func (m *Manager) SaveSession(w http.ResponseWriter, s *Session) error {
	if s.maxAge == -1 {
		return m.DeleteSession(w, s)
	}

	var (
		encodedStr string
		err        error
	)

	if m.IsCookieStore() {
		encodedStr, err = m.Encode(s)
	} else {
		encodedStr, err = m.Encode(s.ID)
	}
	// any error while encode
	if err != nil {
		return err
	}

	if !m.IsCookieStore() {
		// Encode session object send it to store
		encoded, err := m.Encode(s)
		if err != nil {
			return err
		}
		if err = m.store.Save(s.ID, encoded); err != nil {
			return err
		}
	}

	m.cookieMgr.Write(w, encodedStr)
	return nil
}

// DeleteSession method deletes the session from store and sets deletion
// for browser cookie.
func (m *Manager) DeleteSession(w http.ResponseWriter, s *Session) error {
	if !m.IsCookieStore() {
		if err := m.store.Delete(s.ID); err != nil {
			// store delete had an error, log it and go forward to clean the cookie
			log.Error(err)
		}
	}

	opts := *m.cookieMgr.Options
	opts.MaxAge = -1
	http.SetCookie(w, cookie.NewWithOptions("", &opts))
	return nil
}

// DecodeToString method decodes the encoded string into original string.
func (m *Manager) DecodeToString(encodedStr string) (string, error) {
	var id string
	if err := m.Decode(encodedStr, &id); err != nil {
		return "", err
	}
	return id, nil
}

// DecodeToSession method decodes the encoded string into session object.
func (m *Manager) DecodeToSession(encodedStr string) (*Session, error) {
	var session Session
	if err := m.Decode(encodedStr, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

// Encode method encodes given value with name.
//
// It performs:
//   1) Encodes the value using `Gob`
//   2) Encodes value into Base64 (encrypt, sign, cookie size check)
func (m *Manager) Encode(value interface{}) (string, error) {
	b, err := toBytes(value)
	if err != nil {
		return "", err
	}
	return m.cookieMgr.Encode(b)
}

// Decode method decodes given value with name.
//
// It performs:
//   1) Decrypts the value (size check, decode base64, sign verify, timestamp verify, decrypt)
//   2) Decode into result object using `Gob`
func (m *Manager) Decode(value string, dst interface{}) error {
	b, err := m.cookieMgr.Decode(value)
	if err != nil {
		return err
	}
	return decodeGob(dst, b)
}

// IsStateful methdo returns true if session mode is stateful otherwise false.
func (m *Manager) IsStateful() bool {
	return m.mode == "stateful"
}

// IsCookieStore method returns true if session store is cookie otherwise false.
func (m *Manager) IsCookieStore() bool {
	return m.storeName == "cookie"
}

// ReleaseSession method puts session object back to pool.
func ReleaseSession(s *Session) {
	if s != nil {
		s.Reset()
		sessionPool.Put(s)
	}
}
