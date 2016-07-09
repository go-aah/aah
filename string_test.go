// Copyright (c) 2016 Jeevanandam M (https://github.com/jeevatkm)
// essentails source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ess

import "testing"

func TestStringEmptyNotEmpty(t *testing.T) {
	assertEqual(t, "TestStringEmptyNotEmpty", false, StrIsEmpty("    Welcome    "))

	assertEqual(t, "TestStringEmptyNotEmpty", true, StrIsEmpty("        "))
}
