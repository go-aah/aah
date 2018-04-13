// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"html/template"
	"io"
	"os"
	"strings"

	"aahframework.org/essentials.v0"
)

const (
	defaultSecureJSONPrefix = ")]}',\n"
)

var (
	// JSONMarshal is used to register external JSON library for Marshalling.
	JSONMarshal func(v interface{}) ([]byte, error)

	// JSONMarshalIndent is used to register external JSON library for Marshal indent.
	JSONMarshalIndent func(v interface{}, prefix, indent string) ([]byte, error)

	xmlHeaderBytes = []byte(xml.Header)
)

type (

	// Render interface to various rendering classifcation for HTTP responses.
	Render interface {
		Render(io.Writer) error
	}

	// RenderFunc type is an adapter to allow the use of regular function as
	// custom Render.
	RenderFunc func(w io.Writer) error
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// RenderFunc methods
//______________________________________________________________________________

// Render method is implementation of Render interface in the adapter type.
func (rf RenderFunc) Render(w io.Writer) error {
	return rf(w)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Plain Text Render
//______________________________________________________________________________

// textRender renders the response as plain text
type textRender struct {
	Format string
	Values []interface{}
}

// textRender method writes given text into HTTP response.
func (t textRender) Render(w io.Writer) (err error) {
	if len(t.Values) > 0 {
		_, err = fmt.Fprintf(w, t.Format, t.Values...)
	} else {
		_, err = io.WriteString(w, t.Format)
	}
	return
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// JSON Render
//______________________________________________________________________________

// jsonRender renders the response JSON content.
type jsonRender struct {
	Data interface{}
}

// Render method writes JSON into HTTP response.
func (j jsonRender) Render(w io.Writer) error {
	jsonBytes, err := JSONMarshal(j.Data)
	if err != nil {
		return err
	}

	_, err = w.Write(jsonBytes)
	return err
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// JSONP Render
//______________________________________________________________________________

// jsonpRender renders the JSONP response.
type jsonpRender struct {
	Callback string
	Data     interface{}
}

// Render method writes JSONP into HTTP response.
func (j jsonpRender) Render(w io.Writer) error {
	jsonBytes, err := JSONMarshal(j.Data)
	if err != nil {
		return err
	}

	if ess.IsStrEmpty(j.Callback) {
		_, err = w.Write(jsonBytes)
	} else {
		_, err = fmt.Fprintf(w, "%s(%s);", j.Callback, jsonBytes)
	}

	return err
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// SecureJSON Render
//______________________________________________________________________________

type secureJSONRender struct {
	Prefix string
	Data   interface{}
}

func (s secureJSONRender) Render(w io.Writer) error {
	jsonBytes, err := JSONMarshal(s.Data)
	if err != nil {
		return err
	}

	if _, err = w.Write([]byte(s.Prefix)); err != nil {
		return err
	}

	_, err = w.Write(jsonBytes)
	return err
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// XML Render
//______________________________________________________________________________

// xmlRender renders the response XML content.
type xmlRender struct {
	Data interface{}
}

// Render method writes XML into HTTP response.
func (x xmlRender) Render(w io.Writer) error {
	xmlBytes, err := xml.Marshal(x.Data)
	if err != nil {
		return err
	}

	if _, err = w.Write(xmlHeaderBytes); err != nil {
		return err
	}

	if _, err = w.Write(xmlBytes); err != nil {
		return err
	}

	return nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Data
//______________________________________________________________________________

// Data type used for convenient data type of map[string]interface{}
type Data map[string]interface{}

// MarshalXML method is to marshal `aah.Data` into XML.
func (d Data) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	tokens := []xml.Token{start}
	for k, v := range d {
		token := xml.StartElement{Name: xml.Name{Local: strings.Title(k)}}
		tokens = append(tokens, token,
			xml.CharData(fmt.Sprintf("%v", v)),
			xml.EndElement{Name: token.Name})
	}

	tokens = append(tokens, xml.EndElement{Name: start.Name})

	var err error
	for _, t := range tokens {
		if err = e.EncodeToken(t); err != nil {
			return err
		}
	}

	// flush to ensure tokens are written
	return e.Flush()
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Reader Render
//______________________________________________________________________________

// Binary renders given path or io.Reader into response and closes the file.
type binaryRender struct {
	Path   string
	Reader io.Reader
}

// Render method writes File into HTTP response.
func (f binaryRender) Render(w io.Writer) error {
	if f.Reader != nil {
		defer ess.CloseQuietly(f.Reader)
		_, err := io.Copy(w, f.Reader)
		return err
	}

	file, err := os.Open(f.Path)
	if err != nil {
		return err
	}
	defer ess.CloseQuietly(file)

	fi, err := file.Stat()
	if err != nil {
		return err
	}

	if fi.IsDir() {
		return fmt.Errorf("'%s' is a directory", f.Path)
	}

	_, err = io.Copy(w, file)
	return err
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// HTML Render
//______________________________________________________________________________

// htmlRender renders the given HTML template into response with given model data.
type htmlRender struct {
	Template *template.Template
	Layout   string
	Filename string
	ViewArgs Data
}

// Render method renders the HTML template into HTTP response.
func (h htmlRender) Render(w io.Writer) error {
	if h.Template == nil {
		return errors.New("template is nil")
	}

	if h.Layout == "" {
		return h.Template.Execute(w, h.ViewArgs)
	}

	return h.Template.ExecuteTemplate(w, h.Layout, h.ViewArgs)
}

func init() {
	// Registering default standard JSON library
	JSONMarshal = json.Marshal
	JSONMarshalIndent = json.MarshalIndent
}
