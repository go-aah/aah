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
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
)

// Cookie errors
var (
	ErrSessionStoreIsNil        = errors.New("session: store value is nil")
	ErrCookieValueIsTooLarge    = errors.New("session: cookie value is greater than 4096")
	ErrCookieValueIsInvalid     = errors.New("session: cookie value is not valid")
	ErrCookieInvaildTimestamp   = errors.New("session: cookie timestamp is invalid")
	ErrCookieTimestampIsTooNew  = errors.New("session: cookie timestamp is too new")
	ErrCookieTimestampIsExpired = errors.New("session: cookie timestamp expried")
	ErrSignVerificationIsFailed = errors.New("session: sign verification is failed")
	ErrUnableToDecrypt          = errors.New("session: given value unable to decrypt")
	ErrBase64Decode             = errors.New("session: base64 decode error")

	registerStores = make(map[string]Storer)

	sessionPool = sync.Pool{New: func() interface{} { return &Session{Values: make(map[string]interface{})} }}
)

type (
	// Storer is interface for implementing pluggable storage implementation.
	Storer interface {
		Init(appCfg *config.Config) error
		Read(id string) string
		Save(id, value string) error
		Delete(id string) error
		IsExists(id string) bool
		Cleanup(m *Manager)
	}

	// Manager is a session manager to manage sessions.
	Manager struct {
		cfg     *config.Config
		Options *Options

		mode            string
		store           Storer
		storeName       string
		isSignKey       bool
		signKey         []byte
		isEncKey        bool
		encKey          []byte
		cipBlock        cipher.Block
		idLength        int
		maxCookieSize   int
		cleanupInterval int64
	}

	// Options to hold session cookie options.
	Options struct {
		Name     string
		Domain   string
		Path     string
		MaxAge   int64
		HTTPOnly bool
		Secure   bool
	}
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Global methods
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
	m := &Manager{cfg: appCfg}

	// Session mode
	m.mode = m.cfg.StringDefault("security.session.mode", "stateless")

	// Store
	m.storeName = m.cfg.StringDefault("security.session.store.type", "cookie")
	if m.storeName != "cookie" {
		store, found := registerStores[m.storeName]
		if !found {
			return nil, fmt.Errorf("session: store name '%v' not exists", m.storeName)
		}
		m.store = store
		if err := m.store.Init(m.cfg); err != nil {
			return nil, err
		}
	}

	// Sign key
	signKey := m.cfg.StringDefault("security.session.sign_key", "")
	m.isSignKey = !ess.IsStrEmpty(signKey)
	if m.isSignKey {
		m.signKey = []byte(signKey)
	}

	// Enc key
	var err error
	encKey := m.cfg.StringDefault("security.session.enc_key", "")
	m.isEncKey = !ess.IsStrEmpty(encKey)
	if m.isEncKey {
		m.encKey = []byte(encKey)
		if m.cipBlock, err = aes.NewCipher(m.encKey); err != nil {
			return nil, err
		}
	}

	m.idLength = m.cfg.IntDefault("security.session.id_length", 32)

	// Cookie Options
	m.Options = &Options{
		Name:     m.cfg.StringDefault("security.session.prefix", "aah") + "_session",
		Domain:   m.cfg.StringDefault("security.session.domain", ""),
		Path:     m.cfg.StringDefault("security.session.path", "/"),
		HTTPOnly: m.cfg.BoolDefault("security.session.http_only", true),
		Secure:   m.cfg.BoolDefault("security.session.secure", true),
	}

	// TTL value
	ttl := m.cfg.StringDefault("security.session.ttl", "")
	if ess.IsStrEmpty(ttl) {
		m.Options.MaxAge = 0
	} else {
		if m.Options.MaxAge, err = toSeconds(ttl); err != nil {
			return nil, err
		}
	}

	// Cleanup
	if m.cleanupInterval, err = toSeconds(m.cfg.StringDefault("security.session.cleanup_interval", "30m")); err != nil {
		return nil, err
	}

	// Maximum cookie is 4Kb = 4096 bytes
	m.maxCookieSize = 4096

	// Schedule cleanup
	if !m.IsCookieStore() {
		time.AfterFunc(time.Duration(m.cleanupInterval)*time.Second, func() {
			log.Info("Running expired session cleanup at %v", time.Now())
			m.store.Cleanup(m)
		})
	}

	return m, nil
}

// NewSession method creates a new session for the request.
func (m *Manager) NewSession() *Session {
	s := sessionPool.Get().(*Session)
	s.ID = ess.RandomString(m.idLength)
	s.IsNew = true
	t := time.Now()
	s.CreatedTime = &t
	return s
}

// GetSession method returns the session for given request instance otherwise
// it returns nil.
func (m *Manager) GetSession(r *http.Request) *Session {
	cookie, err := r.Cookie(m.Options.Name)
	if err == http.ErrNoCookie {
		log.Debug("aah application session is not yet created or unavailable")
		return nil
	}

	encodedStr := cookie.Value
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
		if err == ErrCookieTimestampIsExpired && !m.IsCookieStore() {
			if id, err := m.DecodeToString(cookie.Value); err == nil {
				log.Info("Cleaning expried session: %v", id)
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
		encodedStr, err = m.Encode(m.Options.Name, s)
	} else {
		encodedStr, err = m.Encode(m.Options.Name, s.ID)
	}
	// any error while encode
	if err != nil {
		return err
	}

	if !m.IsCookieStore() {
		// Encode session object send it to store
		encoded, err := m.Encode(m.Options.Name, s)
		if err != nil {
			return err
		}
		if err = m.store.Save(s.ID, encoded); err != nil {
			return err
		}
	}

	log.Debugf("Session saved, ID: %v", s.ID)
	http.SetCookie(w, newCookie(encodedStr, m.Options))
	return nil
}

// DeleteSession method deletes the session from store and sets deletion
// for browser cookie.
func (m *Manager) DeleteSession(w http.ResponseWriter, s *Session) error {
	if !m.IsCookieStore() {
		if err := m.store.Delete(s.ID); err != nil {
			// store delete had error, log it and go forward to clean the cookie
			log.Error(err)
		}
	}

	opts := *m.Options
	opts.MaxAge = -1
	log.Debugf("Session deleted, ID: %v", s.ID)
	http.SetCookie(w, newCookie("", &opts))
	return nil
}

// DecodeToString method decodes the encoded string into original string.
func (m *Manager) DecodeToString(encodedStr string) (string, error) {
	var id string
	if err := m.Decode(m.Options.Name, encodedStr, &id); err != nil {
		return "", err
	}
	return id, nil
}

// DecodeToSession method decodes the encoded string into session object.
func (m *Manager) DecodeToSession(encodedStr string) (*Session, error) {
	var session Session
	if err := m.Decode(m.Options.Name, encodedStr, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

// Encode method encodes given value with name.
//
// It performs:
//   1) Encodes the value using `Gob`
//   2) Encrypts it if encryption key `session.enc_key` configured
//   3) Signs the value if sign key `session.sign_key` configured
//   4) Encodes value into Base64 string
//   5) Checks max cookie size i.e 4Kb
func (m *Manager) Encode(name string, value interface{}) (string, error) {
	var b []byte
	if bytes, err := toBytes(value); err == nil {
		b = bytes
	} else {
		return "", err
	}

	// Encrypt it
	if m.isEncKey {
		b = encrypt(m.cipBlock, b)
	}

	// Encode and sign it
	b = encodeBase64(b)

	// compose value of "name|date|value". Pipe is used while Decode
	b = []byte(fmt.Sprintf("%s|%d|%s|", name, currentTimestamp(), b))

	// Sign it if enabled
	if m.isSignKey {
		signed := sign(m.signKey, b[:len(b)-1])

		// Append signed value
		b = append(b, signed...)
	}

	// Remove name
	b = b[len(name)+1:]

	// Encode to base64
	b = encodeBase64(b)

	// Check cookie max size.
	if len(b) > m.maxCookieSize {
		return "", ErrCookieValueIsTooLarge
	}

	return string(b), nil
}

// Decode method decodes given value with name.
//
// It performs:
//   1) Checks max cookie size i.e 4Kb
//   2) Decodes the value using Base64
//   3) Validates the signed data
//   4) Validates timestamp
//   5) Decrypts the value
//   6) Decode into result object using `Gob`
func (m *Manager) Decode(name, value string, dst interface{}) error {
	// Check cookie max size.
	if len(value) > m.maxCookieSize {
		return ErrCookieValueIsTooLarge
	}

	// Decode base64
	b, err := decodeBase64([]byte(value))
	if err != nil {
		return err
	}

	// Check value parts, value is "date|value|signed-data"
	parts := bytes.SplitN(b, []byte("|"), 3)
	if len(parts) != 3 {
		return ErrCookieValueIsInvalid
	}

	b = append([]byte(name+"|"), b[:len(b)-len(parts[2])-1]...)

	// Verify signed data, if enabled
	if m.isSignKey {
		if !verify(m.signKey, b, parts[2]) {
			return ErrSignVerificationIsFailed
		}
	}

	// Verify timestamp
	var t1 int64
	if t1, err = strconv.ParseInt(string(parts[0]), 10, 64); err != nil {
		return ErrCookieInvaildTimestamp
	}
	t2 := currentTimestamp()
	if t1 > t2 {
		return ErrCookieTimestampIsTooNew
	}
	if m.Options.MaxAge != 0 && t1 < t2-m.Options.MaxAge {
		return ErrCookieTimestampIsExpired
	}

	// Decrypt it
	b, err = decodeBase64(parts[1])
	if err != nil {
		return err
	}
	if m.isEncKey {
		if b, err = decrypt(m.cipBlock, b); err != nil {
			return err
		}
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
