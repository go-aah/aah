package aah

import (
	"encoding/json"
	"encoding/xml"
	"io"
)

type (
	//RequestParser is an interface for writing the body of an HTTP Request into a struct or given datatype)
	RequestParser interface {
		Parse(r io.Reader, v interface{}) error
	}
)

//JSONRequestParser is a RequestParser for JSON requests
type JSONRequestParser struct{}

//Parse writes the body of the given reader into a specified data type
func (j *JSONRequestParser) Parse(r io.Reader, v interface{}) error {
	return json.NewDecoder(r).Decode(v)
}

// XMLRequestParser is a RequestParser for XML requests
type XMLRequestParser struct{}

// Parse writes the body of a given reader into a specified data type
func (x *XMLRequestParser) Parse(r io.Reader, v interface{}) error {
	return xml.NewDecoder(r).Decode(v)
}
