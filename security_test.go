// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"testing"

	"aahframework.org/security.v0/session"
	"aahframework.org/test.v0/assert"
)

func TestSecuritySessionStore(t *testing.T) {
	err := AddSessionStore("file", &session.FileStore{})
	assert.NotNil(t, err)
	assert.Equal(t, "session: store name 'file' is already added, skip it", err.Error())

	err = AddSessionStore("custom", nil)
	assert.NotNil(t, err)
	assert.Equal(t, "session: store value is nil", err.Error())
}

func TestSecuritySessionTemplateFuns(t *testing.T) {
	viewArgs := make(map[string]interface{})

	assert.Nil(t, viewArgs[keySessionValues])

	bv1 := tmplSessionValue(viewArgs, "my-testvalue")
	assert.Nil(t, bv1)

	bv2 := tmplFlashValue(viewArgs, "my-flashvalue")
	assert.Nil(t, bv2)

	session := &session.Session{Values: make(map[string]interface{})}
	session.Set("my-testvalue", 38458473684763)
	session.SetFlash("my-flashvalue", "user not found")

	viewArgs[keySessionValues] = session
	assert.NotNil(t, viewArgs[keySessionValues])

	v1 := tmplSessionValue(viewArgs, "my-testvalue")
	assert.Equal(t, 38458473684763, v1)

	v2 := tmplFlashValue(viewArgs, "my-flashvalue")
	assert.Equal(t, "user not found", v2)

	v3 := tmplIsAuthenticated(viewArgs)
	assert.False(t, v3)
}
