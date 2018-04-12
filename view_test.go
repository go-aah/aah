// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"io"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"aahframework.org/ahttp.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/test.v0/assert"
	"aahframework.org/view.v0"
)

func TestViewStore(t *testing.T) {
	importPath := filepath.Join(testdataBaseDir(), "webapp1")
	ts, err := newTestServer(t, importPath)
	assert.Nil(t, err)
	defer ts.Close()

	t.Logf("Test Server URL [View Store]: %s", ts.URL)

	err = ts.app.AddViewEngine("go", &view.GoViewEngine{})
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
	ts, err := newTestServer(t, importPath)
	assert.Nil(t, err)
	defer ts.Close()

	t.Logf("Test Server URL [Resolve View]: %s", ts.URL)

	vm := ts.app.viewMgr
	assert.NotNil(t, vm)
	assert.NotNil(t, vm.engine)

	req := httptest.NewRequest(ahttp.MethodGet, ts.URL, nil)
	ctx := newContext(httptest.NewRecorder(), req)
	ctx.a = ts.app

	type AppController struct{}
	cType := reflect.TypeOf(AppController{})
	ctx.controller = &controllerInfo{Name: cType.Name(), Type: cType, NoSuffixName: "app"}
	ctx.action = &MethodInfo{Name: "Index", Parameters: []*ParameterInfo{}}
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
	ts.app.envProfile = "prod"
	ctx.controller = &controllerInfo{Type: reflect.TypeOf(AppController{}), Namespace: "frontend"}
	ctx.Reply().HTMLf("index.html", Data{})
	vm.resolve(ctx)
	htmlRdr = ctx.Reply().Rdr.(*htmlRender)
	assert.Equal(t, "index.html", htmlRdr.Filename)
	assert.Equal(t, "View Not Found", htmlRdr.ViewArgs["ViewNotFound"])
	ts.app.envProfile = "dev"
}

func TestViewMinifier(t *testing.T) {
	defer ess.DeleteFiles("webapp1.pid")

	importPath := filepath.Join(testdataBaseDir(), "webapp1")
	ts, err := newTestServer(t, importPath)
	assert.Nil(t, err)
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

func TestViewDefaultContentType(t *testing.T) {
	assert.Nil(t, resolveDefaultContentType(""))

	v1 := resolveDefaultContentType("html")
	assert.Equal(t, "text/html; charset=utf-8", v1.Raw())

	v2 := resolveDefaultContentType("xml")
	assert.Equal(t, "application/xml; charset=utf-8", v2.Raw())

	v3 := resolveDefaultContentType("json")
	assert.Equal(t, "application/json; charset=utf-8", v3.Raw())

	v4 := resolveDefaultContentType("text")
	assert.Equal(t, "text/plain; charset=utf-8", v4.Raw())
}
