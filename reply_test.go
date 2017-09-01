// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/test.v0/assert"
)

func TestReplyStatusCodes(t *testing.T) {
	re := NewReply()

	assert.Equal(t, http.StatusOK, re.Code)

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

	re.Found()
	assert.Equal(t, http.StatusFound, re.Code)

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

func TestReplyText(t *testing.T) {
	buf, re1 := acquireBuffer(), acquireReply()

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

func TestReplyJSON(t *testing.T) {
	buf, re1 := acquireBuffer(), acquireReply()
	appConfig = getReplyRenderCfg()

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

	re1.HeaderAppend(ahttp.HeaderContentType, "application/json")
	assert.True(t, re1.IsContentTypeSet())

	err := re1.Rdr.Render(buf)
	assert.FailOnError(t, err, "")
	assert.Equal(t, `{
    "Name": "John",
    "Age": 28,
    "Address": "this is my street"
}`, buf.String())

	buf.Reset()

	appConfig.SetBool("render.pretty", false)

	err = re1.Rdr.Render(buf)
	assert.FailOnError(t, err, "")
	assert.Equal(t, `{"Name":"John","Age":28,"Address":"this is my street"}`,
		buf.String())
}

func TestReplyJSONP(t *testing.T) {
	buf, re1 := acquireBuffer(), acquireReply()
	re1.body = buf
	appConfig = getReplyRenderCfg()

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

	err := re1.Rdr.Render(re1.body)
	assert.FailOnError(t, err, "")
	assert.Equal(t, `mycallback({
    "Name": "John",
    "Age": 28,
    "Address": "this is my street"
});`, re1.body.String())
	assert.NotNil(t, re1.Body())

	re1.body.Reset()

	appConfig.SetBool("render.pretty", false)

	err = re1.Rdr.Render(re1.body)
	assert.FailOnError(t, err, "")
	assert.Equal(t, `mycallback({"Name":"John","Age":28,"Address":"this is my street"});`,
		re1.body.String())
}

func TestReplyXML(t *testing.T) {
	buf, re1 := acquireBuffer(), acquireReply()
	appConfig = getReplyRenderCfg()

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
	assert.Equal(t, `<?xml version="1.0" encoding="UTF-8"?>
<Sample>
    <Name>John</Name>
    <Age>28</Age>
    <Address>this is my street</Address>
</Sample>`, buf.String())

	buf.Reset()

	appConfig.SetBool("render.pretty", false)

	err = re1.Rdr.Render(buf)
	assert.FailOnError(t, err, "")
	assert.Equal(t, `<?xml version="1.0" encoding="UTF-8"?>
<Sample><Name>John</Name><Age>28</Age><Address>this is my street</Address></Sample>`,
		buf.String())

	buf.Reset()

	data2 := Data{
		"Name":    "John",
		"Age":     28,
		"Address": "this is my street",
	}

	re1.Rdr.(*XML).reset()
	re1.XML(data2)
	assert.True(t, re1.IsContentTypeSet())

	err = re1.Rdr.Render(buf)
	assert.FailOnError(t, err, "")
	assert.True(t, strings.HasPrefix(buf.String(), `<?xml version="1.0" encoding="UTF-8"?>`))
	re1.Rdr.(*XML).reset()
}

func TestReplyReadfrom(t *testing.T) {
	buf, re1 := acquireBuffer(), acquireReply()
	re1.ContentType(ahttp.ContentTypeOctetStream.Raw()).
		Binary([]byte(`<Sample><Name>John</Name><Age>28</Age><Address>this is my street</Address></Sample>`))

	assert.Equal(t, http.StatusOK, re1.Code)

	// Just apply it again, no reason!
	re1.Header(ahttp.HeaderContentType, ahttp.ContentTypeXML.Raw())

	err := re1.Rdr.Render(buf)
	assert.FailOnError(t, err, "")
	assert.Equal(t, `<Sample><Name>John</Name><Age>28</Age><Address>this is my street</Address></Sample>`,
		buf.String())
}

func TestReplyFileDownload(t *testing.T) {
	buf, re1 := acquireBuffer(), acquireReply()
	re1.FileDownload(getReplyFilepath("file1.txt"), "sample.txt")
	assert.Equal(t, http.StatusOK, re1.Code)

	err := re1.Rdr.Render(buf)
	assert.FailOnError(t, err, "")
	assert.Equal(t, `
Each incoming request passes through a pre-defined list of steps
`, buf.String())

	buf.Reset()

	re2 := &Reply{Hdr: http.Header{}}
	re2.FileInline(getReplyFilepath("file1.txt"), "sample.txt")

	err = re2.Rdr.Render(buf)
	assert.FailOnError(t, err, "")
	assert.Equal(t, `
Each incoming request passes through a pre-defined list of steps
`, buf.String())
}

func TestReplyHTML(t *testing.T) {
	tmplStr := `
	{{ define "title" }}<title>This is test title</title>{{ end }}
	{{ define "body" }}<p>This is test body</p>{{ end }}
	`

	buf, re1 := acquireBuffer(), acquireReply()

	tmpl := template.Must(template.New("test").Parse(tmplStr))
	assert.NotNil(t, tmpl)

	masterTmpl := getReplyFilepath(filepath.Join("views", "master.html"))
	_, err := tmpl.ParseFiles(masterTmpl)
	assert.Nil(t, err)

	re1.HTMLl("master", nil)
	htmlRdr := re1.Rdr.(*HTML)
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
	releaseReply(re1)

	// HTMLlf
	relf := acquireReply()
	relf.HTMLlf("docs.html", "Filename.html", nil)
	assert.Equal(t, "text/html; charset=utf-8", relf.ContType)

	htmllf := relf.Rdr.(*HTML)
	assert.Equal(t, "docs.html", htmllf.Layout)
	assert.Equal(t, "Filename.html", htmllf.Filename)
	releaseRender(htmllf)

	// HTMLf
	ref := acquireReply()
	ref.HTMLf("Filename1.html", nil)
	assert.Equal(t, "text/html; charset=utf-8", ref.ContType)

	htmlf := ref.Rdr.(*HTML)
	assert.True(t, ess.IsStrEmpty(htmlf.Layout))
	assert.Equal(t, "Filename1.html", htmlf.Filename)
	releaseRender(htmlf)
}

func TestReplyRedirect(t *testing.T) {
	redirect1 := acquireReply()
	redirect1.Redirect("/go-to-see.page")
	assert.Equal(t, http.StatusFound, redirect1.Code)
	assert.True(t, redirect1.redirect)
	assert.Equal(t, "/go-to-see.page", redirect1.path)
	releaseReply(redirect1)

	redirect2 := acquireReply()
	redirect2.RedirectSts("/go-to-see-gone-premanent.page", http.StatusMovedPermanently)
	assert.Equal(t, http.StatusMovedPermanently, redirect2.Code)
	assert.True(t, redirect2.redirect)
	assert.Equal(t, "/go-to-see-gone-premanent.page", redirect2.path)
	releaseReply(redirect2)
}

func TestReplyDone(t *testing.T) {
	re1 := acquireReply()

	assert.False(t, re1.done)
	re1.Done()
	assert.True(t, re1.done)
}

func TestReplyCookie(t *testing.T) {
	re1 := acquireReply()

	re1.Cookie(&http.Cookie{
		Name:     "aah-test-cookie",
		Value:    "This is reply cookie interface test value",
		Path:     "/",
		Domain:   "*.sample.com",
		HttpOnly: true,
	})

	assert.NotNil(t, re1.cookies)

	cookie := re1.cookies[0]
	assert.Equal(t, "aah-test-cookie", cookie.Name)
	releaseReply(re1)
}

// customRender implements the interface `aah.Render`.
type customRender struct {
	// ... your fields goes here
}

func (cr *customRender) Render(w io.Writer) error {
	fmt.Fprint(w, "This is custom render struct")
	return nil
}

func TestReplyCustomRender(t *testing.T) {
	re := acquireReply()
	buf := acquireBuffer()

	re.Render(&customRender{})
	err := re.Rdr.Render(buf)
	assert.Nil(t, err)
	assert.Equal(t, "This is custom render struct", buf.String())

	releaseBuffer(buf)
	releaseReply(re)

	// Render func
	re = acquireReply()
	buf = acquireBuffer()

	re.Render(RenderFunc(func(w io.Writer) error {
		fmt.Fprint(w, "This is custom render func")
		return nil
	}))
	err = re.Rdr.Render(buf)
	assert.Nil(t, err)
	assert.Equal(t, "This is custom render func", buf.String())

	releaseBuffer(buf)
	releaseReply(re)
}

func getReplyRenderCfg() *config.Config {
	cfg, _ := config.ParseString(`
  render {
    pretty = true
  }
    `)
	return cfg
}

func getReplyFilepath(name string) string {
	wd, _ := os.Getwd()
	return filepath.Join(wd, "testdata", "reply", name)
}
