// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package security

import (
	"fmt"

	"aahframework.org/security.v0/authc"
	"aahframework.org/security.v0/authz"
	"aahframework.org/security.v0/session"
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Subject and its methods
//___________________________________

// Subject instance represents state and security operations for a single
// application user. These operations include authentication info (principal),
// authorization (access control), and session access. It is aah framework's
// primary mechanism for single-user security functionality.
//
// Acquiring a Subject
//
// To acquire the currently-executing Subject, use `ctx.Subject()`. Almost all
// security operations should be performed with the Subject returned from
// this method.
//
// Permission methods
//
// Subject instance provides a convenience wrapper method for all authentication
// (primary principal, is-authenticated, logout) and authorization (hasrole,
// hasanyrole, hasallroles, ispermitted, ispermittedall) purpose.
type Subject struct {
	AuthenticationInfo *authc.AuthenticationInfo
	AuthorizationInfo  *authz.AuthorizationInfo
	Session            *session.Session
}

// PrimaryPrincipal method is convenience wrapper. See `AuthenticationInfo.PrimaryPrincipal`.
func (s *Subject) PrimaryPrincipal() *authc.Principal {
	return s.AuthenticationInfo.PrimaryPrincipal()
}

// Principal method returns the principal value for given Claim.
// See `AuthenticationInfo.Principal`.
func (s *Subject) Principal(claim string) *authc.Principal {
	return s.AuthenticationInfo.Principal(claim)
}

// AllPrincipals method is convenience wrapper.
func (s *Subject) AllPrincipals() []*authc.Principal {
	return s.AuthenticationInfo.Principals
}

// IsAuthenticated method is convenience wrapper. See `Session.IsAuthenticated`.
func (s *Subject) IsAuthenticated() bool {
	if s.Session == nil {
		return false
	}
	return s.Session.IsAuthenticated
}

// Logout method is convenience wrapper. See `Session.Clear`.
func (s *Subject) Logout() {
	if s.Session != nil {
		s.Session.Clear()
	}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Subject's Authorization methods
//___________________________________

// HasRole method is convenience wrapper. See `AuthorizationInfo.HasRole`.
func (s *Subject) HasRole(role string) bool {
	return s.AuthorizationInfo.HasRole(role)
}

// HasAllRoles method is convenience wrapper. See `AuthorizationInfo.HasAllRoles`.
func (s *Subject) HasAllRoles(roles ...string) bool {
	return s.AuthorizationInfo.HasAllRoles(roles...)
}

// HasAnyRole method is convenience wrapper. See `AuthorizationInfo.HasAnyRole`.
func (s *Subject) HasAnyRole(roles ...string) bool {
	return s.AuthorizationInfo.HasAnyRole(roles...)
}

// IsPermitted method is convenience wrapper. See `AuthorizationInfo.IsPermitted`.
func (s *Subject) IsPermitted(permission string) bool {
	return s.AuthorizationInfo.IsPermitted(permission)
}

// IsPermittedAll method is convenience wrapper. See `AuthorizationInfo.IsPermittedAll`.
func (s *Subject) IsPermittedAll(permissions ...string) bool {
	return s.AuthorizationInfo.IsPermittedAll(permissions...)
}

// Reset method clear the instance for reuse.
func (s *Subject) Reset() {
	s.AuthenticationInfo = nil
	s.AuthorizationInfo = nil
	s.Session = nil
}

// String method is stringer interface implementation.
func (s Subject) String() string {
	return fmt.Sprintf("%s, %s, %s", s.AuthenticationInfo, s.AuthorizationInfo, s.Session)
}
