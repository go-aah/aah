// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package auth

import "strings"

// AuthorizationInfo ...
type AuthorizationInfo struct {
	roles       parts
	permissions []*Permission
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Global methods
//___________________________________

// NewAuthorizationInfo ...
func NewAuthorizationInfo() *AuthorizationInfo {
	return &AuthorizationInfo{
		roles:       make(parts, 0),
		permissions: make([]*Permission, 0),
	}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// AuthorizationInfo methods
//___________________________________

// AddRoles ...
func (a *AuthorizationInfo) AddRoles(roles ...string) *AuthorizationInfo {
	a.roles = append(a.roles, roles...)
	return a
}

// AddPermissions ...
func (a *AuthorizationInfo) AddPermissions(permissions ...*Permission) *AuthorizationInfo {
	a.permissions = append(a.permissions, permissions...)
	return a
}

// AddStringPermissions ...
func (a *AuthorizationInfo) AddStringPermissions(permissions ...string) *AuthorizationInfo {
	for _, ps := range permissions {
		p, _ := NewPermission(ps)
		a.AddPermissions(p)
	}
	return a
}

// HasRole ...
func (a *AuthorizationInfo) HasRole(role string) bool {
	for _, r := range a.roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasAllRoles ...
func (a *AuthorizationInfo) HasAllRoles(roles ...string) bool {
	return a.roles.ContainsAll(roles)
}

// HasAnyRole ...
func (a *AuthorizationInfo) HasAnyRole(roles ...string) bool {
	return a.roles.ContainsAny(roles)
}

// IsPermitted ...
func (a *AuthorizationInfo) IsPermitted(permission string) bool {
	p, _ := NewPermission(permission)
	return a.IsPermittedp(p)
}

// IsPermittedAll ...
func (a *AuthorizationInfo) IsPermittedAll(permissions ...string) bool {
	for _, permission := range permissions {
		p, _ := NewPermission(permission)
		if !a.IsPermittedp(p) {
			return false
		}
	}
	return true
}

// IsPermittedp ...
func (a *AuthorizationInfo) IsPermittedp(permission *Permission) bool {
	if permission == nil {
		return false
	}

	for _, rp := range a.permissions {
		if rp.Implies(permission) {
			return true
		}
	}
	return false
}

// IsPermittedAllp ...
func (a *AuthorizationInfo) IsPermittedAllp(permissions ...*Permission) bool {
	for _, permission := range permissions {
		if !a.IsPermittedp(permission) {
			return false
		}
	}
	return true
}

// String ...
func (a *AuthorizationInfo) String() string {
	var str string
	if len(a.roles) > 0 {
		str += "Roles[" + strings.Join(a.roles, "|") + "]"
	} else {
		str += "Roles[]"
	}

	if len(a.permissions) > 0 {
		var ps []string
		for _, p := range a.permissions {
			ps = append(ps, p.String())
		}
		str += " Permissions[" + strings.Join(ps, "|") + "]"
	} else {
		str += " Permissions[]"
	}

	return str
}
