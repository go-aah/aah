// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/view source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package view

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"path/filepath"
	"strings"

	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
	"aahframework.org/pool.v0"
)

// Version no. of aah framework view library
const Version = "0.3"

var (
	// TemplateFuncMap aah framework Go template function map.
	TemplateFuncMap = make(template.FuncMap)

	viewEngines    = make(map[string]Enginer)
	commonTemplate = &CommonTemplate{}
)

// view error messages
var (
	ErrTemplateEngineIsNil = errors.New("view: engine value is nil")
	ErrTemplateNotFound    = errors.New("view: template not found")
)

type (
	// Enginer interface defines a methods for pluggable view engine.
	Enginer interface {
		Init(appCfg *config.Config, baseDir string) error
		Get(layout, path, tmplName string) (*template.Template, error)
	}

	// CommonTemplate holds the implementation of common templates which can
	// be imported via template function `import "name.ext" .`.
	CommonTemplate struct {
		templates *template.Template
		bufPool   *pool.Pool
	}

	// Templates hold template reference of lowercase key and case sensitive key
	// with reference to compliled template.
	Templates struct {
		TemplateLower map[string]*template.Template
		Template      map[string]*template.Template
	}
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Global methods
//___________________________________

// AddTemplateFunc method adds given Go template funcs into function map.
func AddTemplateFunc(funcMap template.FuncMap) {
	for fname, funcImpl := range funcMap {
		if _, found := TemplateFuncMap[fname]; found {
			log.Warnf("Template func name '%s' already exists, skip it.", fname)
		} else {
			TemplateFuncMap[fname] = funcImpl
		}
	}
}

// AddEngine method adds the given name and engine to view store.
func AddEngine(name string, engine Enginer) error {
	if engine == nil {
		return ErrTemplateEngineIsNil
	}

	if _, found := viewEngines[name]; found {
		return fmt.Errorf("view: engine name '%v' is already added, skip it", name)
	}

	viewEngines[name] = engine
	return nil
}

// GetEngine method returns the view engine from store by name otherwise nil.
func GetEngine(name string) (Enginer, bool) {
	if engine, found := viewEngines[name]; found {
		return engine, found
	}
	return nil, false
}

// TemplateKey returns the unique key for given path.
func TemplateKey(path string) string {
	path = path[strings.Index(path, "pages"):]
	path = strings.Replace(path, "/", "_", -1)
	path = strings.Replace(path, "\\", "_", -1)
	return path
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// CommonTemplate methods
//___________________________________

// Init method initializes the common templates which can be imported via
// template function `import "name.ext" .`.
func (c *CommonTemplate) Init(cfg *config.Config, baseDir string) error {
	commonBaseDir := filepath.Join(baseDir, "common")
	if !ess.IsFileExists(commonBaseDir) {
		return nil
	}

	c.bufPool = pool.NewPool(
		cfg.IntDefault("pooling.buffer", 0),
		func() interface{} {
			return &bytes.Buffer{}
		},
	)

	viewFileExt := cfg.StringDefault("view.ext", ".html")
	commons, err := filepath.Glob(filepath.Join(commonBaseDir, "*"+viewFileExt))
	if err != nil {
		return err
	}

	c.templates, err = template.New("common").Funcs(TemplateFuncMap).ParseFiles(commons...)
	return err
}

// Execute method does lookup of common template and renders it. It returns
// template output otherwise empty string with error.
func (c *CommonTemplate) Execute(name string, viewArgs map[string]interface{}) (string, error) {
	tmpl := c.templates.Lookup(name)
	if tmpl == nil {
		return "", fmt.Errorf("commontemplate: template not found: %s", name)
	}

	buf := c.getBuffer()
	defer c.putBuffer(buf)

	if err := tmpl.Execute(buf, viewArgs); err != nil {
		return "", err
	}

	return buf.String(), nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// CommonTemplate unexported methods
//___________________________________

func (c *CommonTemplate) getBuffer() *bytes.Buffer {
	return c.bufPool.Get().(*bytes.Buffer)
}

func (c *CommonTemplate) putBuffer(buf *bytes.Buffer) {
	buf.Reset()
	c.bufPool.Put(buf)
}

func init() {
	AddTemplateFunc(template.FuncMap{
		"safeHTML": tmplSafeHTML,
		"import":   tmplImport,
	})
}
