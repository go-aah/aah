package aah

import (
	"strings"
	"testing"

	"aahframework.org/test.v0/assert"
)

type project struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ProjectType string `json:"type"`
}

var _ RequestParser = (*JSONRequestParser)(nil)

func TestJSONRequestParseer(t *testing.T) {
	p := &project{}
	parser := &JSONRequestParser{}
	r := strings.NewReader(`{"name":"aah", "description":"A Go web framework", "type":"web framework"}`)

	if err := parser.Parse(r, p); err != nil {
		t.Fatalf("Expected error to be nil after parsing.. \n Got %v", err)
	}

	assert.Equal(t, p.Name, "aah")
	assert.Equal(t, p.Description, "A Go web framework")
	assert.Equal(t, p.ProjectType, "web framework")
}

func TestJSONRequestParseerInvalidReader(t *testing.T) {
	p := &project{}
	parser := &JSONRequestParser{}
	r := strings.NewReader("oops")

	if err := parser.Parse(r, p); err == nil {
		t.Fatalf("Expected error not to be nil after parsing.. \n Got a nil error %v", err)
	}
}
