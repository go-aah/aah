// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package authc

import "fmt"

type (
	// AuthenticationInfo represents a Subject's (aka user's) stored account
	// information relevant to the authentication/log-in process only.
	//
	// It is important to understand the difference between this interface and
	// the AuthenticationToken struct. AuthenticationInfo implementations represent
	// already-verified and stored account data, whereas an AuthenticationToken
	// represents data submitted for any given login attempt (which may or may not
	// successfully match the verified and stored account AuthenticationInfo).
	//
	// Because the act of authentication (log-in) is orthogonal to authorization
	// (access control), this struct is intended to represent only the account data
	// needed by aah framework during an authentication attempt. aah framework also
	// has a parallel AuthorizationInfo struct for use during the authorization
	// process that references access control data such as roles and permissions.
	AuthenticationInfo struct {
		Credential []byte
		IsLocked   bool
		IsExpired  bool

		principals []*Principal
	}

	// Principal struct holds the principal associated with a corresponding Subject.
	// A principal is just a security term for an identifying attribute, such as a
	// username or user id or social security number or anything else that can be
	// considered an 'identifying' attribute for a Subject.
	Principal struct {
		Realm     string
		Value     string
		IsPrimary bool
	}
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Global methods
//___________________________________

// NewAuthenticationInfo method creates an `AuthenticationInfo` instance with zero
// values. Then using this instance you fill-in user credential, principals, locked,
// expried information.
func NewAuthenticationInfo() *AuthenticationInfo {
	return &AuthenticationInfo{
		principals: make([]*Principal, 0),
	}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// AuthenticationInfo methods
//___________________________________

// AddPrincipal method adds the given `Principal` instance into `AuthenticationInfo`.
func (a *AuthenticationInfo) AddPrincipal(principals ...*Principal) *AuthenticationInfo {
	a.principals = append(a.principals, principals...)
	return a
}

// PrimaryPrincipal method returns the primary Principal instance if principal
// object has `IsPrimary` as true otherwise nil.
//
// Typically one principal is required for the subject aka user.
func (a *AuthenticationInfo) PrimaryPrincipal() *Principal {
	for _, p := range a.principals {
		if p.IsPrimary {
			return p
		}
	}
	return nil
}

// AllPrincipals method returns all the principals available from `AuthenticationInfo`.
func (a *AuthenticationInfo) AllPrincipals() []*Principal {
	return a.principals
}

// Merge method merges the given authentication information into existing
// `AuthenticationInfo` instance. IsExpired and IsLocked values considered as latest
// from the given object.
func (a *AuthenticationInfo) Merge(oa *AuthenticationInfo) *AuthenticationInfo {
	a.AddPrincipal(oa.principals...)
	a.IsExpired = oa.IsExpired
	a.IsLocked = oa.IsLocked
	return a
}

// String method is stringer interface implementation.
func (a *AuthenticationInfo) String() string {
	return fmt.Sprintf("AuthenticationInfo:: Principals%s, Credential: *******, IsLocked: %v, IsExpired: %v",
		a.principals, a.IsLocked, a.IsExpired)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Principal methods
//___________________________________

// String method is stringer interface implementation.
func (p *Principal) String() string {
	return fmt.Sprintf("Realm: %v, Principal: %s, IsPrimary: %v", p.Realm, p.Value, p.IsPrimary)
}
