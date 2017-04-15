// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"path/filepath"
	"testing"

	"aahframework.org/essentials.v0"
	"aahframework.org/test.v0/assert"
)

func TestAahInitAppVariables(t *testing.T) {
	cfgDir := filepath.Join(getTestdataPath(), appConfigDir())
	err := initConfig(cfgDir)
	assert.Nil(t, err)

	err = initAppVariables()
	assert.NotNil(t, err)
	assert.Equal(t, "profile doesn't exists: env.dev", err.Error())

	AppConfig().SetString("env.dev.test_value", "dev test value")
	err = initAppVariables()
	assert.Nil(t, err)

	assert.Equal(t, "aahframework", AppName())
	assert.Equal(t, "127.0.0.1", AppHTTPAddress())
	assert.Equal(t, "80", AppHTTPPort())
	assert.Equal(t, "2006-01-02", AppDateFormat())
	assert.Equal(t, "2006-01-02 15:04:05", AppDateTimeFormat())
	assert.Equal(t, "en", AppDefaultI18nLang())
	assert.True(t, ess.IsStrEmpty(AppImportPath()))
	assert.False(t, AppIsSSLEnabled())
	assert.Equal(t, "dev", AppProfile())
	assert.Equal(t, "1m30s", appHTTPReadTimeout.String())
	assert.Equal(t, "1m30s", appHTTPWriteTimeout.String())
	assert.Equal(t, 1048576, appHTTPMaxHdrBytes)
	assert.False(t, appInitialized)
	assert.Equal(t, int64(33554432), appMultipartMaxMemory)
	assert.True(t, ess.IsStrEmpty(appSSLCert))
	assert.True(t, ess.IsStrEmpty(appSSLKey))

	// config error scenario

	// unsupported read timeout
	AppConfig().SetString("server.timeout.read", "20h")
	err = initAppVariables()
	assert.NotNil(t, err)
	assert.Equal(t, "'server.timeout.{read|write}' value is not a valid time unit", err.Error())
	AppConfig().SetString("server.timeout.read", "90s")

	// read timeout parsing error
	AppConfig().SetString("server.timeout.read", "ss")
	err = initAppVariables()
	assert.Equal(t, "'server.timeout.read': time: invalid duration ss", err.Error())
	AppConfig().SetString("server.timeout.read", "90s")

	// write timout pasring error
	AppConfig().SetString("server.timeout.write", "mm")
	err = initAppVariables()
	assert.Equal(t, "'server.timeout.write': time: invalid duration mm", err.Error())
	AppConfig().SetString("server.timeout.write", "90s")

	// max header bytes parsing error
	AppConfig().SetString("server.max_header_bytes", "2sb")
	err = initAppVariables()
	assert.Equal(t, "'server.max_header_bytes' value is not a valid size unit", err.Error())
	AppConfig().SetString("server.max_header_bytes", "1mb")

	// ssl cert required if enabled
	AppConfig().SetBool("server.ssl.enable", true)
	err = initAppVariables()
	assert.Equal(t, "HTTP SSL is enabled, so 'server.ssl.cert' & 'server.ssl.key' value is required", err.Error())
	AppConfig().SetBool("server.ssl.enable", false)

	// multipart size parsing error
	AppConfig().SetString("request.multipart_size", "2sb")
	err = initAppVariables()
	assert.Equal(t, "'request.multipart_size' value is not a valid size unit", err.Error())
	AppConfig().SetString("request.multipart_size", "12mb")
}
