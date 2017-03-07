// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"reflect"
	"testing"

	"aahframework.org/test.v0-unstable/assert"
)

func TestActualType(t *testing.T) {
	ct := actualType((*engine)(nil))
	assert.Equal(t, "aah.engine", ct.String())

	ct = actualType(engine{})
	assert.Equal(t, "aah.engine", ct.String())

	ct = actualType(&engine{})
	assert.Equal(t, "aah.engine", ct.String())
}

func TestAddController(t *testing.T) {
	type (
		Level1 struct{ *Controller }

		Level2 struct{ Level1 }

		Level3 struct{ Level2 }

		Level4 struct{ Level3 }

		Path1 struct{ *Controller }

		Path2 struct {
			Level1
			Path1
			Level4
		}
	)

	cRegistry = controllerRegistry{}

	AddController((*Level1)(nil), nil)
	AddController((*Level2)(nil), nil)
	AddController((*Level3)(nil), nil)
	AddController((*Level4)(nil), nil)
	AddController((*Path1)(nil), nil)
	AddController((*Path2)(nil), nil)

	assertIndexes(t, Level1{}, [][]int{{0}})
	assertIndexes(t, Level2{}, [][]int{{0, 0}})
	assertIndexes(t, Level3{}, [][]int{{0, 0, 0}})
	assertIndexes(t, Level4{}, [][]int{{0, 0, 0, 0}})
	assertIndexes(t, Path1{}, [][]int{{0}})
	assertIndexes(t, Path2{}, [][]int{{0, 0}, {1, 0}, {2, 0, 0, 0, 0}})
}

func assertIndexes(t *testing.T, c interface{}, expected [][]int) {
	actual := findEmbeddedController(reflect.TypeOf(c))
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Indexes do not match. expected %v actual %v", expected, actual)
	}
}
