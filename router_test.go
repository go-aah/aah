// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package router

import (
	"testing"

	"github.com/go-aah/test/assert"
)

func TestPathParamGet(t *testing.T) {
	pathParameters := pathParams{
		pathParam{"dir", "js"},
		pathParam{"filepath", "/inc/framework.js"},
	}

	fp := pathParameters.Get("filepath")
	assert.Equal(t, "/inc/framework.js", fp)

	notfound := pathParameters.Get("notfound")
	assert.Equal(t, "", notfound)
}
