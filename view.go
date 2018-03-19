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
	"reflect"

	"aahframework.org/config.v0"
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
