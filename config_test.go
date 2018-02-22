// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"aahframework.org/test.v0/assert"
)

func TestConfigInit(t *testing.T) {
	cfgDir := filepath.Join(getTestdataPath(), appConfigDir())
	err := initConfig(cfgDir)

	assert.Nil(t, err)
	assert.NotNil(t, AppConfig())
	assert.Equal(t, "127.0.0.1", AppConfig().StringDefault("server.address", ""))
	assert.Equal(t, "X-Request-Id", AppConfig().StringDefault("request.id.header", ""))

	appConfig = nil
	err = initConfig(getTestdataPath())
	assert.Nil(t, AppConfig())
	assert.NotNil(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "aah application configuration does not exists"))
}

func TestConfigTemplateFuncs(t *testing.T) {
	cfgDir := filepath.Join(getTestdataPath(), appConfigDir())
	err := initConfig(cfgDir)
	assert.Nil(t, err)
	assert.NotNil(t, AppConfig())

	v1 := tmplConfig("request.multipart_size")
	assert.Equal(t, "32mb", v1.(string))

	v2 := tmplConfig("server.timeout.grace_shutdown")
	assert.Equal(t, "60s", v2.(string))

	v3 := tmplConfig("key.not.exists")
	assert.Equal(t, "", v3.(string))
}

func TestConfigHotReload(t *testing.T) {
	SetAppBuildInfo(&BuildInfo{
		BinaryName: "testapp",
		Date:       time.Now().Format(time.RFC3339),
		Version:    "1.0.0",
	})

	assert.False(t, isHotReload)
	appBaseDir = getTestdataPath()
	hotReloadConfig()
	appBaseDir = ""
}
