// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/session source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package session

import (
	"net/http"
	"time"
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

// newCookie method returns http.Cookie with the options set from
// `session {...}`. It also sets the `Expires` field calculated based on the
// MaxAge value, for Internet Explorer compatibility.
func newCookie(value string, opts *Options) *http.Cookie {
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
