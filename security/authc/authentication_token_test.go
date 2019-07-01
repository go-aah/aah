// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package authc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuthcAuthenticationToken(t *testing.T) {
	authToken := &AuthenticationToken{
		Scheme:     "form",
		Identity:   "jeeva",
		Credential: "welcome123",
		Values: map[string]interface{}{
			"key1": "value 1",
			"key2": "value 2",
		},
	}

	assert.Equal(t, "authenticationtoken(scheme:form identity:jeeva credential:*******, values:map[key1:value 1 key2:value 2])", authToken.String())
}
