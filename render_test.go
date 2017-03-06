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

	"aahframework.org/config.v0-unstable"
	"aahframework.org/essentials.v0-unstable"
	"aahframework.org/test.v0-unstable/assert"
)

func TestTextRender(t *testing.T) {
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

func TestJSONRender(t *testing.T) {
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

func TestJSONPRender(t *testing.T) {
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

func TestXMLRender(t *testing.T) {
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

func TestFailureXMLRender(t *testing.T) {
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

func TestBytesRender(t *testing.T) {
	buf := &bytes.Buffer{}
	bytes1 := Bytes{Data: []byte(`<Sample><Name>John</Name><Age>28</Age><Address>this is my street</Address></Sample>`)}

	err := bytes1.Render(buf)
	assert.FailOnError(t, err, "")
	assert.Equal(t, `<Sample><Name>John</Name><Age>28</Age><Address>this is my street</Address></Sample>`,
		buf.String())
}

func TestFileRender(t *testing.T) {
	f, _ := os.Open(getRenderFilepath("file1.txt"))
	defer ess.CloseQuietly(f)

	buf := &bytes.Buffer{}
	file1 := File{Data: f}

	err := file1.Render(buf)
	assert.FailOnError(t, err, "")
	assert.Equal(t, `
Each incoming request passes through a pre-defined list of steps
`, buf.String())
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
