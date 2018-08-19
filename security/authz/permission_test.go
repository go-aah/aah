// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package authz

import (
	"testing"

	"aahframework.org/test.v0/assert"
)

func TestAuthPart(t *testing.T) {
	p1 := parts{"one", "two", "three"}
	assert.True(t, p1.Contains("two"))
	assert.True(t, p1.Contains("three"))
	assert.False(t, p1.Contains("four"))

	assert.True(t, p1.ContainsAll(parts{"one", "two", "three"}))
	assert.False(t, p1.ContainsAll(parts{"one", "two", "three", "four"}))
	assert.True(t, p1.ContainsAll(parts{"one", "three"}))
}

func TestAuthPermissionError(t *testing.T) {
	_, err := NewPermission("")
	assert.Equal(t, ErrPermissionStringEmpty, err)

	_, err = NewPermission("   ")
	assert.Equal(t, ErrPermissionStringEmpty, err)

	_, err = NewPermission("::,,::,:")
	assert.Equal(t, ErrPermissionImproperFormat, err)

	_, err = NewPermission("one :, two ,three:four:five,:*")
	assert.Nil(t, err)
}

func TestAuthPermissionSimple(t *testing.T) {
	var p1, p2 *Permission

	// Case insensitive, same
	p1, _ = NewPermission("something")
	p2, _ = NewPermission("something")
	assert.True(t, p1.Implies(p2))
	assert.True(t, p2.Implies(p1))
	releasePermission(p1, p2)

	// Case insensitive, different case
	p1, _ = NewPermission("something")
	p2, _ = NewPermission("SOMETHING")
	assert.True(t, p1.Implies(p2))
	assert.True(t, p2.Implies(p1))
	releasePermission(p1, p2)

	// Case insensitive, different word
	p1, _ = NewPermission("something")
	p2, _ = NewPermission("else")
	assert.False(t, p1.Implies(p2))
	assert.False(t, p2.Implies(p1))

	// Case sensitive same
	p1, _ = NewPermissioncs("YESYES", false)
	p2, _ = NewPermissioncs("YESYES", false)
	assert.True(t, p1.Implies(p2))
	assert.True(t, p2.Implies(p1))

	// Case sensitive, different case
	p1, _ = NewPermissioncs("YESYES", false)
	p2, _ = NewPermissioncs("yESYeS", false)
	assert.True(t, p1.Implies(p2))
	assert.True(t, p2.Implies(p1))

	// Case sensitive, different word
	p1, _ = NewPermissioncs("YESYES", false)
	p2, _ = NewPermissioncs("nono", false)
	assert.False(t, p1.Implies(p2))
	assert.False(t, p2.Implies(p1))
}

func TestAuthPermissionList(t *testing.T) {
	var p1, p2, p3 *Permission

	p1, _ = NewPermission("one,two")
	p2, _ = NewPermission("one")
	assert.True(t, p1.Implies(p2))
	assert.False(t, p2.Implies(p1))

	p1, _ = NewPermission("one,two,three")
	p2, _ = NewPermission("one,three")
	assert.True(t, p1.Implies(p2))
	assert.False(t, p2.Implies(p1))

	p1, _ = NewPermission("one,two:one,two,three")
	p2, _ = NewPermission("one:three")
	p3, _ = NewPermission("one:two,three")
	assert.True(t, p1.Implies(p2))
	assert.False(t, p2.Implies(p1))
	assert.True(t, p1.Implies(p3))
	assert.False(t, p2.Implies(p3))
	assert.True(t, p3.Implies(p2))

	p1, _ = NewPermission("one,two,three:one,two,three:one,two")
	p2, _ = NewPermission("one:three:two")
	assert.True(t, p1.Implies(p2))
	assert.False(t, p2.Implies(p1))
	assert.Equal(t, "permission(one,two,three:one,two,three:one,two)", p1.String())

	p1, _ = NewPermission("one")
	p2, _ = NewPermission("one:two,three,four")
	p3, _ = NewPermission("one:two,three,four:five:six:seven")
	assert.True(t, p1.Implies(p2))
	assert.True(t, p1.Implies(p3))
	assert.False(t, p2.Implies(p1))
	assert.False(t, p3.Implies(p1))
	assert.True(t, p2.Implies(p3))
	releasePermission(p1, p2, p3)

	// Check permissions with that contain the same list parts are equal.
	p1, _ = NewPermission("one,two:three,four")
	p2, _ = NewPermission("two,one:four,three")
	assert.True(t, p1.Implies(p2))
	assert.True(t, p2.Implies(p1))
}

func TestAuthPermissionWildcard(t *testing.T) {
	var p1, p2, p3, p4, p5, p6, p7, p8 *Permission

	p1, _ = NewPermission("*")
	p2, _ = NewPermission("one")
	p3, _ = NewPermission("one:two")
	p4, _ = NewPermission("one,two:three,four")
	p5, _ = NewPermission("one,two:three,four,five:six:seven,eight")
	assert.True(t, p1.Implies(p2))
	assert.True(t, p1.Implies(p3))
	assert.True(t, p1.Implies(p4))
	assert.True(t, p1.Implies(p5))

	p1, _ = NewPermission("newsletter:*")
	p2, _ = NewPermission("newsletter:read")
	p3, _ = NewPermission("newsletter:read,write")
	p4, _ = NewPermission("newsletter:*")
	p5, _ = NewPermission("newsletter:*:*")
	p6, _ = NewPermission("newsletter:*:read")
	p7, _ = NewPermission("newsletter:write:*")
	p8, _ = NewPermission("newsletter:read,write:*")
	assert.True(t, p1.Implies(p2))
	assert.True(t, p1.Implies(p3))
	assert.True(t, p1.Implies(p4))
	assert.True(t, p1.Implies(p5))
	assert.True(t, p1.Implies(p6))
	assert.True(t, p1.Implies(p7))
	assert.True(t, p1.Implies(p8))

	p1, _ = NewPermission("newsletter:*:*")
	assert.True(t, p1.Implies(p2))
	assert.True(t, p1.Implies(p3))
	assert.True(t, p1.Implies(p4))
	assert.True(t, p1.Implies(p5))
	assert.True(t, p1.Implies(p6))
	assert.True(t, p1.Implies(p7))
	assert.True(t, p1.Implies(p8))

	p1, _ = NewPermission("newsletter:*:*:*")
	assert.True(t, p1.Implies(p2))
	assert.True(t, p1.Implies(p3))
	assert.True(t, p1.Implies(p4))
	assert.True(t, p1.Implies(p5))
	assert.True(t, p1.Implies(p6))
	assert.True(t, p1.Implies(p7))
	assert.True(t, p1.Implies(p8))

	p1, _ = NewPermission("newsletter:*:read")
	p2, _ = NewPermission("newsletter:123:read")
	p3, _ = NewPermission("newsletter:123,456:read,write")
	p4, _ = NewPermission("newsletter:read")
	p5, _ = NewPermission("newsletter:read,write")
	p6, _ = NewPermission("newsletter:123:read:write")
	assert.True(t, p1.Implies(p2))
	assert.False(t, p1.Implies(p3))
	assert.False(t, p1.Implies(p4))
	assert.False(t, p1.Implies(p5))
	assert.True(t, p1.Implies(p6))

	p1, _ = NewPermission("newsletter:*:read:*")
	assert.True(t, p1.Implies(p2))
	assert.True(t, p1.Implies(p6))
}
