// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/router source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package router

import (
	"strings"

	"aahframework.org/essentials.v0"
)

const (
	ruleStartByte = '['
	ruleEndByte   = ']'
)

func suffixCommaValue(s, v string) string {
	if ess.IsStrEmpty(s) {
		return v
	}
	return s + ", " + v
}

func findActionByHTTPMethod(method string) string {
	if action, found := HTTPMethodActionMap[method]; found {
		return action
	}
	return ""
}

// Return Values are
// 1. path param
// 2. validation rules
// 3. is validation rules exists
// 4. valild validation rules
func checkValidationRule(pathSeg string) (string, string, bool, bool) {
	sidx := strings.IndexByte(pathSeg, ruleStartByte)
	eidx := strings.IndexByte(pathSeg, ruleEndByte)

	// Validation rule exists but invalid
	if (sidx == -1 && eidx > 0) || (sidx >= 0 && eidx == -1) {
		return "", "", true, false
	}

	// Validation rule not exists
	if sidx == -1 && eidx == -1 {
		return pathSeg, "", false, true
	}

	pathParam := strings.TrimSpace(pathSeg[:sidx])
	paramRule := strings.TrimSpace(pathSeg[sidx+1 : eidx])
	return pathParam, paramRule, true, len(paramRule) > 0
}
