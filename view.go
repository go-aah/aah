// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/view source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

// Package view is implementation of aah framework view engine using Go
// Template engine. It supports multi-layouts, no-layout, partial inheritance
// and error pages.
package view

import (
	"errors"
	"fmt"
	"html/template"
	"path/filepath"
	"reflect"
	"strings"

	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
)

var (
	// TemplateFuncMap aah framework Go template function map.
	TemplateFuncMap = make(template.FuncMap)

	// DefaultDelimiter template default delimiter
	DefaultDelimiter = "{{.}}"

	viewEngines = make(map[string]Enginer)
)

// view error messages
var (
	ErrTemplateEngineIsNil = errors.New("view: engine value is nil")
	ErrTemplateNotFound    = errors.New("view: template not found")
	ErrTemplateKeyExists   = errors.New("view: template key exists")
)

// Enginer interface defines a methods for pluggable view engine.
type Enginer interface {
	Init(appCfg *config.Config, baseDir string) error
	Get(layout, path, tmplName string) (*template.Template, error)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package methods
//___________________________________

// AddTemplateFunc method adds given Go template funcs into function map.
func AddTemplateFunc(funcMap template.FuncMap) {
	for fname, funcImpl := range funcMap {
		if _, found := TemplateFuncMap[fname]; !found {
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
		ty := reflect.TypeOf(engine)
		return reflect.New(ty.Elem()).Interface().(Enginer), found
	}
	return nil, false
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// type Templates, methods
//___________________________________

// Templates hold template reference of lowercase key and case sensitive key
// with reference to compliled template.
type Templates struct {
	set map[string]*template.Template
}

// Get method return the template for given key.
func (t *Templates) Lookup(key string) *template.Template {
	return t.set[key]
}

// Add method adds the given template for the key.
func (t *Templates) Add(key string, tmpl *template.Template) error {
	if t.IsExists(key) {
		return ErrTemplateKeyExists
	}

	if t.set == nil {
		t.set = make(map[string]*template.Template)
	}

	t.set[key] = tmpl
	return nil
}

// IsExists method returns true if template key exists otherwise false.
func (t *Templates) IsExists(key string) bool {
	if _, found := t.set[key]; found {
		return found
	}
	return false
}

// Keys method returns all the template keys.
func (t *Templates) Keys() []string {
	var keys []string
	for k := range t.set {
		keys = append(keys, k)
	}
	return keys
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// type EngineBase, methods
//___________________________________

// EngineBase struct is to create common and repurpose the implementation.
// Could used for custom view implementation.
type EngineBase struct {
	Name            string
	AppConfig       *config.Config
	BaseDir         string
	Templates       map[string]*Templates
	FileExt         string
	CaseSensitive   bool
	IsLayoutEnabled bool
	LeftDelim       string
	RightDelim      string
	AntiCSRFField   *AntiCSRFField
}

// Init method is to initialize the base fields values.
func (eb *EngineBase) Init(appCfg *config.Config, baseDir, defaultEngineName, defaultFileExt string) error {
	if appCfg == nil {
		return fmt.Errorf("view: app config is nil")
	}

	eb.Name = appCfg.StringDefault("view.engine", defaultEngineName)

	// check base directory
	if !ess.IsFileExists(baseDir) {
		return fmt.Errorf("%sviewengine: views base dir is not exists: %s", eb.Name, baseDir)
	}

	eb.Templates = make(map[string]*Templates)
	eb.AppConfig = appCfg
	eb.BaseDir = baseDir
	eb.FileExt = appCfg.StringDefault("view.ext", defaultFileExt)
	eb.CaseSensitive = appCfg.BoolDefault("view.case_sensitive", false)
	eb.IsLayoutEnabled = appCfg.BoolDefault("view.default_layout", true)

	delimiter := strings.Split(appCfg.StringDefault("view.delimiters", DefaultDelimiter), ".")
	if len(delimiter) != 2 || ess.IsStrEmpty(delimiter[0]) || ess.IsStrEmpty(delimiter[1]) {
		return fmt.Errorf("%sviewengine: config 'view.delimiters' value is invalid", eb.Name)
	}
	eb.LeftDelim, eb.RightDelim = delimiter[0], delimiter[1]

	// Anti CSRF
	eb.AntiCSRFField = NewAntiCSRFField("go", eb.LeftDelim, eb.RightDelim)
	return nil
}

// Get method returns the template based given name if found, otherwise nil.
func (eb *EngineBase) Get(layout, path, tmplName string) (*template.Template, error) {
	if ess.IsStrEmpty(layout) {
		layout = noLayout
	}

	if tmpls, found := eb.Templates[layout]; found {
		key := filepath.Join(path, tmplName)
		if layout == noLayout {
			key = noLayout + "-" + key
		}

		if !eb.CaseSensitive {
			key = strings.ToLower(key)
		}

		if t := tmpls.Lookup(key); t != nil {
			return t, nil
		}
	}

	return nil, ErrTemplateNotFound
}

// AddTemplate method adds the given template for layout and key.
func (eb *EngineBase) AddTemplate(layout, key string, tmpl *template.Template) error {
	if eb.Templates[layout] == nil {
		eb.Templates[layout] = &Templates{}
	}
	return eb.Templates[layout].Add(key, tmpl)
}

// ParseErrors method to parse and log the template error messages.
func (eb *EngineBase) ParseErrors(errs []error) error {
	if len(errs) > 0 {
		var msg []string
		for _, e := range errs {
			msg = append(msg, e.Error())
		}
		log.Errorf("View templates parsing error(s):\n    %s", strings.Join(msg, "\n    "))
		return errors.New(eb.Name + "viewengine: error processing templates, please check the log")
	}
	return nil
}

// LayoutFiles method returns the all layout files from `<view-base-dir>/layouts`.
// If layout directory doesn't exists it returns error.
func (eb *EngineBase) LayoutFiles() ([]string, error) {
	baseDir := filepath.Join(eb.BaseDir, "layouts")
	if !ess.IsFileExists(baseDir) {
		return nil, fmt.Errorf("%sviewengine: layouts base dir is not exists: %s", eb.Name, baseDir)
	}

	return filepath.Glob(filepath.Join(baseDir, "*"+eb.FileExt))
}

// DirsPath method returns all sub directories from `<view-base-dir>/<sub-dir-name>`.
// if it not exists returns error.
func (eb *EngineBase) DirsPath(subDir string) ([]string, error) {
	baseDir := filepath.Join(eb.BaseDir, subDir)
	if !ess.IsFileExists(baseDir) {
		return nil, fmt.Errorf("%sviewengine: %s base dir is not exists: %s", eb.Name, subDir, baseDir)
	}

	return ess.DirsPath(baseDir, true)
}

// FilesPath method returns all file path from `<view-base-dir>/<sub-dir-name>`.
// if it not exists returns error.
func (eb *EngineBase) FilesPath(subDir string) ([]string, error) {
	baseDir := filepath.Join(eb.BaseDir, subDir)
	if !ess.IsFileExists(baseDir) {
		return nil, fmt.Errorf("%sviewengine: %s base dir is not exists: %s", eb.Name, subDir, baseDir)
	}

	return ess.FilesPath(baseDir, true)
}

// NewTemplate method return new instance on `template.Template` initialized with
// key, template funcs and delimiters.
func (eb *EngineBase) NewTemplate(key string) *template.Template {
	return template.New(key).Funcs(TemplateFuncMap).Delims(eb.LeftDelim, eb.RightDelim)
}
