// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org/view source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package view

import (
	"bytes"
	"errors"
	"html/template"
	"io/ioutil"
	"strings"
	"testing"

	"aahframe.work/aah/config"
	"aahframe.work/aah/log"
	"github.com/stretchr/testify/assert"
)

func TestViewAppPages(t *testing.T) {
	// _ = log.SetLevel("trace")
	log.SetWriter(ioutil.Discard)
	cfg, _ := config.ParseString(`view { }`)
	ge := loadGoViewEngine(t, cfg, "views", false)

	data := map[string]interface{}{
		"GreetName": "aah framework",
		"PageName":  "home page",
	}

	tmpl, err := ge.Get("master.html", "pages/app", "index.html")
	assert.Nil(t, err)
	assert.NotNil(t, tmpl)

	var buf bytes.Buffer
	err = tmpl.ExecuteTemplate(&buf, "master.html", data)
	assert.Nil(t, err, "")

	htmlStr := buf.String()
	t.Logf("HTML String: %s", htmlStr)
	assert.True(t, strings.Contains(htmlStr, "<title>aah framework - Home</title>"))
	assert.True(t, strings.Contains(htmlStr, "aah framework home page"))

	tmpl, err = ge.Get("no_master", "pages/app", "index.html")
	assert.NotNil(t, err)
	assert.Nil(t, tmpl)
}

func TestViewUserPages(t *testing.T) {
	// _ = log.SetLevel("trace")
	log.SetWriter(ioutil.Discard)
	cfg, _ := config.ParseString(`view {
		delimiters = "{{.}}"
	}`)
	ge := loadGoViewEngine(t, cfg, "views", true)

	data := map[string]interface{}{
		"GreetName": "aah framework",
		"PageName":  "user home page",
	}

	ge.CaseSensitive = true

	tmpl, err := ge.Get("master.html", "pages/user", "index.html")
	assert.Nil(t, err)
	assert.NotNil(t, tmpl)

	var buf bytes.Buffer
	err = tmpl.ExecuteTemplate(&buf, "master.html", data)
	assert.Nil(t, err, "")

	htmlStr := buf.String()
	t.Logf("HTML String: %s", htmlStr)
	assert.True(t, strings.Contains(htmlStr, "<title>aah framework - User Home</title>"))
	assert.True(t, strings.Contains(htmlStr, "aah framework user home page"))
	assert.True(t, strings.Contains(htmlStr, `cdnjs.cloudflare.com/ajax/libs/jquery/2.2.4/jquery.min.js`))

	tmpl, err = ge.Get("master.html", "pages/user", "not_exists.html")
	assert.NotNil(t, err)
	assert.Nil(t, tmpl)
}

func TestViewUserPagesNoLayout(t *testing.T) {
	// _ = log.SetLevel("trace")
	log.SetWriter(ioutil.Discard)
	cfg, _ := config.ParseString(`view {
		delimiters = "{{.}}"
		default_layout = false
	}`)
	ge := loadGoViewEngine(t, cfg, "views", false)

	data := map[string]interface{}{
		"GreetName": "aah framework",
		"PageName":  "user home page",
	}

	tmpl, err := ge.Get("", "pages/user", "index-nolayout.html")
	assert.Nil(t, err)
	assert.NotNil(t, tmpl)

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	assert.Nil(t, err, "")

	htmlStr := buf.String()
	t.Logf("HTML String: %s", htmlStr)
	assert.True(t, strings.Contains(htmlStr, "aah framework user home page - no layout"))
}

func TestViewBaseDirNotExists(t *testing.T) {
	viewsDir := join("testdata", "views1")
	ge := &GoViewEngine{}
	cfg, _ := config.ParseString(`view { }`)

	err := ge.Init(newVFS(), cfg, viewsDir)
	assert.NotNil(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "goviewengine: views base dir is not exists:"))
}

func TestViewDelimitersError(t *testing.T) {
	viewsDir := join("testdata", "views")
	ge := &GoViewEngine{}
	cfg, _ := config.ParseString(`view {
		delimiters = "{{."
	}`)

	err := ge.Init(newVFS(), cfg, viewsDir)
	assert.NotNil(t, err)
	assert.Equal(t, "goviewengine: config 'view.delimiters' value is invalid", err.Error())
}

func TestViewErrors(t *testing.T) {
	// _ = log.SetLevel("trace")
	log.SetWriter(ioutil.Discard)
	cfg, _ := config.ParseString(`view {
		default_layout = false
	}`)

	fs := newVFS()

	// No layout directiry
	viewsDir := join("testdata", "views-no-layouts-dir")
	ge := &GoViewEngine{}
	err := ge.Init(fs, cfg, viewsDir)
	assert.NotNil(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "goviewengine: layouts base dir is not exists:"))

	// No Common directory
	viewsDir = join("testdata", "views-no-common-dir")
	ge = &GoViewEngine{}
	err = ge.Init(fs, cfg, viewsDir)
	assert.NotNil(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "goviewengine: common base dir is not exists:"))

	// No Pages directory
	viewsDir = join("testdata", "views-no-pages-dir")
	ge = &GoViewEngine{}
	err = ge.Init(fs, cfg, viewsDir)
	assert.NotNil(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "goviewengine: pages base dir is not exists:"))

	// handle errors methods
	err = ge.ParseErrors([]error{errors.New("error 1"), errors.New("error 2")})
	assert.NotNil(t, err)
	assert.Equal(t, "goviewengine: error processing templates, please check the log", err.Error())
}

func loadGoViewEngine(t *testing.T, cfg *config.Config, dir string, hotreload bool) *GoViewEngine {
	// dummy func for test
	AddTemplateFunc(template.FuncMap{
		"anticsrftoken": func(arg interface{}) string {
			return ""
		},
		"rurl": func(args map[string]interface{}, key string) string {
			return "//localhost:8080/login"
		},
		"qparam": func(args map[string]interface{}, key string) string {
			return "/index"
		},
	})

	viewsDir := join("testdata", dir)
	ge := &GoViewEngine{}

	err := ge.Init(newVFS(), cfg, viewsDir)
	assert.Nil(t, err, "")
	ge.hotReload = hotreload

	assert.Equal(t, viewsDir, ge.BaseDir)
	assert.NotNil(t, ge.AppConfig)
	assert.NotNil(t, ge.Templates)

	assert.NotNil(t, (&EngineBase{}).Init(nil, nil, "", "", ""))

	log.SetWriter(ioutil.Discard)

	return ge
}
