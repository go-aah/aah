// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/view source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package view

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"aahframework.org/config.v0"
	"aahframework.org/log.v0"
	"aahframework.org/test.v0/assert"
)

func TestViewAppPages(t *testing.T) {
	_ = log.SetLevel("trace")
	cfg, _ := config.ParseString(`view { }`)
	ge := loadGoViewEngine(t, cfg, "views")

	data := map[string]interface{}{
		"GreetName": "aah framework",
		"PageName":  "home page",
	}

	tmpl, err := ge.Get("master.html", "pages_app", "index.html")
	assert.Nil(t, err)
	assert.NotNil(t, tmpl)

	var buf bytes.Buffer
	err = tmpl.ExecuteTemplate(&buf, "master.html", data)
	assert.FailNowOnError(t, err, "")

	htmlStr := buf.String()
	t.Logf("HTML String: %s", htmlStr)
	assert.True(t, strings.Contains(htmlStr, "<title>aah framework - Home</title>"))
	assert.True(t, strings.Contains(htmlStr, "aah framework home page"))

	tmpl, err = ge.Get("no_master", "pages_app", "index.html")
	assert.NotNil(t, err)
	assert.Nil(t, tmpl)
}

func TestViewUserPages(t *testing.T) {
	cfg, _ := config.ParseString(`view {
		delimiters = "{{.}}"
	}`)
	ge := loadGoViewEngine(t, cfg, "views")

	data := map[string]interface{}{
		"GreetName": "aah framework",
		"PageName":  "user home page",
	}

	ge.caseSensitive = true

	tmpl, err := ge.Get("master.html", "pages_user", "index.html")
	assert.Nil(t, err)
	assert.NotNil(t, tmpl)

	var buf bytes.Buffer
	err = tmpl.ExecuteTemplate(&buf, "master.html", data)
	assert.FailNowOnError(t, err, "")

	htmlStr := buf.String()
	t.Logf("HTML String: %s", htmlStr)
	assert.True(t, strings.Contains(htmlStr, "<title>aah framework - User Home</title>"))
	assert.True(t, strings.Contains(htmlStr, "aah framework user home page"))
	assert.True(t, strings.Contains(htmlStr, `cdnjs.cloudflare.com/ajax/libs/jquery/2.2.4/jquery.min.js`))

	tmpl, err = ge.Get("master.html", "pages_user", "not_exists.html")
	assert.NotNil(t, err)
	assert.Nil(t, tmpl)
}

func TestViewUserPagesNoLayout(t *testing.T) {
	cfg, _ := config.ParseString(`view {
		delimiters = "{{.}}"
		default_layout = false
	}`)
	ge := loadGoViewEngine(t, cfg, "views")

	data := map[string]interface{}{
		"GreetName": "aah framework",
		"PageName":  "user home page",
	}

	tmpl, err := ge.Get("", "pages_user", "index-nolayout.html")
	assert.Nil(t, err)
	assert.NotNil(t, tmpl)

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	assert.FailNowOnError(t, err, "")

	htmlStr := buf.String()
	t.Logf("HTML String: %s", htmlStr)
	assert.True(t, strings.Contains(htmlStr, "aah framework user home page - no layout"))
}

func TestViewBaseDirNotExists(t *testing.T) {
	viewsDir := filepath.Join(getTestdataPath(), "views1")
	ge := &GoViewEngine{}
	cfg, _ := config.ParseString(`view { }`)

	err := ge.Init(cfg, viewsDir)
	assert.NotNil(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "goviewengine: views base dir is not exists:"))
}

func TestViewErrors(t *testing.T) {
	_ = log.SetLevel("trace")
	cfg, _ := config.ParseString(`view {
		default_layout = false
	}`)

	// Scenario 1
	viewsDir := filepath.Join(getTestdataPath(), "views-error1")
	ge := &GoViewEngine{}

	err := ge.Init(cfg, viewsDir)
	assert.NotNil(t, err)
	assert.NotNil(t, ge)

	// Scenario 2
	viewsDir = filepath.Join(getTestdataPath(), "views-error2")
	ge = &GoViewEngine{}

	err = ge.Init(cfg, viewsDir)
	assert.NotNil(t, err)
	assert.NotNil(t, ge)
}

func loadGoViewEngine(t *testing.T, cfg *config.Config, dir string) *GoViewEngine {
	viewsDir := filepath.Join(getTestdataPath(), dir)
	ge := &GoViewEngine{}

	err := ge.Init(cfg, viewsDir)
	assert.FailNowOnError(t, err, "")

	assert.Equal(t, viewsDir, ge.baseDir)
	assert.NotNil(t, ge.cfg)
	assert.NotNil(t, ge.layouts)

	return ge
}
