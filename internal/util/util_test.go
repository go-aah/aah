// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMimeTypeByExtension(t *testing.T) {
	testcases := []struct {
		input  string
		output string
	}{
		{},
		{input: ".html", output: "text/html; charset=utf-8"},
		{input: ".htm", output: "text/html; charset=utf-8"},
		{input: ".xml", output: "application/xml; charset=utf-8"},
		{input: ".json", output: "application/json; charset=utf-8"},
		{input: ".txt", output: "text/plain; charset=utf-8"},
		{input: ".text", output: "text/plain; charset=utf-8"},
		{input: ".css", output: "text/css; charset=utf-8"},
		{input: ".js", output: "application/javascript; charset=utf-8"},
	}

	for _, tc := range testcases {
		assert.Equal(t, tc.output, MimeTypeByExtension(tc.input))
	}
}
