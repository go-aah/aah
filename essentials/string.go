// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org/essentials source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ess

import "strings"

// StringEmpty is empty string constant. Using `ess.StringEmpty` instead of "".
const StringEmpty = ""

// IsStrEmpty returns true if strings is empty otherwise false
func IsStrEmpty(v string) bool {
	return len(strings.TrimSpace(v)) == 0
}

// IsSliceContainsString method checks given string in the slice if found returns
// true otherwise false.
func IsSliceContainsString(strSlice []string, search string) bool {
	for _, str := range strSlice {
		if strings.EqualFold(str, search) {
			return true
		}
	}
	return false
}
