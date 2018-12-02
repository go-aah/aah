// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ess

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetFunctionInfo(t *testing.T) {
	type SampleStr struct {
		Name string
	}

	info := GetFunctionInfo(testFunc1)
	assert.True(t, strings.Contains(info.QualifiedName, "essentials.testFunc1"))
	assert.True(t, strings.Contains(info.QualifiedName, "testFunc1"))

	info = GetFunctionInfo(SampleStr{})
	assert.Equal(t, "", info.Name)

	info = GetFunctionInfo(nil)
	assert.NotNil(t, info)
	assert.Equal(t, "", info.Name)
}

func TestGetCallerInfo(t *testing.T) {
	caller := GetCallerInfo()
	assert.Equal(t, "TestGetCallerInfo", caller.FunctionName)
	assert.True(t, strings.Contains(caller.QualifiedName, "essentials.TestGetCallerInfo"))
	assert.True(t, strings.Contains(caller.QualifiedName, "TestGetCallerInfo"))
	assert.Equal(t, "reflect_test.go", caller.FileName)
}

func testFunc1() {
	fmt.Println("testFunc1")
}
