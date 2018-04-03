// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package authz

import "strings"

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package methods
//___________________________________

// NewAuthorizationInfo method creates an `AuthorizationInfo`
// instance with zero values. Use the returned instance to add roles and
// permissions for the Subject (aka User).
func NewAuthorizationInfo() *AuthorizationInfo {
	return &AuthorizationInfo{
		roles:       make(parts, 0),
		permissions: make([]*Permission, 0),
	}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// AuthorizationInfo
//___________________________________

// AuthorizationInfo struct holds the information of Subject authorization.
// It performs authorization (access control) operations for any given Subject
// (aka 'application user').
//
// Note that you can add and evaluate Permissions using string and instance.
// aah framework by default implementations do String-to-Permission conversion.
//
// These string methods do forego type-safety for the benefit of convenience and
// simplicity, so you should choose which ones to use based on your preferences
// and needs.
type AuthorizationInfo struct {
	roles       parts
	permissions []*Permission
}

// AddRole method assigns a multiple-role to those associated with the account.
func (a *AuthorizationInfo) AddRole(roles ...string) *AuthorizationInfo {
	a.roles = append(a.roles, roles...)
	return a
}

// AddPermission method assigns a permission to those directly associated with
// the account.
func (a *AuthorizationInfo) AddPermission(permissions ...*Permission) *AuthorizationInfo {
	a.permissions = append(a.permissions, permissions...)
	return a
}

// AddPermissionString method assigns multiple permissions to those associated
// directly with the account.
func (a *AuthorizationInfo) AddPermissionString(permissions ...string) *AuthorizationInfo {
	for _, ps := range permissions {
		p, _ := NewPermission(ps)
		a.AddPermission(p)
	}
	return a
}

// HasRole method returns true if the corresponding Subject/user has the
// specified role, false otherwise.
func (a *AuthorizationInfo) HasRole(role string) bool {
	for _, r := range a.roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasAllRoles method returns true if the corresponding Subject/user has all of
// the specified roles, false otherwise.
func (a *AuthorizationInfo) HasAllRoles(roles ...string) bool {
	return a.roles.ContainsAll(roles)
}

// HasAnyRole method returns true if the corresponding Subject/user has any-one
// of the specified roles, false otherwise.
func (a *AuthorizationInfo) HasAnyRole(roles ...string) bool {
	return a.roles.ContainsAny(roles)
}

// IsPermitted method returns true if the corresponding subject/user is permitted
// to perform an action or access a resource summarized by the specified
// permission string.
func (a *AuthorizationInfo) IsPermitted(permission string) bool {
	p, _ := NewPermission(permission)
	return a.IsPermittedp(p)
}

// IsPermittedAll method returns true if the corresponding Subject/user implies
// all of the specified permission strings, false otherwise.
func (a *AuthorizationInfo) IsPermittedAll(permissions ...string) bool {
	for _, permission := range permissions {
		p, _ := NewPermission(permission)
		if !a.IsPermittedp(p) {
			return false
		}
	}
	return true
}

// IsPermittedp method returns true if the corresponding subject/user is permitted
// to perform an action or access a resource summarized by the specified
// permission string.
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

// IsPermittedAllp method returns true if the corresponding Subject/user implies
// all of the specified permission strings, false otherwise.
func (a *AuthorizationInfo) IsPermittedAllp(permissions ...*Permission) bool {
	for _, permission := range permissions {
		if !a.IsPermittedp(permission) {
			return false
		}
	}
	return true
}

// String method is stringer interface implementation.
func (a AuthorizationInfo) String() string {
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
