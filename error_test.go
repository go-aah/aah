// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"net/http"
	"testing"

	"aahframework.org/test.v0/assert"
)

func TestErrorHandler(t *testing.T) {
	// 400
	ctx1 := &Context{reply: acquireReply()}
	ctx1.Reply().ContentType("application/json")
	handleError(ctx1, &Error{
		Code:    http.StatusBadRequest,
		Message: http.StatusText(http.StatusBadRequest),
	})
	assert.NotNil(t, ctx1.Reply().Rdr)
	jsonr := ctx1.Reply().Rdr.(*JSON)
	assert.NotNil(t, jsonr)
	assert.NotNil(t, jsonr.Data)
	assert.Equal(t, 400, jsonr.Data.(*Error).Code)
	assert.Equal(t, "Bad Request", jsonr.Data.(*Error).Message)

	// 500
	ctx2 := &Context{reply: acquireReply()}
	ctx2.Reply().ContentType("application/xml")
	handleError(ctx2, &Error{
		Code:    http.StatusInternalServerError,
		Message: http.StatusText(http.StatusInternalServerError),
	})
	assert.NotNil(t, ctx2.Reply().Rdr)
	xmlr := ctx2.Reply().Rdr.(*XML)
	assert.NotNil(t, xmlr)
	assert.NotNil(t, xmlr.Data)
	assert.Equal(t, 500, xmlr.Data.(*Error).Code)
	assert.Equal(t, "Internal Server Error", xmlr.Data.(*Error).Message)

	SetErrorHandler(func(ctx *Context, err *Error) {
		t.Log(ctx, err)
	})
}
