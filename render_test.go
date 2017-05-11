// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"bytes"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/test.v0/assert"
)

func TestRenderText(t *testing.T) {
	buf := &bytes.Buffer{}
	text1 := Text{
		Format: "welcome to %s %s",
		Values: []interface{}{"aah", "framework"},
	}

	err := text1.Render(buf)
	assert.FailOnError(t, err, "")
	assert.Equal(t, "welcome to aah framework", buf.String())

	buf.Reset()
	text2 := Text{Format: "welcome to aah framework"}

	err = text2.Render(buf)
	assert.FailOnError(t, err, "")
	assert.Equal(t, "welcome to aah framework", buf.String())
}

func TestRenderJSON(t *testing.T) {
	buf := &bytes.Buffer{}
	appConfig = getRenderCfg()

	data := struct {
		Name    string
		Age     int
		Address string
	}{
		Name:    "John",
		Age:     28,
		Address: "this is my street",
	}

	json1 := JSON{Data: data}
	err := json1.Render(buf)
	assert.FailOnError(t, err, "")
	assert.Equal(t, `{
    "Name": "John",
    "Age": 28,
    "Address": "this is my street"
}`, buf.String())

	buf.Reset()
	appConfig.SetBool("render.pretty", false)

	err = json1.Render(buf)
	assert.FailOnError(t, err, "")
	assert.Equal(t, `{"Name":"John","Age":28,"Address":"this is my street"}`,
		buf.String())
}

func TestRenderJSONP(t *testing.T) {
	buf := &bytes.Buffer{}
	appConfig = getRenderCfg()

	data := struct {
		Name    string
		Age     int
		Address string
	}{
		Name:    "John",
		Age:     28,
		Address: "this is my street",
	}

	json1 := JSON{Data: data, IsJSONP: true, Callback: "mycallback"}
	err := json1.Render(buf)
	assert.FailOnError(t, err, "")
	assert.Equal(t, `mycallback({
    "Name": "John",
    "Age": 28,
    "Address": "this is my street"
});`, buf.String())

	buf.Reset()
	appConfig.SetBool("render.pretty", false)

	err = json1.Render(buf)
	assert.FailOnError(t, err, "")
	assert.Equal(t, `mycallback({"Name":"John","Age":28,"Address":"this is my street"});`,
		buf.String())
}

func TestRenderXML(t *testing.T) {
	buf := &bytes.Buffer{}
	appConfig = getRenderCfg()

	type Sample struct {
		Name    string
		Age     int
		Address string
	}

	data := Sample{
		Name:    "John",
		Age:     28,
		Address: "this is my street",
	}

	xml1 := XML{Data: data}
	err := xml1.Render(buf)
	assert.FailOnError(t, err, "")
	assert.Equal(t, `<Sample>
    <Name>John</Name>
    <Age>28</Age>
    <Address>this is my street</Address>
</Sample>`, buf.String())

	buf.Reset()

	appConfig.SetBool("render.pretty", false)

	err = xml1.Render(buf)
	assert.FailOnError(t, err, "")
	assert.Equal(t, `<Sample><Name>John</Name><Age>28</Age><Address>this is my street</Address></Sample>`,
		buf.String())
}

func TestRenderFailureXML(t *testing.T) {
	buf := &bytes.Buffer{}
	appConfig = getRenderCfg()

	data := struct {
		Name    string
		Age     int
		Address string
	}{
		Name:    "John",
		Age:     28,
		Address: "this is my street",
	}

	xml1 := XML{Data: data}
	err := xml1.Render(buf)
	assert.Equal(t, "xml: unsupported type: struct { Name string; Age int; Address string }", err.Error())
}

func TestRenderFileAndReader(t *testing.T) {
	f, _ := os.Open(getRenderFilepath("file1.txt"))
	defer ess.CloseQuietly(f)

	buf := &bytes.Buffer{}
	file1 := Binary{Reader: f}

	err := file1.Render(buf)
	assert.FailOnError(t, err, "")
	assert.Equal(t, `
Each incoming request passes through a pre-defined list of steps
`, buf.String())

	// Reader
	buf.Reset()
	file2 := Binary{Path: getRenderFilepath("file1.txt")}
	err = file2.Render(buf)
	assert.FailOnError(t, err, "")
	assert.Equal(t, `
Each incoming request passes through a pre-defined list of steps
`, buf.String())

	// Reader string
	buf.Reset()
	file3 := Binary{Reader: strings.NewReader(`<Sample><Name>John</Name><Age>28</Age><Address>this is my street</Address></Sample>`)}
	err = file3.Render(buf)
	assert.FailOnError(t, err, "")
	assert.Equal(t, `<Sample><Name>John</Name><Age>28</Age><Address>this is my street</Address></Sample>`,
		buf.String())

	// Directory error
	buf.Reset()
	file4 := Binary{Path: os.Getenv("HOME")}
	err = file4.Render(buf)
	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), "is a directory"))
	assert.True(t, ess.IsStrEmpty(buf.String()))

	// File not exists
	file5 := Binary{Path: filepath.Join(getTestdataPath(), "file-not-exists.txt")[1:]}
	err = file5.Render(buf)
	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), "file-not-exists.txt: no such file or directory"))
	assert.True(t, ess.IsStrEmpty(buf.String()))
}

func TestHTMLRender(t *testing.T) {
	tmplStr := `
	{{ define "title" }}<title>This is test title</title>{{ end }}
	{{ define "body" }}<p>This is test body</p>{{ end }}
	`

	tmpl := template.Must(template.New("test").Parse(tmplStr))
	assert.NotNil(t, tmpl)

	masterTmpl := getRenderFilepath(filepath.Join("views", "master.html"))
	_, err := tmpl.ParseFiles(masterTmpl)
	assert.Nil(t, err)

	htmlRdr := HTML{
		Layout:   "master",
		Template: tmpl,
		ViewArgs: nil,
	}

	var buf bytes.Buffer
	err = htmlRdr.Render(&buf)
	assert.Nil(t, err)
	assert.True(t, strings.Contains(buf.String(), "<title>This is test title</title>"))
	assert.True(t, strings.Contains(buf.String(), "<p>This is test body</p>"))

	buf.Reset()
	htmlRdr.Layout = ""
	err = htmlRdr.Render(&buf)
	assert.Nil(t, err)
	assert.True(t, ess.IsStrEmpty(buf.String()))

	// Template is Nil
	htmlTmplNil := HTML{
		Layout: "master",
	}

	buf.Reset()
	err = htmlTmplNil.Render(&buf)
	assert.NotNil(t, err)
	assert.Equal(t, "template is nil", err.Error())
}

func getRenderCfg() *config.Config {
	cfg, _ := config.ParseString(`
  render {
    pretty = true
  }
    `)
	return cfg
}

func getRenderFilepath(name string) string {
	wd, _ := os.Getwd()
	return filepath.Join(wd, "testdata", "render", name)
}
