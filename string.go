// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/essentails source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ess

import "strings"

// IsStrEmpty returns true if strings is empty otherwise false
func IsStrEmpty(v string) bool {
	return len(strings.TrimSpace(v)) == 0
}
