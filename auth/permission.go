// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package auth

import (
	"errors"
	"strings"

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
	ErrPermissionStringEmpty = errors.New("security: permission string is empty")

	// ErrPermissionImproperFormat returned when permission string is composed or
	// formatted properly.
	//    For e.g.:
	//    "printer:print,query:epsoncolor"     # properly formatted
	//    "printer::epsoncolor"                # improperly formatted
	//    "printer::"                          # improperly formatted
	ErrPermissionImproperFormat = errors.New("security: permission string cannot contain parts with only dividers")
)

type (
	// Permission ...
	Permission struct {
		parts []parts
	}

	parts []string
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Global methods
//___________________________________

// NewPermission ...
func NewPermission(permission string) (*Permission, error) {
	return NewPermissioncs(permission, false)
}

// NewPermissioncs ...
func NewPermissioncs(permission string, caseSensitive bool) (*Permission, error) {
	permission = strings.TrimSpace(permission)
	if ess.IsStrEmpty(permission) {
		return nil, ErrPermissionStringEmpty
	}

	if !caseSensitive {
		permission = strings.ToLower(permission)
	}

	p := &Permission{parts: make([]parts, 0)}
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

// Implies ...
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

// String ...
func (p Permission) String() string {
	var strs []string
	for _, part := range p.parts {
		strs = append(strs, strings.Join(part, subPartDividerToken))
	}
	return strings.Join(strs, partDividerToken)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// parts methods
//___________________________________

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
