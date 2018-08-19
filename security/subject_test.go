// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package security

import (
	"strings"
	"testing"

	"aahframework.org/config.v0"
	"aahframework.org/security.v0/authc"
	"aahframework.org/security.v0/authz"
	"aahframework.org/security.v0/session"
	"aahframework.org/test.v0/assert"
)

func TestSecuritySubject(t *testing.T) {
	authcInfo := authc.NewAuthenticationInfo()
	authcInfo.Principals = append(authcInfo.Principals, &authc.Principal{Value: "user@sample.com", IsPrimary: true})

	authzInfo := authz.NewAuthorizationInfo().
		AddRole("role1", "role2", "role3", "role4").
		AddPermissionString("newsletter:read,write")

	cfg, _ := config.ParseString(`
		security {
				session {
			}
		}
		`)
	sessionManager, err := session.NewManager(cfg)
	assert.FailNowOnError(t, err, "unexpected")

	sub := AcquireSubject()
	sub.AuthenticationInfo = authcInfo
	sub.AuthorizationInfo = authzInfo
	sub.Session = sessionManager.NewSession()
	sub.Session.IsAuthenticated = true

	// AuthenticationInfo
	p := sub.PrimaryPrincipal()
	assert.NotNil(t, p)
	assert.Equal(t, "user@sample.com", p.Value)
	assert.True(t, p.IsPrimary)
	assert.Equal(t, "principal(realm: isprimary:true claim: value:user@sample.com)", p.String())

	all := sub.AllPrincipals()
	assert.NotNil(t, all)
	assert.True(t, len(all) == 1)

	//AuthorizationInfo
	assert.True(t, sub.IsPermitted("newsletter:read"))
	assert.True(t, sub.IsPermittedAll("newsletter:read", "newsletter:write"))
	assert.True(t, sub.HasRole("role3"))
	assert.True(t, sub.HasAnyRole("one", "two", "role2"))
	assert.True(t, sub.HasAllRoles("role1", "role3", "role4"))
	assert.False(t, sub.HasRole("one"))
	assert.False(t, sub.HasAnyRole("one", "two", "three"))
	assert.False(t, sub.HasAllRoles("one", "two", "three"))

	str := sub.String()
	assert.True(t, strings.Contains(str, "user@sample.com"))
	assert.True(t, strings.Contains(str, "role1, role2, role3, role4"))
	assert.True(t, strings.Contains(str, "newsletter:read,write"))

	// Session
	assert.True(t, sub.IsAuthenticated())
	sub.Logout()
	sub.Reset()
	assert.False(t, sub.IsAuthenticated())

	str = sub.String()
	assert.Equal(t, "<nil>, <nil>, <nil>", str)

	ReleaseSubject(sub)
}
