// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"strings"
	"testing"

	"aahframework.org/test.v0/assert"
)

type project struct {
	Name        string `json:"name" xml:"name"`
	Description string `json:"description" xml:"description"`
	ProjectType string `json:"type" xml:"type"`
}

var _ RequestParser = (*JSONRequestParser)(nil)
var _ RequestParser = (*XMLRequestParser)(nil)

func TestJSONRequestParseer(t *testing.T) {
	p := &project{}
	parser := &JSONRequestParser{}
	r := strings.NewReader(`{"name":"aah", "description":"A Go web framework", "type":"web framework"}`)

	if err := parser.Parse(r, p); err != nil {
		t.Fatalf("Expected error to be nil after parsing.. \n Got %v", err)
	}

	assertParserParsesCorrectly(t, p)
}

func TestJSONRequestParseerInvalidReader(t *testing.T) {
	p := &project{}
	parser := &JSONRequestParser{}
	r := strings.NewReader("oops")

	if err := parser.Parse(r, p); err == nil {
		t.Fatalf("Expected error not to be nil after parsing.. \n Got a nil error %v", err)
	}
}

func TestXMLRequestParser(t *testing.T) {
	p := &project{}
	parser := &XMLRequestParser{}
	r := strings.NewReader(`
	<project><name>aah</name><description>A Go web framework</description><type>web framework</type></project>
	`)

	if err := parser.Parse(r, p); err != nil {
		t.Fatalf("Expected error to be nil after parsing.. \n Got %v", err)
	}

	assertParserParsesCorrectly(t, p)
}

func TestXMLRequestParserInvalidReader(t *testing.T) {
	p := &project{}
	parser := &XMLRequestParser{}
	r := strings.NewReader(`
	<projec></project>
	`)

	if err := parser.Parse(r, p); err == nil {
		t.Fatalf("Expected error not to be nil after parsing.. \n Got a nil error %v", err)
	}
}

func assertParserParsesCorrectly(t *testing.T, p *project) {
	assert.Equal(t, p.Name, "aah")
	assert.Equal(t, p.Description, "A Go web framework")
	assert.Equal(t, p.ProjectType, "web framework")
}
