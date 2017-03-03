// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/essentials source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ess

import (
	"fmt"
	"testing"

	"aahframework.org/test.v0/assert"
)

func TestGetFunctionInfo(t *testing.T) {
	type SampleStr struct {
		Name string
	}

	info := GetFunctionInfo(testFunc1)
	assert.Equal(t, "aahframework.org/essentials.v0.testFunc1", info.QualifiedName)

	info = GetFunctionInfo(SampleStr{})
	assert.Equal(t, "", info.Name)

	info = GetFunctionInfo(nil)
	assert.NotNil(t, info)
	assert.Equal(t, "", info.Name)
}

func TestGetCallerInfo(t *testing.T) {
	caller := GetCallerInfo()
	assert.Equal(t, "TestGetCallerInfo", caller.FunctionName)
	assert.Equal(t, "aahframework.org/essentials.v0.TestGetCallerInfo", caller.QualifiedName)
	assert.Equal(t, "reflect_test.go", caller.FileName)
}

func testFunc1() {
	fmt.Println("testFunc1")
}
