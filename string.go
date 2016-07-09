// Copyright (c) 2016 Jeevanandam M (https://github.com/jeevatkm)
// essentails source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ess

import "strings"

// StrIsEmpty returns true if strings is empty otherwise false
func StrIsEmpty(v string) bool {
	return len(strings.TrimSpace(v)) == 0
}
