// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package atemplate

import (
	"bytes"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aahframework.org/config"
	"aahframework.org/test/assert"
)

func TestAppPagesTemplates(t *testing.T) {
	cfg := getTemplateConfig()
	te := loadTemplates(t, cfg)

	data := map[string]string{
		"GreetName": "aah framework",
		"PageName":  "home page",
	}

	tmpl := te.Get("master", "pages_app", "index.html")
	assert.NotNil(t, tmpl)

	var buf bytes.Buffer
	err := tmpl.ExecuteTemplate(&buf, "master", data)
	assert.FailOnError(t, err, "")

	htmlStr := buf.String()
	assert.True(t, strings.Contains(htmlStr, "<title>aah framework - Home</title>"))
	assert.True(t, strings.Contains(htmlStr, "aah framework home page"))

	tmpl = te.Get("no_master", "pages_app", "index.html")
	assert.Nil(t, tmpl)
}

func TestUserPagesTemplates(t *testing.T) {
	cfg := getTemplateConfig()
	te := loadTemplates(t, cfg)

	data := map[string]string{
		"GreetName": "aah framework",
		"PageName":  "user home page",
	}

	cfg.SetBool("template.case_sensitive", true)

	tmpl := te.Get("master", "pages_user", "index.html")
	assert.NotNil(t, tmpl)

	var buf bytes.Buffer
	err := tmpl.ExecuteTemplate(&buf, "master", data)
	assert.FailOnError(t, err, "")

	htmlStr := buf.String()
	assert.True(t, strings.Contains(htmlStr, "<title>aah framework - User Home</title>"))
	assert.True(t, strings.Contains(htmlStr, "aah framework user home page"))
	assert.True(t, strings.Contains(htmlStr, `cdnjs.cloudflare.com/ajax/libs/jquery/2.2.4/jquery.min.js`))

	tmpl = te.Get("master", "pages_user", "not_exists.html")
	assert.Nil(t, tmpl)
}

func TestRelaod(t *testing.T) {
	cfg := getTemplateConfig()
	te := loadTemplates(t, cfg)

	err := te.Reload()
	assert.FailOnError(t, err, "")

	assert.NotNil(t, te.baseDir)
	assert.NotNil(t, te.appConfig)
	assert.NotNil(t, te.layouts)
}

func TestAddTemplateFunc(t *testing.T) {
	AddTemplateFunc(template.FuncMap{
		"join": strings.Join,
	})
}

func TestBaseDirNotExists(t *testing.T) {
	viewsDir := filepath.Join(getTestdataPath(), "views1")
	te := &TemplateEngine{}

	te.Init(getTemplateConfig(), viewsDir)
	err := te.Load()
	assert.NotNil(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "views base dir is not exists:"))
}

func loadTemplates(t *testing.T, cfg *config.Config) *TemplateEngine {
	viewsDir := filepath.Join(getTestdataPath(), "views")
	te := &TemplateEngine{}

	te.Init(cfg, viewsDir)

	assert.Equal(t, viewsDir, te.baseDir)
	assert.NotNil(t, te.appConfig)
	assert.NotNil(t, te.layouts)

	err := te.Load()
	assert.FailOnError(t, err, "")

	return te
}

func getTemplateConfig() *config.Config {
	cfg, _ := config.ParseString(`
template {
  ext = ".html"

  # Default is false
  # "/views/app/login.tmpl" == "/views/App/Login.tmpl"
  case_sensitive = false

  delimiters = "{{.}}"
}
    `)

	return cfg
}

func getTestdataPath() string {
	wd, _ := os.Getwd()
	return filepath.Join(wd, "testdata")
}
