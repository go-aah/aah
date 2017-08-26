// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/view source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package view

import (
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aahframework.org/config.v0"
	"aahframework.org/test.v0/assert"
)

func TestViewAddTemplateFunc(t *testing.T) {
	AddTemplateFunc(template.FuncMap{
		"join":     strings.Join,
		"safeHTML": strings.Join, // for duplicate test, don't mind
	})

	_, found := TemplateFuncMap["join"]
	assert.True(t, found)
}

func TestViewStore(t *testing.T) {
	err := AddEngine("go", &GoViewEngine{})
	assert.NotNil(t, err)
	assert.Equal(t, "view: engine name 'go' is already added, skip it", err.Error())

	err = AddEngine("custom", nil)
	assert.NotNil(t, err)
	assert.Equal(t, "view: engine value is nil", err.Error())

	engine, found := GetEngine("go")
	assert.NotNil(t, engine)
	assert.True(t, found)

	engine, found = GetEngine("myengine")
	assert.Nil(t, engine)
	assert.False(t, found)
}

func TestViewCommonTemplateInit(t *testing.T) {
	c := &CommonTemplate{}
	cfg, _ := config.ParseString(`view { }`)

	err := c.Init(cfg, filepath.Join(getTestdataPath(), "common-not-exists"))
	assert.Nil(t, err)
}

func TestViewTemplateKey(t *testing.T) {
	key1 := TemplateKey(getTestdataPath() + "/views/pages/app/index.html")
	assert.Equal(t, "pages_app_index.html", key1)

	fSeparator = '\\'
	key2 := TemplateKey(strings.Replace(getTestdataPath()+"/views/pages/app/index.html", "/", "\\", -1))
	assert.Equal(t, "pages_app_index.html", key2)

	key3 := parseKey(strings.Replace(getTestdataPath(), "/", "\\", -1),
		strings.Replace(getTestdataPath()+"/views/pages/app/index.html", "/", "\\", -1))
	assert.Equal(t, "views_pages_app_index.html", key3)
}

func getTestdataPath() string {
	wd, _ := os.Getwd()
	return filepath.Join(wd, "testdata")
}
