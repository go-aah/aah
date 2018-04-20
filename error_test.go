// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"net/http"
	"testing"

	"aahframework.org/ahttp.v0"
	"aahframework.org/ainsp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/log.v0"
	"aahframework.org/test.v0/assert"
)

type testErrorController1 struct {
}

func (tec *testErrorController1) HandleError(err *Error) bool {
	log.Info("I have handler it")
	return true
}

func TestErrorCallControllerHandler(t *testing.T) {
	req, err := http.NewRequest(ahttp.MethodGet, "http://localhost:8080", nil)
	assert.Nil(t, err)
	ctx := &Context{
		Req:        ahttp.AcquireRequest(req),
		controller: &ainsp.Target{FqName: "testErrorController1"},
		target:     &testErrorController1{},
	}

	cfg, err := config.ParseString("")
	assert.Nil(t, err)

	l, err := log.New(cfg)
	assert.Nil(t, err)
	ctx.logger = l

	ctx.Reply().ContentType("application/json")
	ctx.Reply().Error(&Error{
		Code:    http.StatusBadRequest,
		Message: http.StatusText(http.StatusBadRequest),
	})

	em := new(errorManager)
	em.Handle(ctx)
}
