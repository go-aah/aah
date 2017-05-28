// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package auth

import "errors"

var (
	// ErrInvalidCredentials returned when given authentication token doesn't prove
	// to be valid subject against authenticating realm.
	ErrInvalidCredentials = errors.New("security: invalid credentials")
)
