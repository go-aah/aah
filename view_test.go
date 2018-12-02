// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"io"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"aahframe.work/ahttp"
	"aahframe.work/ainsp"
	"aahframe.work/essentials"
	"aahframe.work/view"
	"github.com/stretchr/testify/assert"
)

func TestViewStore(t *testing.T) {
	importPath := filepath.Join(testdataBaseDir(), "webapp1")
	ts := newTestServer(t, importPath)
	defer ts.Close()

	t.Logf("Test Server URL [View Store]: %s", ts.URL)

	err := ts.app.AddViewEngine("go", &view.GoViewEngine{})
	assert.NotNil(t, err)
	assert.Equal(t, "view: engine name 'go' is already added, skip it", err.Error())

	err = ts.app.AddViewEngine("custom", nil)
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
	defer ess.DeleteFiles("webapp1.pid")

	importPath := filepath.Join(testdataBaseDir(), "webapp1")
	ts := newTestServer(t, importPath)
	defer ts.Close()

	t.Logf("Test Server URL [Resolve View]: %s", ts.URL)

	vm := ts.app.viewMgr
	assert.NotNil(t, vm)
	assert.NotNil(t, vm.engine)
	vm.setHotReload(false)

	req := httptest.NewRequest(ahttp.MethodGet, ts.URL, nil)
	ctx := newContext(httptest.NewRecorder(), req)
	ctx.a = ts.app

	type AppController struct{}
	cType := reflect.TypeOf(AppController{})
	ctx.controller = &ainsp.Target{Name: cType.Name(), Type: cType, NoSuffixName: "app"}
	ctx.action = &ainsp.Method{Name: "Index", Parameters: []*ainsp.Parameter{}}
	ctx.Reply().ContentType(ahttp.ContentTypeHTML.Raw())
	ctx.AddViewArg("MyName", "aah framework")

	t.Log("Template exists")
	vm.resolve(ctx)
	assert.NotNil(t, ctx.Reply().Rdr)
	htmlRdr := ctx.Reply().Rdr.(*htmlRender)
	assert.Equal(t, "master.html", htmlRdr.Layout)
	assert.Equal(t, "pages/app/index.html", htmlRdr.Template.Name())
	assert.Equal(t, "http", htmlRdr.ViewArgs["Scheme"])
	assert.True(t, strings.Contains(ts.URL, htmlRdr.ViewArgs["Host"].(string)))
	assert.Equal(t, "", htmlRdr.ViewArgs["RequestPath"])
	assert.Equal(t, Version, htmlRdr.ViewArgs["AahVersion"])
	assert.Equal(t, "aah framework", htmlRdr.ViewArgs["MyName"])
	assert.True(t, htmlRdr.ViewArgs["ClientIP"].(string) != "")

	// User provided template file
	t.Log("User provided template file")
	ctx.Reply().HTMLf("/admin/index.html", Data{})
	vm.resolve(ctx)
	htmlRdr = ctx.Reply().Rdr.(*htmlRender)
	assert.Equal(t, "/admin/index.html", htmlRdr.Filename)
	assert.Equal(t, "View Not Found: views/pages/admin/index.html", htmlRdr.ViewArgs["ViewNotFound"])

	// User provided template file with controller context
	t.Log("User provided template file with controller context")
	ctx.Reply().HTMLf("user/index.html", Data{})
	vm.resolve(ctx)
	htmlRdr = ctx.Reply().Rdr.(*htmlRender)
	assert.Equal(t, "user/index.html", htmlRdr.Filename)
	assert.Equal(t, "View Not Found: views/pages/app/user/index.html", htmlRdr.ViewArgs["ViewNotFound"])

	// Namespace/Sub-package
	t.Log("Namespace/Sub-package")
	ts.app.settings.EnvProfile = "prod"
	ctx.controller = &ainsp.Target{Type: reflect.TypeOf(AppController{}), Namespace: "frontend"}
	ctx.Reply().HTMLf("index.html", Data{})
	vm.resolve(ctx)
	htmlRdr = ctx.Reply().Rdr.(*htmlRender)
	assert.Equal(t, "index.html", htmlRdr.Filename)
	assert.Equal(t, "View Not Found", htmlRdr.ViewArgs["ViewNotFound"])
	ts.app.settings.EnvProfile = "dev"
}

func TestViewMinifier(t *testing.T) {
	defer ess.DeleteFiles("webapp1.pid")

	importPath := filepath.Join(testdataBaseDir(), "webapp1")
	ts := newTestServer(t, importPath)
	defer ts.Close()

	t.Logf("Test Server URL [View Minifier]: %s", ts.URL)

	assert.NotNil(t, ts.app.viewMgr)
	assert.Nil(t, ts.app.viewMgr.minifier)
	ts.app.SetMinifier(func(contentType string, w io.Writer, r io.Reader) error {
		t.Log(contentType, w, r)
		return nil
	})
	assert.NotNil(t, ts.app.viewMgr.minifier)

	t.Log("Second set")
	ts.app.SetMinifier(func(contentType string, w io.Writer, r io.Reader) error {
		t.Log("this is second set", contentType, w, r)
		return nil
	})
}
