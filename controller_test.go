// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"testing"

	"aahframework.org/router.v0"
	"aahframework.org/test.v0/assert"
)

func TestActualType(t *testing.T) {
	ct := actualType((*engine)(nil))
	assert.Equal(t, "aah.engine", ct.String())

	ct = actualType(engine{})
	assert.Equal(t, "aah.engine", ct.String())

	ct = actualType(&engine{})
	assert.Equal(t, "aah.engine", ct.String())
}

func TestCRegistryLookup(t *testing.T) {
	addToCRegistry()

	ci := cRegistry.Lookup(&router.Route{Controller: "Path1"})
	assert.NotNil(t, ci)
	assert.Equal(t, "Path1", ci.Name())

	ci = cRegistry.Lookup(&router.Route{Controller: "ControllerNotExists"})
	assert.Nil(t, ci)
}

func TestFindMethodController(t *testing.T) {
	addToCRegistry()

	ci := cRegistry.Lookup(&router.Route{Controller: "Level3"})
	assert.NotNil(t, ci)
	mi := ci.FindMethod("Testing")
	assert.NotNil(t, mi)
	assert.Equal(t, "Testing", mi.Name)

	ci = cRegistry.Lookup(&router.Route{Controller: "Path1"})
	assert.NotNil(t, ci)
	mi = ci.FindMethod("NoMethodExists")
	assert.Nil(t, mi)
}
