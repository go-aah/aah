// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package reply

import (
	"bytes"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aahframework.org/aah/ahttp"
	"aahframework.org/aah/render"
	"aahframework.org/config"
	ess "aahframework.org/essentials"
	"aahframework.org/test/assert"
)

func TestStatusCodes(t *testing.T) {
	re := NewReply()

	assert.False(t, re.IsStatusSet())

	re.Ok()
	assert.Equal(t, http.StatusOK, re.Code)

	re.Created()
	assert.Equal(t, http.StatusCreated, re.Code)

	re.Accepted()
	assert.Equal(t, http.StatusAccepted, re.Code)

	re.NoContent()
	assert.Equal(t, http.StatusNoContent, re.Code)

	re.MovedPermanently()
	assert.Equal(t, http.StatusMovedPermanently, re.Code)

	re.TemporaryRedirect()
	assert.Equal(t, http.StatusTemporaryRedirect, re.Code)

	re.BadRequest()
	assert.Equal(t, http.StatusBadRequest, re.Code)

	re.Unauthorized()
	assert.Equal(t, http.StatusUnauthorized, re.Code)

	re.Forbidden()
	assert.Equal(t, http.StatusForbidden, re.Code)

	re.NotFound()
	assert.Equal(t, http.StatusNotFound, re.Code)

	re.MethodNotAllowed()
	assert.Equal(t, http.StatusMethodNotAllowed, re.Code)

	re.Conflict()
	assert.Equal(t, http.StatusConflict, re.Code)

	re.InternalServerError()
	assert.Equal(t, http.StatusInternalServerError, re.Code)

	re.ServiceUnavailable()
	assert.Equal(t, http.StatusServiceUnavailable, re.Code)
}

func TestTextReply(t *testing.T) {
	buf, re1 := getBufferAndReply()

	re1.Text("welcome to %s %s", "aah", "framework")
	assert.True(t, re1.IsContentTypeSet())

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
	assert.True(t, re1.IsContentTypeSet())

	re1.Header(ahttp.HeaderContentType, "")
	assert.False(t, re1.IsContentTypeSet())

	re1.HeaderAppend(ahttp.HeaderContentType, ahttp.ContentTypePlainText.Raw())
	assert.True(t, re1.IsContentTypeSet())

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
	assert.True(t, re1.IsContentTypeSet())

	re1.HeaderAppend("X-Request-Type", "JSONP")
	re1.Header("X-Request-Type", "")

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
	assert.True(t, re1.IsContentTypeSet())

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

	assert.False(t, re1.IsStatusSet())

	// Just apply it again, no reason!
	re1.Header(ahttp.HeaderContentType, ahttp.ContentTypeXML.Raw())

	err := re1.Rdr.Render(buf)
	assert.FailOnError(t, err, "")
	assert.Equal(t, `<Sample><Name>John</Name><Age>28</Age><Address>this is my street</Address></Sample>`,
		buf.String())
}

func TestAttachmentFileReply(t *testing.T) {
	f1, _ := os.Open(getFilepath("file1.txt"))
	defer ess.CloseQuietly(f1)

	buf, re1 := getBufferAndReply()
	re1.File("sample.txt", f1)
	assert.False(t, re1.IsStatusSet())

	err := re1.Rdr.Render(buf)
	assert.FailOnError(t, err, "")
	assert.Equal(t, `
Each incoming request passes through a pre-defined list of steps
`, buf.String())

	buf.Reset()

	f2, _ := os.Open(getFilepath("file1.txt"))
	defer ess.CloseQuietly(f2)

	re2 := &Reply{Hdr: http.Header{}}
	re2.FileInline("sample.txt", f2)

	err = re2.Rdr.Render(buf)
	assert.FailOnError(t, err, "")
	assert.Equal(t, `
Each incoming request passes through a pre-defined list of steps
`, buf.String())

}

func TestHTMLReply(t *testing.T) {
	tmplStr := `
	{{ define "title" }}<title>This is test title</title>{{ end }}
	{{ define "body" }}<p>This is test body</p>{{ end }}
	`

	buf, re1 := getBufferAndReply()

	tmpl := template.Must(template.New("test").Parse(tmplStr))
	assert.NotNil(t, tmpl)

	testdataPath := getTestdataPath()
	masterTmpl := filepath.Join(testdataPath, "views", "master.html")
	_, err := tmpl.ParseFiles(masterTmpl)
	assert.Nil(t, err)

	re1.HTMLl("master", nil)
	htmlRdr := re1.Rdr.(*render.HTML)
	htmlRdr.Template = tmpl

	err = re1.Rdr.Render(buf)
	assert.Nil(t, err)
	assert.True(t, strings.Contains(buf.String(), "<title>This is test title</title>"))
	assert.True(t, strings.Contains(buf.String(), "<p>This is test body</p>"))

	// Not template/layout name
	buf.Reset()
	htmlRdr.Layout = ""
	err = re1.Rdr.Render(buf)
	assert.Nil(t, err)
	assert.True(t, ess.IsStrEmpty(buf.String()))

	// Template is Nil
	buf.Reset()

	re1.HTML(nil)
	err = re1.Rdr.Render(buf)
	assert.NotNil(t, err)
	assert.Equal(t, "template is nil", err.Error())
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
	return filepath.Join(getTestdataPath(), name)
}

func getTestdataPath() string {
	wd, _ := os.Getwd()
	return filepath.Join(wd, "testdata")
}
