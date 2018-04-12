// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"path/filepath"
	"testing"

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
