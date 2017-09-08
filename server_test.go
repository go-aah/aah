// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"aahframework.org/config.v0"
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
	err = initSecurity(AppConfig())
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
	AppConfig().SetString("server.port", "80")
	Start()
}

func TestServerHTTPRedirect(t *testing.T) {
	cfg, _ := config.ParseString("")

	// redirect not enabled
	startHTTPRedirect(cfg)

	// redirect enabled but port not provided
	cfg.SetBool("server.ssl.redirect_http.enable", true)
	cfg.SetString("server.port", "8443")
	startHTTPRedirect(cfg)

	// redirect enabled with port
	cfg.SetString("server.ssl.redirect_http.port", "8080")
	go startHTTPRedirect(cfg)

	// http.NewRequest("GET", "http://localhost:8080/", nil)
	resp, err := http.Get("http://localhost:8080/contactus.html?utm_source=footer")
	assert.Nil(t, resp)
	assert.True(t, strings.Contains(err.Error(), "localhost:8443"))
}
