// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package cookie

import (
	"net/http"
	"time"
)

type (
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
// Package methods
//___________________________________

// NewWithOptions method returns http.Cookie with the options set from
// `session {...}`. It also sets the `Expires` field calculated based on the
// MaxAge value, for Internet Explorer compatibility.
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
