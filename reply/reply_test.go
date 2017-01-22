// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package reply

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"aahframework.org/aah/ahttp"
	"aahframework.org/aah/render"
	"aahframework.org/config"
	"aahframework.org/test/assert"
)

func TestTextReply(t *testing.T) {
	buf, re1 := getBufferAndReply()
	re1.Text("welcome to %s %s", "aah", "framework")

	err := re1.Rdr.Render(buf)
	assert.FailOnError(t, err, "")
	assert.Equal(t, "welcome to aah framework", buf.String())

	buf.Reset()

	re2 := Reply{}
	re2.Text("welcome to aah framework")

	err = re2.Rdr.Render(buf)
	assert.FailOnError(t, err, "")
	assert.Equal(t, "welcome to aah framework", buf.String())
}

func TestJSONReply(t *testing.T) {
	buf, re1 := getBufferAndReply()
	cfg := getReplyRenderCfg()
	render.Init(cfg)

	data := struct {
		Name    string
		Age     int
		Address string
	}{
		Name:    "John",
		Age:     28,
		Address: "this is my street",
	}

	re1.JSON(data)
	err := re1.Rdr.Render(buf)
	assert.FailOnError(t, err, "")
	assert.Equal(t, `{
    "Name": "John",
    "Age": 28,
    "Address": "this is my street"
}`, buf.String())

	buf.Reset()

	cfg.SetBool("render.pretty", false)

	err = re1.Rdr.Render(buf)
	assert.FailOnError(t, err, "")
	assert.Equal(t, `{"Name":"John","Age":28,"Address":"this is my street"}`,
		buf.String())
}

func TestJSONPReply(t *testing.T) {
	buf, re1 := getBufferAndReply()
	cfg := getReplyRenderCfg()
	render.Init(cfg)

	data := struct {
		Name    string
		Age     int
		Address string
	}{
		Name:    "John",
		Age:     28,
		Address: "this is my street",
	}

	re1.JSONP(data, "mycallback")
	err := re1.Rdr.Render(buf)
	assert.FailOnError(t, err, "")
	assert.Equal(t, `mycallback({
    "Name": "John",
    "Age": 28,
    "Address": "this is my street"
});`, buf.String())

	buf.Reset()

	cfg.SetBool("render.pretty", false)

	err = re1.Rdr.Render(buf)
	assert.FailOnError(t, err, "")
	fmt.Println(buf.String())
	assert.Equal(t, `mycallback({"Name":"John","Age":28,"Address":"this is my street"});`,
		buf.String())
}

func TestXMLReply(t *testing.T) {
	buf, re1 := getBufferAndReply()
	cfg := getReplyRenderCfg()
	render.Init(cfg)

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

	re1.XML(data)
	err := re1.Rdr.Render(buf)
	assert.FailOnError(t, err, "")
	assert.Equal(t, `<Sample>
    <Name>John</Name>
    <Age>28</Age>
    <Address>this is my street</Address>
</Sample>`, buf.String())

	buf.Reset()

	cfg.SetBool("render.pretty", false)

	err = re1.Rdr.Render(buf)
	assert.FailOnError(t, err, "")
	assert.Equal(t, `<Sample><Name>John</Name><Age>28</Age><Address>this is my street</Address></Sample>`,
		buf.String())
}

func TestBytesReply(t *testing.T) {
	buf, re1 := getBufferAndReply()
	re1.Bytes(ahttp.ContentTypeXML.Raw(),
		[]byte(`<Sample><Name>John</Name><Age>28</Age><Address>this is my street</Address></Sample>`))

	err := re1.Rdr.Render(buf)
	assert.FailOnError(t, err, "")
	assert.Equal(t, `<Sample><Name>John</Name><Age>28</Age><Address>this is my street</Address></Sample>`,
		buf.String())
}

func TestAttachmentFileReply(t *testing.T) {
	f1, _ := os.Open(getFilepath("file1.txt"))
	defer func() {
		_ = f1.Close()
	}()

	buf, re1 := getBufferAndReply()
	re1.File("sample.txt", f1)

	err := re1.Rdr.Render(buf)
	assert.FailOnError(t, err, "")
	assert.Equal(t, `
Each incoming request passes through a pre-defined list of steps
`, buf.String())

	buf.Reset()

	f2, _ := os.Open(getFilepath("file1.txt"))
	defer func() {
		_ = f2.Close()
	}()
	re2 := &Reply{Hdr: http.Header{}}
	re2.FileInline("sample.txt", f2)

	err = re2.Rdr.Render(buf)
	assert.FailOnError(t, err, "")
	assert.Equal(t, `
Each incoming request passes through a pre-defined list of steps
`, buf.String())

}

func getReplyRenderCfg() *config.Config {
	cfg, _ := config.ParseString(`
  render {
    pretty = true
  }
    `)
	return cfg
}

func getBufferAndReply() (*bytes.Buffer, *Reply) {
	return &bytes.Buffer{}, NewReply()
}

func getFilepath(name string) string {
	wd, _ := os.Getwd()
	return filepath.Join(wd, "testdata", name)
}
