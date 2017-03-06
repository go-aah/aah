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

	"aahframework.org/essentials.v0-unstable"
)

type (
	// Data type used for convenient data type of map[string]interface{}
	Data map[string]interface{}

	// Render interface
	Render interface {
		Render(io.Writer) error
	}

	// Text renders the response as plain text
	Text struct {
		Format string
		Values []interface{}
	}

	// JSON renders the response JSON content.
	JSON struct {
		IsJSONP  bool
		Callback string
		Data     interface{}
	}

	// XML renders the response XML content.
	XML struct {
		Data interface{}
	}

	// Bytes renders the bytes into response.
	Bytes struct {
		Data []byte
	}

	// File renders given ReadCloser into response and closes the file.
	File struct {
		Data io.ReadCloser
	}

	// HTML renders the given HTML into response with given model data.
	HTML struct {
		Template *template.Template
		Layout   string
		ViewArgs Data
	}
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Plain Text Render methods
//___________________________________

// Render method writes Text to HTTP response.
func (t *Text) Render(w io.Writer) error {
	if len(t.Values) > 0 {
		fmt.Fprintf(w, t.Format, t.Values...)
	} else {
		_, _ = io.WriteString(w, t.Format)
	}

	return nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// JSON Render methods
//___________________________________

// Render method writes JSON to HTTP response.
func (j *JSON) Render(w io.Writer) error {
	var (
		bytes []byte
		err   error
	)

	if appConfig.BoolDefault("render.pretty", false) {
		bytes, err = json.MarshalIndent(j.Data, "", "    ")
	} else {
		bytes, err = json.Marshal(j.Data)
	}

	if err != nil {
		return err
	}

	if j.IsJSONP {
		if _, err = w.Write([]byte(j.Callback + "(")); err != nil {
			return err
		}
	}

	if _, err = w.Write(bytes); err != nil {
		return err
	}

	if j.IsJSONP {
		if _, err = w.Write([]byte(");")); err != nil {
			return err
		}
	}

	return nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// XML Render methods
//___________________________________

// Render method writes XML to HTTP response.
func (x *XML) Render(w io.Writer) error {
	var (
		bytes []byte
		err   error
	)

	if appConfig.BoolDefault("render.pretty", false) {
		bytes, err = xml.MarshalIndent(x.Data, "", "    ")
	} else {
		bytes, err = xml.Marshal(x.Data)
	}

	if err != nil {
		return err
	}

	if _, err = w.Write(bytes); err != nil {
		return err
	}

	return nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Bytes Render methods
//___________________________________

// Render method writes XML to HTTP response.
func (b *Bytes) Render(w io.Writer) error {
	_, err := w.Write(b.Data)
	return err
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// File Render methods
//___________________________________

// Render method writes File to HTTP response.
func (f *File) Render(w io.Writer) error {
	defer ess.CloseQuietly(f.Data)
	_, err := io.Copy(w, f.Data)
	return err
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// HTML Render methods
//___________________________________

// Render method renders the HTML template into given writer.
func (h *HTML) Render(w io.Writer) error {
	if h.Template == nil {
		return errors.New("template is nil")
	}

	if ess.IsStrEmpty(h.Layout) {
		return h.Template.Execute(w, h.ViewArgs)
	}

	return h.Template.ExecuteTemplate(w, h.Layout, h.ViewArgs)
}
