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
	"path/filepath"
	"strings"
	"sync"

	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0-unstable"
)

var (
	xmlHeaderBytes = []byte(xml.Header)
	rdrHTMLPool    = &sync.Pool{New: func() interface{} { return &HTML{} }}
	rdrJSONPool    = &sync.Pool{New: func() interface{} { return &JSON{} }}
	rdrXMLPool     = &sync.Pool{New: func() interface{} { return &XML{} }}
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

	// Binary renders given path or io.Reader into response and closes the file.
	Binary struct {
		Path   string
		Reader io.Reader
	}

	// HTML renders the given HTML into response with given model data.
	HTML struct {
		Template *template.Template
		Layout   string
		Filename string
		ViewArgs Data
	}
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Plain Text Render methods
//___________________________________

// Render method writes Text into HTTP response.
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

// Render method writes JSON into HTTP response.
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

func (j *JSON) reset() {
	j.Callback = ""
	j.IsJSONP = false
	j.Data = nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// XML Render methods
//___________________________________

// Render method writes XML into HTTP response.
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

	if _, err = w.Write(xmlHeaderBytes); err != nil {
		return err
	}

	if _, err = w.Write(bytes); err != nil {
		return err
	}

	return nil
}

func (x *XML) reset() {
	x.Data = nil
}

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

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// File and Reader Render methods
//___________________________________

// Render method writes File into HTTP response.
func (f *Binary) Render(w io.Writer) error {
	if f.Reader != nil {
		defer ess.CloseQuietly(f.Reader)
		_, err := io.Copy(w, f.Reader)
		return err
	}

	if !filepath.IsAbs(f.Path) {
		f.Path = filepath.Join(AppBaseDir(), "static", f.Path)
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

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// HTML Render methods
//___________________________________

// Render method renders the HTML template into HTTP response.
func (h *HTML) Render(w io.Writer) error {
	if h.Template == nil {
		return errors.New("template is nil")
	}

	if ess.IsStrEmpty(h.Layout) {
		return h.Template.Execute(w, h.ViewArgs)
	}

	return h.Template.ExecuteTemplate(w, h.Layout, h.ViewArgs)
}

func (h *HTML) reset() {
	h.Template = nil
	h.Filename = ""
	h.Layout = ""
	h.ViewArgs = make(Data)
}

// doRender method renders and detects the errors earlier. Writes the
// error info if any.
func (e *engine) doRender(ctx *Context) {
	if ctx.Reply().Rdr != nil {
		reply := ctx.Reply()
		reply.body = acquireBuffer()
		if jsonp, ok := reply.Rdr.(*JSON); ok && jsonp.IsJSONP {
			if ess.IsStrEmpty(jsonp.Callback) {
				jsonp.Callback = ctx.Req.QueryValue("callback")
			}
		}

		if err := reply.Rdr.Render(reply.body); err != nil {
			log.Error("Render response body error: ", err)
			// TODO handle response based on content type
			reply.InternalServerError()
			reply.body.Reset()
			reply.body.WriteString("500 Internal Server Error\n")
		}
	}
}

func acquireHTML() *HTML {
	return rdrHTMLPool.Get().(*HTML)
}

func acquireJSON() *JSON {
	return rdrJSONPool.Get().(*JSON)
}

func acquireXML() *XML {
	return rdrXMLPool.Get().(*XML)
}

func releaseRender(r Render) {
	if r != nil {
		if t, ok := r.(*JSON); ok {
			t.reset()
			rdrJSONPool.Put(t)
		} else if t, ok := r.(*HTML); ok {
			t.reset()
			rdrHTMLPool.Put(t)
		} else if t, ok := r.(*XML); ok {
			t.reset()
			rdrXMLPool.Put(t)
		}
	}
}
