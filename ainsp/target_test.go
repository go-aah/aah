// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// aahframework.org/ainsp source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ainsp

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type (
	Context struct {
	}

	Anonymous1 struct {
		Name string
	}

	Func1 func(e *string)

	Level1 struct{ *Context }

	Level2 struct{ Level1 }

	Level3 struct{ Level2 }

	Level4 struct{ Level3 }

	Path1 struct {
		Anonymous Anonymous1
		*Context
	}

	Path2 struct {
		Level1
		Path1
		Level4
		Func1
	}
)

var ctxPtrType = reflect.TypeOf((*Context)(nil))

func TestTargetRegisterForTargets(t *testing.T) {
	tr := &TargetRegistry{
		Registry:   make(map[string]*Target),
		SearchType: ctxPtrType,
	}

	tr.Add((*Level1)(nil), []*Method{
		{
			Name:       "Index",
			Parameters: []*Parameter{},
		},
	})
	tr.Add((*Level2)(nil), []*Method{
		{
			Name:       "Scope",
			Parameters: []*Parameter{},
		},
	})
	tr.Add((*Level3)(nil), []*Method{
		{
			Name: "Testing",
			Parameters: []*Parameter{
				{
					Name: "userId",
					Type: reflect.TypeOf((*int)(nil)),
				},
			},
		},
	})
	tr.Add((*Level4)(nil), nil)
	tr.Add((*Path1)(nil), nil)
	tr.Add((*Path2)(nil), nil)

	// target exists
	t.Log("target exists")
	target := tr.Lookup("aahframe.work/aah/ainsp.Level3")
	assert.NotNil(t, target)
	assert.Equal(t, "aahframe.work/aah/ainsp/Level3", target.FqName)
	assert.Equal(t, "aahframe.work/aah/ainsp", target.Namespace)
	assert.NotNil(t, target.Methods)
	assert.Equal(t, "Level3", target.Name)

	// method exists
	t.Log("method exists")
	method := target.Lookup("Testing")
	assert.NotNil(t, method)
	assert.Equal(t, "Testing", method.Name)
	assert.NotNil(t, method.Parameters)

	// target not exists
	t.Log("target not exists")
	target = tr.Lookup("Level10NotExists")
	assert.Nil(t, target)
	target = tr.Lookup("aahframe.work/aah/ainsp/Level3")

	// method not exists
	t.Log("method not exists")
	method = target.Lookup("MethodNotExists")
	assert.Nil(t, method)

}

func TestTypeEmbeddedIndexes(t *testing.T) {
	testTypeEmbeddedIndexes(t, Level1{}, [][]int{{0}})
	testTypeEmbeddedIndexes(t, Level2{}, [][]int{{0, 0}})
	testTypeEmbeddedIndexes(t, Level3{}, [][]int{{0, 0, 0}})
	testTypeEmbeddedIndexes(t, Level4{}, [][]int{{0, 0, 0, 0}})
	testTypeEmbeddedIndexes(t, Path1{}, [][]int{{1}})
	testTypeEmbeddedIndexes(t, Path2{}, [][]int{{0, 0}, {1, 1}, {2, 0, 0, 0, 0}})
}

func testTypeEmbeddedIndexes(t *testing.T, c interface{}, expected [][]int) {
	actual := FindFieldIndexes(reflect.TypeOf(c), ctxPtrType)
	assert.Equalf(t, expected, actual, "Indexes do not match")
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Indexes do not match. expected %v actual %v", expected, actual)
	}
}
