// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"path/filepath"
	"testing"
	"time"

	"aahframework.org/essentials.v0"
	"aahframework.org/test.v0/assert"
)

func TestServerStart1(t *testing.T) {
	defer ess.DeleteFiles("testapp.pid")
	// App Config
	cfgDir := filepath.Join(getTestdataPath(), appConfigDir())
	err := initConfig(cfgDir)
	assert.Nil(t, err)
	assert.NotNil(t, AppConfig())

	AppConfig().SetString("env.dev.name", "testapp")

	err = initAppVariables()
	assert.Nil(t, err)
	appInitialized = true
	Start()
}

func TestServerStart2(t *testing.T) {
	defer ess.DeleteFiles("testapp.pid")
	testEng.Lock()
	defer testEng.Unlock()

	// App Config
	cfgDir := filepath.Join(getTestdataPath(), appConfigDir())
	err := initConfig(cfgDir)
	assert.Nil(t, err)
	assert.NotNil(t, AppConfig())

	AppConfig().SetString("env.dev.name", "testapp")

	err = initAppVariables()
	assert.Nil(t, err)
	appInitialized = true

	// Router
	err = initRoutes(cfgDir, AppConfig())
	assert.Nil(t, err)
	assert.NotNil(t, AppRouter())

	// Security
	err = initSecurity(cfgDir, AppConfig())
	assert.Nil(t, err)

	// i18n
	i18nDir := filepath.Join(getTestdataPath(), appI18nDir())
	err = initI18n(i18nDir)
	assert.Nil(t, err)
	assert.NotNil(t, AppI18n())

	buildTime := time.Now().Format(time.RFC3339)
	SetAppBuildInfo(&BuildInfo{
		BinaryName: "testapp",
		Date:       buildTime,
		Version:    "1.0.0",
	})
	AppConfig().SetString("server.port", "8080")
	go Start()
}
