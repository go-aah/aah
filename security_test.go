// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package security

import (
	"os"
	"path/filepath"
	"testing"

	"aahframework.org/config.v0"
	"aahframework.org/test.v0/assert"
)

func TestSecurityInit(t *testing.T) {
	appCfg, _ := config.ParseString("")
	configPath := filepath.Join(getTestdataPath(), "security.conf")

	s1, err := New(configPath, appCfg)
	assert.Nil(t, err)
	assert.NotNil(t, s1.SessionManager)

	s2, err := New("not-exists.conf", appCfg)
	assert.Equal(t, "security: configuration does not exists: not-exists.conf", err.Error())
	assert.Nil(t, s2)
}

func getTestdataPath() string {
	wd, _ := os.Getwd()
	return filepath.Join(wd, "testdata")
}
