// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package authc

import (
	"testing"

	"aahframework.org/test.v0/assert"
)

func TestAuthcAuthenticationToken(t *testing.T) {
	authToken := &AuthenticationToken{
		Scheme:     "form",
		Identity:   "jeeva",
		Credential: "welcome123",
	}

	assert.Equal(t, "authenticationtoken(scheme:form identity:jeeva credential:*******)", authToken.String())
}
