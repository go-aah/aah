// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"html/template"
	"io"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	security "aahframework.org/security.v0"
	"aahframework.org/test.v0/assert"
	"aahframework.org/view.v0"
)

func TestViewInit(t *testing.T) {
	appCfg, _ := config.ParseString("")
	viewDir := filepath.Join(getTestdataPath(), appViewsDir())
	err := initViewEngine(viewDir, appCfg)
	assert.Nil(t, err)
	assert.NotNil(t, AppViewEngine())

	// cleanup
	appViewEngine = nil
}

func TestViewInitDirNotExists(t *testing.T) {
	appCfg, _ := config.ParseString("")
	viewDir := filepath.Join(getTestdataPath(), "views-not-exists")

	err := initViewEngine(viewDir, appCfg)
	assert.True(t, err == nil)
	assert.Nil(t, AppViewEngine())
}

func TestViewInitEngineNotFound(t *testing.T) {
	appCfg, _ := config.ParseString(`
  view {
    engine = "jade1"
  }
  `)
	viewDir := filepath.Join(getTestdataPath(), appViewsDir())
	err := initViewEngine(viewDir, appCfg)
	assert.Equal(t, "view: named engine not found: jade1", err.Error())
	assert.Nil(t, AppViewEngine())
}

func TestViewInitExternalEngine(t *testing.T) {
	appCfg, _ := config.ParseString("")
	viewDir := filepath.Join(getTestdataPath(), appViewsDir())

	assert.False(t, appIsExternalTmplEngine)

	appViewEngine = &view.GoViewEngine{}
	err := initViewEngine(viewDir, appCfg)
	assert.Nil(t, err)

	assert.True(t, appIsExternalTmplEngine)

	// cleanup
	appViewEngine = nil
	appIsExternalTmplEngine = false
}

func TestViewAddTemplateFunc(t *testing.T) {
	AddTemplateFunc(template.FuncMap{
		"join":     strings.Join,
		"safeHTML": strings.Join, // for duplicate test, don't mind
	})

	_, found := view.TemplateFuncMap["join"]
	assert.True(t, found)
}

func TestViewStore(t *testing.T) {
	err := AddViewEngine("go", &view.GoViewEngine{})
	assert.NotNil(t, err)
	assert.Equal(t, "view: engine name 'go' is already added, skip it", err.Error())

	err = AddViewEngine("custom", nil)
	assert.NotNil(t, err)
	assert.Equal(t, "view: engine value is nil", err.Error())

	engine, found := view.GetEngine("go")
	assert.NotNil(t, engine)
	assert.True(t, found)

	engine, found = view.GetEngine("myengine")
	assert.Nil(t, engine)
	assert.False(t, found)
}

func TestViewResolveView(t *testing.T) {
	defer ess.DeleteFiles("testapp.pid")
	appCfg, _ := config.ParseString("")
	e := newEngine(appCfg)

	viewDir := filepath.Join(getTestdataPath(), appViewsDir())
	err := initViewEngine(viewDir, appCfg)
	assert.Nil(t, err)
	assert.NotNil(t, AppViewEngine())

	req := httptest.NewRequest("GET", "http://localhost:8080/index.html", nil)
	ctx := e.prepareContext(httptest.NewRecorder(), req)

	type AppController struct{}
	ctx.controller = &controllerInfo{Type: reflect.TypeOf(AppController{})}
	ctx.action = &MethodInfo{
		Name:       "Index",
		Parameters: []*ParameterInfo{},
	}
	ctx.Reply().ContentType(ahttp.ContentTypeHTML.Raw())
	ctx.AddViewArg("MyName", "aah framework")

	e.resolveView(ctx)

	assert.NotNil(t, ctx.Reply().Rdr)
	htmlRdr := ctx.Reply().Rdr.(*HTML)

	assert.Equal(t, "master.html", htmlRdr.Layout)
	assert.Equal(t, "pages_app_index.html", htmlRdr.Template.Name())
	assert.Equal(t, "http", htmlRdr.ViewArgs["Scheme"])
	assert.Equal(t, "localhost:8080", htmlRdr.ViewArgs["Host"])
	assert.Equal(t, "/index.html", htmlRdr.ViewArgs["RequestPath"])
	assert.Equal(t, Version, htmlRdr.ViewArgs["AahVersion"])
	assert.Equal(t, "aah framework", htmlRdr.ViewArgs["MyName"])

	// User provided template file
	ctx.Reply().HTMLf("/admin/index.html", Data{})
	e.resolveView(ctx)
	htmlRdr = ctx.Reply().Rdr.(*HTML)
	assert.Equal(t, "/admin/index.html", htmlRdr.Filename)
	assert.Equal(t, "View Not Found: views/pages/admin/index.html", htmlRdr.ViewArgs["ViewNotFound"])

	// User provided template file with controller context
	ctx.Reply().HTMLf("user/index.html", Data{})
	e.resolveView(ctx)
	htmlRdr = ctx.Reply().Rdr.(*HTML)
	assert.Equal(t, "user/index.html", htmlRdr.Filename)
	assert.Equal(t, "View Not Found: views/pages/app/user/index.html", htmlRdr.ViewArgs["ViewNotFound"])

	// Namespace/Sub-package
	appIsProfileProd = true
	ctx.controller = &controllerInfo{Type: reflect.TypeOf(AppController{}), Namespace: "frontend"}
	ctx.Reply().HTMLf("index.html", Data{})
	e.resolveView(ctx)
	htmlRdr = ctx.Reply().Rdr.(*HTML)
	assert.Equal(t, "index.html", htmlRdr.Filename)
	assert.Equal(t, "View Not Found", htmlRdr.ViewArgs["ViewNotFound"])
	appIsProfileProd = false

	ctx.Req.AcceptContentType.Mime = ""
	appConfig = appCfg
	assert.Nil(t, identifyContentType(ctx))

	// cleanup
	appViewEngine = nil
	appConfig = nil
}

func TestViewResolveViewNotFound(t *testing.T) {
	e := &engine{}
	appViewEngine = &view.GoViewEngine{}

	req := httptest.NewRequest("GET", "http://localhost:8080/index.html", nil)
	type AppController struct{}
	ctx := &Context{
		Req:        ahttp.ParseRequest(req, &ahttp.Request{}),
		controller: &controllerInfo{Type: reflect.TypeOf(AppController{}), Namespace: "site"},
		action: &MethodInfo{
			Name:       "Index",
			Parameters: []*ParameterInfo{},
		},
		reply:   NewReply(),
		subject: security.AcquireSubject(),
	}
	ctx.Reply().ContentType(ahttp.ContentTypeHTML.Raw())
	appViewExt = ".html"

	e.resolveView(ctx)

	assert.NotNil(t, ctx.Reply().Rdr)
	htmlRdr := ctx.Reply().Rdr.(*HTML)
	assert.NotNil(t, htmlRdr.Template)

	// cleanup
	appViewEngine = nil
}

func TestViewDefaultContentType(t *testing.T) {
	appConfig, _ = config.ParseString("")
	assert.Nil(t, defaultContentType())

	appConfig, _ = config.ParseString(`
  render {
    default = "html"
  }
  `)

	v1 := defaultContentType()
	assert.Equal(t, "text/html; charset=utf-8", v1.Raw())

	AppConfig().SetString("render.default", "xml")
	v2 := defaultContentType()
	assert.Equal(t, "application/xml; charset=utf-8", v2.Raw())

	AppConfig().SetString("render.default", "json")
	v3 := defaultContentType()
	assert.Equal(t, "application/json; charset=utf-8", v3.Raw())

	AppConfig().SetString("render.default", "text")
	v4 := defaultContentType()
	assert.Equal(t, "text/plain; charset=utf-8", v4.Raw())

	// cleanup
	appConfig = nil
}

func TestViewSetMinifier(t *testing.T) {
	testMinifier := func(contentType string, w io.Writer, r io.Reader) error {
		t.Log("called minifier func")
		return nil
	}

	assert.Nil(t, minifier)
	SetMinifier(testMinifier)
	assert.NotNil(t, minifier)

	SetMinifier(testMinifier)
	assert.NotNil(t, minifier)
}
