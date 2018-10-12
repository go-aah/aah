// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"crypto/tls"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"aahframe.work/ahttp"
	"aahframe.work/ainsp"
	"aahframe.work/config"
	"aahframe.work/essentials"
	"aahframe.work/log"
	"aahframe.work/view"
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
	assert.NotNil(t, AppCacheManager())

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
		t.Log("Error handler")
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

	assert.Nil(t, SetAppProfile("dev"))
}

func TestHotAppReload(t *testing.T) {
	importPath := filepath.Join(testdataBaseDir(), "webapp1")
	ts := newTestServer(t, importPath)
	defer ts.Close()

	t.Logf("Test Server URL [Hot Reload]: %s", ts.URL)

	ts.app.performHotReload()
}

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
	ctx.Reply().BadRequest().Error(newError(nil, http.StatusBadRequest))

	em := new(errorManager)
	em.Handle(ctx)
}
