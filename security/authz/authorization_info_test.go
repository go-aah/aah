// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package authz

import (
	"testing"

	"aahframework.org/test.v0/assert"
)

func TestAuthAuthorizationRoles(t *testing.T) {
	a1 := NewAuthorizationInfo()
	a1.AddRole("role1", "role2", "role3", "role4")
	assert.True(t, a1.HasRole("role3"))
	assert.True(t, a1.HasAnyRole("one", "two", "role2"))
	assert.True(t, a1.HasAllRoles("role1", "role3", "role4"))
	assert.False(t, a1.HasRole("one"))
	assert.False(t, a1.HasAnyRole("one", "two", "three"))
	assert.False(t, a1.HasAllRoles("one", "two", "three"))
	assert.Equal(t, "authorizationinfo(roles(role1, role2, role3, role4) allpermissions())", a1.String())
}

func TestAuthAuthorizationPermissions(t *testing.T) {
	a1 := NewAuthorizationInfo()
	a1.AddPermissionString("newsletter:*:*")
	assert.True(t, a1.IsPermitted("newsletter:*:read"))
	assert.True(t, a1.IsPermittedAll("newsletter:read,write", "newsletter:*:read"))
	assert.True(t, a1.IsPermitted("newsletter:123:read:write"))
	assert.Equal(t, "authorizationinfo(roles() allpermissions(permission(newsletter:*:*)))", a1.String())

	p1, _ := NewPermission("newsletter:read,write")
	p2, _ := NewPermission("newsletter:*:read")
	assert.True(t, a1.IsPermittedAllp(p1, p2))

	p3, _ := NewPermission("one")
	assert.False(t, a1.IsPermittedAllp(p1, p2, p3))

	a2 := NewAuthorizationInfo()
	a2.AddPermissionString("newsletter:read")
	assert.True(t, a2.IsPermitted("newsletter:read"))
	assert.False(t, a2.IsPermitted(""))
	assert.False(t, a2.IsPermitted("newsletter:*:read"))
	assert.False(t, a2.IsPermitted("newsletter:write"))
	assert.False(t, a2.IsPermittedAll("newsletter:read", "newsletter:write"))
}
