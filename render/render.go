// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package render

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"

	"aahframework.org/config"
)

var appConfig *config.Config

type (
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

	// TODO HTML template and rendering
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Global methods
//___________________________________

// Init assigns the application config.
func Init(cfg *config.Config) {
	appConfig = cfg
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Plain Text Render methods
//___________________________________

// Render methods writes Text to HTTP response.
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

// Render methods writes JSON to HTTP response.
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

// Render methods writes XML to HTTP response.
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

// Render methods writes XML to HTTP response.
func (b *Bytes) Render(w io.Writer) error {
	_, err := w.Write(b.Data)
	return err
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// File Render methods
//___________________________________

// Render methods writes File to HTTP response.
func (f *File) Render(w io.Writer) error {
	defer func() {
		_ = f.Data.Close()
	}()

	_, err := io.Copy(w, f.Data)
	return err
}
