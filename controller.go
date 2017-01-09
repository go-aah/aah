// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import "aahframework.org/aah/ahttp"

// Controller type for aah framework, gets embedded in application controller.
type Controller struct {
	// Req is HTTP request instance
	Req *ahttp.Request

	res ahttp.ResponseWriter
}
