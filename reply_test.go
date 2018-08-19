// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"aahframework.org/essentials"
	"github.com/stretchr/testify/assert"
)

func TestReplyStatusCodes(t *testing.T) {
	re := newReply(newContext(nil, nil))

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
func TestReplyHTML(t *testing.T) {
	tmplStr := `
	{{ define "title" }}<title>This is test title</title>{{ end }}
	{{ define "body" }}<p>This is test body</p>{{ end }}
	`

	buf, re1 := acquireBuffer(), newReply(nil)

	tmpl := template.Must(template.New("test").Parse(tmplStr))
	assert.NotNil(t, tmpl)

	masterTmpl := filepath.Join(testdataBaseDir(), "reply", "views", "master.html")
	_, err := tmpl.ParseFiles(masterTmpl)
	assert.Nil(t, err)

	re1.HTMLl("master", nil)
	htmlRdr := re1.Rdr.(*htmlRender)
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

	// HTMLlf
	relf := newReply(nil)
	relf.HTMLlf("docs.html", "Filename.html", nil)
	assert.Equal(t, "text/html; charset=utf-8", relf.ContType)

	htmllf := relf.Rdr.(*htmlRender)
	assert.Equal(t, "docs.html", htmllf.Layout)
	assert.Equal(t, "Filename.html", htmllf.Filename)

	// HTMLf
	ref := newReply(nil)
	ref.HTMLf("Filename1.html", nil)
	assert.Equal(t, "text/html; charset=utf-8", ref.ContType)

	htmlf := ref.Rdr.(*htmlRender)
	assert.True(t, ess.IsStrEmpty(htmlf.Layout))
	assert.Equal(t, "Filename1.html", htmlf.Filename)
}
func TestReplyDone(t *testing.T) {
	re1 := newReply(nil)

	assert.False(t, re1.done)
	re1.Done()
	assert.True(t, re1.done)
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
	re := newReply(nil)
	buf := acquireBuffer()

	re.Render(&customRender{})
	err := re.Rdr.Render(buf)
	assert.Nil(t, err)
	assert.Equal(t, "This is custom render struct", buf.String())

	releaseBuffer(buf)

	// Render func
	re = newReply(nil)
	buf = acquireBuffer()

	re.Render(RenderFunc(func(w io.Writer) error {
		fmt.Fprint(w, "This is custom render func")
		return nil
	}))
	err = re.Rdr.Render(buf)
	assert.Nil(t, err)
	assert.Equal(t, "This is custom render func", buf.String())

	releaseBuffer(buf)
}
