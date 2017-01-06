// Copyright (c) 2016 Jeevanandam M (https://github.com/jeevatkm)
// go-aah/essentails source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ess

import (
	"fmt"
	"testing"

	"aahframework.org/test/assert"
)

func TestGetFunctionName(t *testing.T) {
	type SampleStr struct {
		Name string
	}

	name := GetFunctionName(testFunc1)
	assert.Equal(t, "aahframework.org/essentials.testFunc1", name)

	name = GetFunctionName(SampleStr{})
	assert.Equal(t, "", name)

	name = GetFunctionName(nil)
	assert.Equal(t, "", name)
}

func TestGetCallerInfo(t *testing.T) {
	caller := GetCallerInfo()
	assert.Equal(t, "TestGetCallerInfo", caller.FunctionName)
	assert.Equal(t, "aahframework.org/essentials.TestGetCallerInfo", caller.QualifiedName)
	assert.Equal(t, "reflect_test.go", caller.FileName)
}

func testFunc1() {
	fmt.Println("testFunc1")
}
