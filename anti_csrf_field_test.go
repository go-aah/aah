// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/view source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package view

import (
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"aahframework.org/test.v0/assert"
)

func TestAntiCSRFFieldNoFormTag(t *testing.T) {
	acsrf := NewAntiCSRFField("go", "{{", "}}")
	fpath := filepath.Join(testdataBaseDir(), "anti-csrf-field", "testhtml-noform.html")

	files := acsrf.InsertOnFiles(fpath)
	bytes, err := ioutil.ReadFile(files[0])
	assert.Nil(t, err)
	assert.False(t, strings.Contains(string(bytes), "{{ anti_csrf_token . }}"))
}

func TestAntiCSRFFieldFormTag(t *testing.T) {
	acsrf := NewAntiCSRFField("go", "%%", "%%")
	fpath := filepath.Join(testdataBaseDir(), "anti-csrf-field", "testhtml-form.html")

	files := acsrf.InsertOnFiles(fpath)
	bytes, err := ioutil.ReadFile(files[0])
	assert.Nil(t, err)
	assert.True(t, strings.Contains(string(bytes), "%% anticsrftoken . %%"))
}

func TestAntiCSRFFieldFormTagDelim(t *testing.T) {
	acsrf := NewAntiCSRFField("go", "[[", "]]")
	fpath := filepath.Join(testdataBaseDir(), "anti-csrf-field", "not-exists.html")

	files := acsrf.InsertOnFiles(fpath)
	assert.NotNil(t, files)
	assert.Equal(t, fpath, files[0])
}
