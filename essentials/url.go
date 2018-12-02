// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ess

import "net/url"

// IsVaildURL method returns true if given raw URL gets parsed without any errors
// otherwise false.
func IsVaildURL(rawurl string) bool {
	_, err := url.Parse(rawurl)
	return err == nil
}

// IsRelativeURL method returns true if given raw URL is relative URL otherwise false.
func IsRelativeURL(rawurl string) bool {
	return !IsAbsURL(rawurl)
}

// IsAbsURL method returns true if given raw URL is absolute URL otherwise false.
func IsAbsURL(rawurl string) bool {
	u, err := url.Parse(rawurl)
	if err != nil {
		return false
	}
	return u.IsAbs()
}
