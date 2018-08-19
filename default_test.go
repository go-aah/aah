// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"crypto/tls"
	"html/template"
	"io"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"aahframework.org/ainsp"
	"aahframework.org/log"
	"aahframework.org/view"
	"github.com/stretchr/testify/assert"
)

func TestDefaultApp(t *testing.T) {
	// Default App init
	t.Log("Default App init")
	importPath := filepath.Join(testdataBaseDir(), "webapp1")
	SetAppBuildInfo(&BuildInfo{
		BinaryName: filepath.Base(importPath),
		Date:       time.Now().Format(time.RFC3339),
		Version:    "1.0.0",
	})
	err := Init(importPath)
	assert.Nil(t, err)

	AppLog().(*log.Logger).SetWriter(ioutil.Discard)

	// Default App information
	assert.Equal(t, "webapp1", AppName())
	assert.Equal(t, "", AppInstanceName())
	assert.Equal(t, "aah framework web application", AppDesc())
	assert.Equal(t, "dev", AppProfile())
	assert.Equal(t, importPath, AppBaseDir())
	assert.Equal(t, "/app", AppVirtualBaseDir())
	assert.Equal(t, importPath, AppImportPath()) // this is only for test scenario
	assert.Equal(t, "", AppHTTPAddress())
	assert.Equal(t, "8080", AppHTTPPort())
	assert.False(t, AppIsSSLEnabled())
	assert.Equal(t, "webapp1", AppBuildInfo().BinaryName)
	assert.Equal(t, "1.0.0", AppBuildInfo().Version)
	assert.Equal(t, "", AppSSLCert())
	assert.Equal(t, "", AppSSLKey())
	assert.Equal(t, "en", AppDefaultI18nLang())
	assert.True(t, strings.Contains(strings.Join(AppI18nLocales(), ", "), "en-us"))
	assert.True(t, strings.Contains(strings.Join(AllAppProfiles(), ", "), "prod"))

	// Default App module instances
	assert.NotNil(t, AppHTTPEngine())
	assert.NotNil(t, AppWSEngine())
	assert.NotNil(t, AppI18n())
	assert.NotNil(t, AppLog())
	assert.NotNil(t, AppConfig())
	assert.NotNil(t, AppRouter())
	assert.NotNil(t, AppEventStore())
	assert.NotNil(t, AppViewEngine())
	assert.NotNil(t, AppSecurityManager())
	assert.NotNil(t, AppSessionManager())
	assert.NotNil(t, AppVFS())

	// Default App Start and Shutdown
	t.Log("Default App Start and Shutdown")
	go Start()
	time.Sleep(10 * time.Millisecond)
	defer Shutdown()

	// Set default app to packaged
	SetAppPackaged(true)

	// Child app logger
	t.Log("Child app logger")
	ll := NewChildLogger(log.Fields{"key1": "value1"})
	assert.NotNil(t, ll)

	// TLS config
	t.Log("TLS Config")
	SetTLSConfig(&tls.Config{})

	// Add controller
	AddController(reflect.ValueOf(testSiteController{}), make([]*ainsp.Method, 0))

	SetErrorHandler(func(ctx *Context, e *Error) bool {
		t.Log("Error hanlder")
		return true
	})

	AddLoggerHook("testhook", func(e log.Entry) {
		t.Log("test logger hook")
	})

	// View Part
	AddTemplateFunc(template.FuncMap{
		"t1": func() string { return "t1 func" },
	})

	AddViewEngine("go", new(view.GoViewEngine))

	SetMinifier(func(contentType string, w io.Writer, r io.Reader) error {
		t.Log("this is second set", contentType, w, r)
		return nil
	})

	// Events part
	OnInit(func(e *Event) {
		t.Log("Application OnInit extension point")
	})

	OnStart(func(e *Event) {
		t.Log("Application OnStart extension point")
	})

	OnPreShutdown(func(e *Event) {
		t.Log("Application OnPreShutdown extension point")
	})

	OnPostShutdown(func(e *Event) {
		t.Log("Application OnPostShutdown extension point")
	})

	eventFunc1 := func(e *Event) {
		t.Log("custom-event-1")
	}
	SubscribeEvent("custom-event-1", EventCallback{Callback: eventFunc1})
	SubscribeEventFunc("custom-event-2", eventFunc1)
	PublishEvent("custom-event-1", "event data 1")
	PublishEventSync("custom-event-1", "event data 2")
	UnsubscribeEventFunc("custom-event-2", eventFunc1)
	UnsubscribeEvent("custom-event-1", EventCallback{Callback: eventFunc1})

	type testWebSocket struct{}
	// WebSocket
	AddWebSocket((*testWebSocket)(nil), []*ainsp.Method{
		{Name: "Text", Parameters: []*ainsp.Parameter{{Name: "encoding", Type: reflect.TypeOf((*string)(nil))}}},
		{Name: "Binary", Parameters: []*ainsp.Parameter{{Name: "encoding", Type: reflect.TypeOf((*string)(nil))}}},
	})

	// assert.Nil(t, SetAppProfile("dev"))
}

func TestHotAppReload(t *testing.T) {
	importPath := filepath.Join(testdataBaseDir(), "webapp1")
	ts := newTestServer(t, importPath)
	defer ts.Close()

	t.Logf("Test Server URL [Hot Reload]: %s", ts.URL)

	ts.app.hotReloadConfig()
}
