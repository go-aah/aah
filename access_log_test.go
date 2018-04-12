// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"fmt"
	"path/filepath"
	"testing"

	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/test.v0/assert"
)

func TestAccessLogInitAbsPath(t *testing.T) {
	logPath := filepath.Join(testdataBaseDir(), "sample-test-access.log")
	defer ess.DeleteFiles(logPath)

	a := newApp()
	cfg, _ := config.ParseString(fmt.Sprintf(`server {
    access_log {
      file = "%s"
    }
  }`, logPath))
	a.cfg = cfg

	err := a.initAccessLog()
	assert.Nil(t, err)
}
