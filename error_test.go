// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"testing"

	"aahframework.org/test.v0/assert"
)

func TestErrorWriteInfo(t *testing.T) {
	ctx1 := &Context{reply: acquireReply()}
	ctx1.Reply().ContentType("application/json")
	writeErrorInfo(ctx1, 400, "Bad Request")

	assert.NotNil(t, ctx1.Reply().Rdr)
	jsonr := ctx1.Reply().Rdr.(*JSON)
	assert.NotNil(t, jsonr)
	assert.NotNil(t, jsonr.Data)
	assert.Equal(t, 400, jsonr.Data.(Data)["code"])
	assert.Equal(t, "Bad Request", jsonr.Data.(Data)["message"])

	ctx2 := &Context{reply: acquireReply()}
	ctx2.Reply().ContentType("application/xml")
	writeErrorInfo(ctx2, 500, "Internal Server Error")

	assert.NotNil(t, ctx2.Reply().Rdr)
	xmlr := ctx2.Reply().Rdr.(*XML)
	assert.NotNil(t, xmlr)
	assert.NotNil(t, xmlr.Data)
	assert.Equal(t, 500, xmlr.Data.(Data)["code"])
	assert.Equal(t, "Internal Server Error", xmlr.Data.(Data)["message"])
}
