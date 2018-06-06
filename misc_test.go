// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"fmt"
	"net/http"
	"path/filepath"
	"testing"

	"aahframework.org/ahttp.v0"
	"aahframework.org/ainsp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
	"aahframework.org/test.v0/assert"
)

func TestLogInitRelativeFilePath(t *testing.T) {
	logPath := filepath.Join(testdataBaseDir(), "sample-test-app.log")
	defer ess.DeleteFiles(logPath)

	// Relative path file
	a := newApp()
	cfg, _ := config.ParseString(`log {
    receiver = "file"
    file = "sample-test-app.log"
  }`)
	a.cfg = cfg

	err := a.initLog()
	assert.Nil(t, err)

	a.AddLoggerHook("myapphook", func(e log.Entry) {
		t.Logf("%v", e)
	})
}

func TestLogInitNoFilePath(t *testing.T) {
	// No file input - auto location
	logPath := filepath.Join(testdataBaseDir(), "wepapp1.log")
	defer ess.DeleteFiles(logPath)

	// Relative path file
	a := newApp()
	cfg, _ := config.ParseString(`log {
    receiver = "file"
  }`)
	a.cfg = cfg

	err := a.initLog()
	assert.Nil(t, err)

	a.AddLoggerHook("myapphook", func(e log.Entry) {
		t.Logf("%v", e)
	})
}

func TestAccessLogInitAbsPath(t *testing.T) {
	logPath := filepath.Join(testdataBaseDir(), "sample-test-access.log")
	defer ess.DeleteFiles(logPath)

	a := newApp()
	cfg, _ := config.ParseString(fmt.Sprintf(`server {
    access_log {
      file = "%s"
    }
  }`, filepath.ToSlash(logPath)))
	a.cfg = cfg

	err := a.initAccessLog()
	assert.Nil(t, err)
}

type testErrorController1 struct {
}

func (tec *testErrorController1) HandleError(err *Error) bool {
	log.Info("I have handled it at controller level")
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

	l, err := log.New(config.NewEmpty())
	assert.Nil(t, err)
	ctx.logger = l

	ctx.Reply().ContentType("application/json")
	ctx.Reply().Error(newError(nil, http.StatusBadRequest))

	em := new(errorManager)
	em.Handle(ctx)
}
