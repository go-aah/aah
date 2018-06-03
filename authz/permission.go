// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package authz

import (
	"errors"
	"strings"
	"sync"

	"aahframework.org/essentials.v0"
)

const (
	wildcardToken       = "*"
	partDividerToken    = ":"
	subPartDividerToken = ","
)

var (
	// ErrPermissionStringEmpty returned when empty permission string supplied to
	// methods `security.authz.NewPermission` or `security.authz.NewPermissioncs`.
	ErrPermissionStringEmpty = errors.New("security/authz: permission string is empty")

	// ErrPermissionImproperFormat returned when permission string is composed or
	// formatted properly.
	//    For e.g.:
	//    "printer:print,query:epsoncolor"     # properly formatted
	//    "printer::epsoncolor"                # improperly formatted
	//    "printer::"                          # improperly formatted
	ErrPermissionImproperFormat = errors.New("security: permission string cannot contain parts with only dividers")

	permissionPool = &sync.Pool{New: func() interface{} { return &Permission{parts: make([]parts, 0)} }}
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package methods
//___________________________________

// NewPermission method creats the permission instance for the given
// permission string in incase-sensitive. If any error returns nil and error info.
func NewPermission(permission string) (*Permission, error) {
	return NewPermissioncs(permission, false)
}

// NewPermissioncs method creats the permission instance for the given
// permission string in Case-Sensitive. If any error returns nil and error info.
func NewPermissioncs(permission string, caseSensitive bool) (*Permission, error) {
	permission = strings.TrimSpace(permission)
	if ess.IsStrEmpty(permission) {
		return nil, ErrPermissionStringEmpty
	}

	if !caseSensitive {
		permission = strings.ToLower(permission)
	}

	p := acquirePermission()
	for _, part := range strings.Split(permission, partDividerToken) {
		subParts := strings.Split(part, subPartDividerToken)
		if len(subParts) == 1 && ess.IsStrEmpty(subParts[0]) {
			return nil, ErrPermissionImproperFormat
		}

		var sparts parts
		for _, sp := range subParts {
			if !ess.IsStrEmpty(sp) {
				sparts = append(sparts, strings.TrimSpace(sp))
			}
		}

		p.parts = append(p.parts, sparts)
	}

	if len(p.parts) == 0 {
		return nil, ErrPermissionImproperFormat
	}

	return p, nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Permission methods
//___________________________________

// Permission represents the ability to perform an action or access a resource.
// A Permission is the most granular, or atomic, unit in a system's security
// policy and is the cornerstone upon which fine-grained security models are built.
//
// aah framework provides a very powerful security implementation that is
// inspired by `Shiro` security framework.
//
// It is important to understand a Permission instance only represents
// functionality or access - it does not grant it. Granting access to an
// application functionality or a particular resource is done by the
// application's security configuration, typically by assigning Permissions to
// users, roles and/or groups.
//
// Most typical systems are what the `aah framework` calls role-based in nature,
// where a role represents common behavior for certain user types.
// For e.g:, a system might have an Aministrator role, a User or Guest roles, etc.
//
// But if you have a dynamic security model, where roles can be created and
// deleted at runtime, you can't hard-code role names in your code.
// In this environment, roles themselves aren't very useful. What matters is
// what permissions are assigned to these roles.
//
// Under this paradigm, permissions are immutable and reflect an application's
// raw functionality (opening files, accessing a web URL, creating users, etc).
// This is what allows a system's security policy to be dynamic: because
// Permissions represent raw functionality and only change when the application's
// source code changes, they are immutable at runtime - they represent 'what'
// the system can do. Roles, users, and groups are the 'who' of the application.
// Determining 'who' can do 'what' then becomes a simple exercise of associating
// Permissions to roles, users, and groups in some way.
//
// Most applications do this by associating a named role with permissions
// (i.e. a role 'has a' collection of Permissions) and then associate users
// with roles (i.e. a user 'has a' collection of roles) so that by transitive
// association, the user 'has' the permissions in their roles. There are numerous
// variations on this theme (permissions assigned directly to users, or
// assigned to groups, and users added to groups and these groups in turn have
// roles, etc, etc). When employing a permission-based security model instead
// of a role-based one, users, roles, and groups can all be created, configured
// and/or deleted at runtime. This enables an extremely powerful security model.
//
// A benefit to `aah framework` is that, although it assumes most systems are
// based on these types of static role or dynamic role w/ permission schemes,
// it does not require a system to model their security data this way - all
// Permission checks are relegated to `Authorizer` interface to implementations,
// and only those implementations really determine how a user 'has' a permission
// or not. The `Authorizer` could use the semantics described here, or it could
// utilize some other mechanism entirely - it is always up to the application
// developer.
type Permission struct {
	parts []parts
}

// Implies method returns true if this current instance implies all the
// functionality and/or resource access described by the specified Permission
// argument otherwise false.
//
// That is, this current instance must be exactly equal to or a superset of the
// functionality and/or resource access described by the given Permission argument.
// Yet another way of saying this would be:
//
// If "permission1 implies permission2", i.e. permission1.implies(permission2),
// then any Subject granted permission1 would have ability greater than or equal
// to that defined by permission2.
func (p *Permission) Implies(permission *Permission) bool {
	i := 0
	for _, otherPart := range permission.parts {
		// If this permission has less parts than the other permission,
		// everything after the number of parts contained in this permission
		// is automatically implied, so return true.
		if len(p.parts)-1 < i {
			return true
		}

		part := p.parts[i]
		if !part.Contains(wildcardToken) && !part.ContainsAll(otherPart) {
			return false
		}
		i++
	}

	// If this permission has more parts than the other parts,
	// only imply it if all of the other parts are wildcards.
	for ; i < len(p.parts); i++ {
		if !p.parts[i].Contains(wildcardToken) {
			return false
		}
	}

	return true
}

// Reset method resets the instance values for repurpose.
func (p *Permission) Reset() {
	p.parts = make([]parts, 0)
}

// String method `Stringer` interface implementation.
func (p Permission) String() string {
	var strs []string
	for _, part := range p.parts {
		strs = append(strs, strings.Join(part, subPartDividerToken))
	}
	return "permission(" + strings.Join(strs, partDividerToken) + ")"
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// parts methods
//___________________________________

type parts []string

func (p parts) ContainsAll(pp parts) bool {
	for _, v := range pp {
		if !p.Contains(v) {
			return false
		}
	}
	return true
}

func (p parts) ContainsAny(pp parts) bool {
	for _, v := range pp {
		if p.Contains(v) {
			return true
		}
	}
	return false
}

func (p parts) Contains(part string) bool {
	for _, v := range p {
		if v == part {
			return true
		}
	}
	return false
}
