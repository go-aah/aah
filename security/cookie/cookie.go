// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package cookie

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"aahframework.org/essentials.v0"
	"aahframework.org/security.v0/acrypto"
)

// Cookie errors
var (
	ErrCookieValueIsTooLarge    = errors.New("security/cookie: value is greater than 4096")
	ErrCookieValueIsInvalid     = errors.New("security/cookie: value is not valid")
	ErrCookieInvaildTimestamp   = errors.New("security/cookie: timestamp is invalid")
	ErrCookieTimestampIsTooNew  = errors.New("security/cookie: timestamp is too new")
	ErrCookieTimestampIsExpired = errors.New("security/cookie: timestamp expried")
	ErrSignVerificationIsFailed = errors.New("security/cookie: sign verification is failed")
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package methods
//___________________________________

// NewManager method returns the new cookie manager.
func NewManager(opts *Options, signKey, encKey string) (*Manager, error) {
	m := &Manager{Options: opts, maxCookieSize: 4096, sha: "sha-256"}

	// Sign key
	m.isSignKey = !ess.IsStrEmpty(signKey)
	if m.isSignKey {
		m.signKey = []byte(signKey)
	}

	// Enc key
	var err error
	m.isEncKey = !ess.IsStrEmpty(encKey)
	if m.isEncKey {
		m.encKey = []byte(encKey)
		if m.cipherBlock, err = aes.NewCipher(m.encKey); err != nil {
			return nil, err
		}
	}

	return m, nil
}

// NewWithOptions method returns http.Cookie with the options set from
// `session {...}`. It also sets the `Expires` field calculated based on the
// MaxAge value.
func NewWithOptions(value string, opts *Options) *http.Cookie {
	cookie := &http.Cookie{
		Name:     opts.Name,
		Value:    value,
		Path:     opts.Path,
		Domain:   opts.Domain,
		MaxAge:   int(opts.MaxAge),
		Secure:   opts.Secure,
		HttpOnly: opts.HTTPOnly,
	}

	if opts.MaxAge > 0 {
		d := time.Duration(opts.MaxAge) * time.Second
		cookie.Expires = time.Now().Add(d)
	} else if opts.MaxAge < 0 {
		// Set it to the past to expire now.
		cookie.Expires = time.Unix(1, 0)
	}

	return cookie
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Cookie Manager
//___________________________________

// Manager struct used to manage and process secure cookie.
type Manager struct {
	Options *Options

	isSignKey     bool
	signKey       []byte
	sha           string
	isEncKey      bool
	encKey        []byte
	cipherBlock   cipher.Block
	maxCookieSize int
}

// Options to hold session cookie options.
type Options struct {
	Name     string
	Domain   string
	Path     string
	MaxAge   int64
	HTTPOnly bool
	Secure   bool
	SameSite string
}

// New method creates new cookie instance for given value with cookie manager options.
func (m *Manager) New(value string) *http.Cookie {
	return NewWithOptions(value, m.Options)
}

// Write method writes the given cookie value into response.
func (m *Manager) Write(w http.ResponseWriter, value string) {
	c := m.New(value)
	if v := c.String(); !ess.IsStrEmpty(v) {
		// Adding `SameSite` setting
		// https://tools.ietf.org/html/draft-west-first-party-cookies-07
		//
		// Currently Go doesn't have this attribute in `http.Cookie`, for future proof
		// check and then add `SameSite` setting.
		if !strings.Contains(v, "SameSite") && !ess.IsStrEmpty(m.Options.SameSite) {
			v += "; SameSite=" + m.Options.SameSite
		}
		w.Header().Add("Set-Cookie", v)
	}
}

// Encode method encodes given value.
//
// It performs:
//   1) Encrypts it if encryption key configured
//   2) Signs the value if sign key configured
//   3) Encodes value into Base64 string
//   4) Checks max cookie size i.e 4Kb
func (m *Manager) Encode(b []byte) (string, error) {
	// Encrypt it
	if m.isEncKey {
		b = acrypto.AESEncrypt(m.cipherBlock, b)
	}

	// Encode it
	b = ess.EncodeToBase64(b)

	// compose value of "name|date|value". Pipe is used while Decode
	b = []byte(fmt.Sprintf("%s|%d|%s|", m.Options.Name, currentTimestamp(), b))

	// Sign it if enabled
	if m.isSignKey {
		signed := acrypto.Sign(m.signKey, b[:len(b)-1], m.sha)

		// Append signed value
		b = append(b, signed...)
	}

	// Remove name
	b = b[len(m.Options.Name)+1:]

	// Encode to base64
	b = ess.EncodeToBase64(b)

	// Check cookie max size.
	if len(b) > m.maxCookieSize {
		return "", ErrCookieValueIsTooLarge
	}

	return string(b), nil
}

// Decode method decodes the secure cookie value.
//
// It performs:
//   1) Checks max cookie size i.e 4Kb
//   2) Decodes the value using Base64
//   3) Validates the signed data
//   4) Validates timestamp
//   5) Decodes the value using Base64
//   6) Decrypts the value
func (m *Manager) Decode(value string) ([]byte, error) {
	// Check cookie max size.
	if len(value) > m.maxCookieSize {
		return nil, ErrCookieValueIsTooLarge
	}

	// Decode base64
	b, err := ess.DecodeBase64([]byte(value))
	if err != nil {
		return nil, err
	}

	// Check value parts, value is "date|value|signed-data"
	parts := bytes.SplitN(b, []byte("|"), 3)
	if len(parts) != 3 {
		return nil, ErrCookieValueIsInvalid
	}

	b = append([]byte(m.Options.Name+"|"), b[:len(b)-len(parts[2])-1]...)

	// Verify signed data, if enabled
	if m.isSignKey {
		if !acrypto.Verify(m.signKey, b, parts[2], m.sha) {
			return nil, ErrSignVerificationIsFailed
		}
	}

	// Verify timestamp
	var t1 int64
	if t1, err = strconv.ParseInt(string(parts[0]), 10, 64); err != nil {
		return nil, ErrCookieInvaildTimestamp
	}
	t2 := currentTimestamp()
	if t1 > t2 {
		return nil, ErrCookieTimestampIsTooNew
	}
	if m.Options.MaxAge != 0 && t1 < t2-m.Options.MaxAge {
		return nil, ErrCookieTimestampIsExpired
	}

	// Decrypt it
	b, err = ess.DecodeBase64(parts[1])
	if err != nil {
		return nil, err
	}
	if m.isEncKey {
		if b, err = acrypto.AESDecrypt(m.cipherBlock, b); err != nil {
			return nil, err
		}
	}

	return b, nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

// currentTimestamp method return current UTC time in unix format.
func currentTimestamp() int64 {
	return time.Now().UTC().Unix()
}
