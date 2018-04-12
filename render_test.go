// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aahframework.org/essentials.v0"
	"aahframework.org/test.v0/assert"
)

func TestRenderText(t *testing.T) {
	buf := &bytes.Buffer{}
	text1 := textRender{
		Format: "welcome to %s %s",
		Values: []interface{}{"aah", "framework"},
	}

	err := text1.Render(buf)
	assert.FailOnError(t, err, "")
	assert.Equal(t, "welcome to aah framework", buf.String())

	buf.Reset()
	text2 := textRender{Format: "welcome to aah framework"}

	err = text2.Render(buf)
	assert.FailOnError(t, err, "")
	assert.Equal(t, "welcome to aah framework", buf.String())
}

func TestRenderJSON(t *testing.T) {
	buf := acquireBuffer()

	data := struct {
		Name    string
		Age     int
		Address string
	}{
		Name:    "John",
		Age:     28,
		Address: "this is my street",
	}

	a := newApp()
	reply := newReply(&Context{a: a})
	json1 := jsonRender{Data: data, r: reply}
	err := json1.Render(buf)
	assert.FailOnError(t, err, "")
	assert.Equal(t, `{"Name":"John","Age":28,"Address":"this is my street"}`,
		buf.String())
}

// func TestRenderJSONP(t *testing.T) {
// 	buf := acquireBuffer()
//
// 	data := struct {
// 		Name    string
// 		Age     int
// 		Address string
// 	}{
// 		Name:    "John",
// 		Age:     28,
// 		Address: "this is my street",
// 	}
//
// 	renderPretty = true
// 	json1 := jsonpRender{Data: data, Callback: "mycallback"}
// 	err := json1.Render(buf)
// 	assert.FailOnError(t, err, "")
// 	assert.Equal(t, `mycallback({
//     "Name": "John",
//     "Age": 28,
//     "Address": "this is my street"
// });`, buf.String())
//
// 	buf.Reset()
// 	renderPretty = false
//
// 	err = json1.Render(buf)
// 	assert.FailOnError(t, err, "")
// 	assert.Equal(t, `mycallback({"Name":"John","Age":28,"Address":"this is my street"});`,
// 		buf.String())
// }
//
// func TestRenderXML(t *testing.T) {
// 	buf := acquireBuffer()
//
// 	type Sample struct {
// 		Name    string
// 		Age     int
// 		Address string
// 	}
//
// 	data := Sample{
// 		Name:    "John",
// 		Age:     28,
// 		Address: "this is my street",
// 	}
//
// 	renderPretty = true
// 	xml1 := xmlRender{Data: data}
// 	err := xml1.Render(buf)
// 	assert.FailOnError(t, err, "")
// 	assert.Equal(t, `<?xml version="1.0" encoding="UTF-8"?>
// <Sample>
//     <Name>John</Name>
//     <Age>28</Age>
//     <Address>this is my street</Address>
// </Sample>`, buf.String())
//
// 	buf.Reset()
//
// 	renderPretty = false
//
// 	err = xml1.Render(buf)
// 	assert.FailOnError(t, err, "")
// 	assert.Equal(t, `<?xml version="1.0" encoding="UTF-8"?>
// <Sample><Name>John</Name><Age>28</Age><Address>this is my street</Address></Sample>`,
// 		buf.String())
// }
//
func TestRenderFailureXML(t *testing.T) {
	buf := new(bytes.Buffer)

	data := struct {
		Name    string
		Age     int
		Address string
	}{
		Name:    "John",
		Age:     28,
		Address: "this is my street",
	}

	a := newApp()
	reply := newReply(&Context{a: a})
	xml1 := xmlRender{Data: data, r: reply}
	err := xml1.Render(buf)
	assert.Equal(t, "xml: unsupported type: struct { Name string; Age int; Address string }", err.Error())
}

func TestRenderFileNotExistsAndDir(t *testing.T) {
	buf := new(bytes.Buffer)

	// Directory error
	// buf.Reset()
	file1 := binaryRender{Path: os.Getenv("HOME")}
	err := file1.Render(buf)
	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), "is a directory"))
	assert.True(t, ess.IsStrEmpty(buf.String()))

	// File not exists
	file2 := binaryRender{Path: filepath.Join(testdataBaseDir(), "file-not-exists.txt")[1:]}
	err = file2.Render(buf)
	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), "file-not-exists.txt: no such file or directory"))
	assert.True(t, ess.IsStrEmpty(buf.String()))
}

func TestHTMLRenderTmplNil(t *testing.T) {
	// Template is Nil
	htmlTmplNil := htmlRender{
		Layout: "master",
	}

	var buf bytes.Buffer
	err := htmlTmplNil.Render(&buf)
	assert.NotNil(t, err)
	assert.Equal(t, "template is nil", err.Error())
}
