// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org/view source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

// Package view is implementation of aah framework view engine using Go
// Template engine. It supports multi-layouts, no-layout, partial inheritance
// and error pages.
package view

import (
	"errors"
	"fmt"
	"html/template"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
	"aahframework.org/vfs.v0"
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
	Init(fs *vfs.VFS, appCfg *config.Config, baseDir string) error
	Get(layout, path, tmplName string) (*template.Template, error)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package methods
//______________________________________________________________________________

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
	engine, found := viewEngines[name]
	return engine, found
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// type Templates, methods
//______________________________________________________________________________

// Templates hold template reference of lowercase key and case sensitive key
// with reference to compliled template.
type Templates struct {
	set map[string]*template.Template
}

// Lookup method return the template for given key.
func (t *Templates) Lookup(key string) *template.Template {
	return t.set[filepath.ToSlash(key)]
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

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// type EngineBase, its methods
//______________________________________________________________________________

// EngineBase struct is to create common and repurpose the implementation.
// Could be used for custom view engine implementation.
type EngineBase struct {
	CaseSensitive   bool
	IsLayoutEnabled bool
	hotReload       bool
	Name            string
	BaseDir         string
	FileExt         string
	LeftDelim       string
	RightDelim      string
	AppConfig       *config.Config
	Templates       map[string]*Templates
	VFS             *vfs.VFS
	loginFormRegex  *regexp.Regexp
}

// Init method is to initialize the base fields values.
func (eb *EngineBase) Init(fs *vfs.VFS, appCfg *config.Config, baseDir, defaultEngineName, defaultFileExt string) error {
	if appCfg == nil {
		return fmt.Errorf("view: app config is nil")
	}

	eb.VFS = fs
	eb.Name = appCfg.StringDefault("view.engine", defaultEngineName)

	// check base directory
	if !eb.VFS.IsExists(baseDir) {
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

	eb.loginFormRegex = regexp.MustCompile(`(<form(.*)_login_submit__aah\"(.*)(?s)>)`)

	return nil
}

// Open method reads template from VFS if not found resolve from physical
// file system. Also does auto field insertion such as
// Anti-CSRF(anti_csrf_token) and requested page URL (_rt).
func (eb *EngineBase) Open(filename string) (string, error) {
	b, err := vfs.ReadFile(eb.VFS, filename)
	if err != nil {
		return "", err
	}
	return eb.AutoFieldInsertion(filename, string(b)), nil
}

// AutoFieldInsertion method processes the aah view's to auto insert the field.
func (eb *EngineBase) AutoFieldInsertion(name, v string) string {
	// process auto field insertion, if form tag exists
	// anti_csrf_token field
	if strings.Contains(v, "</form>") {
		log.Tracef("Adding field 'anti_csrf_token' into all forms: %s", name)
		fieldName := eb.AppConfig.StringDefault("security.anti_csrf.form_field_name", "anti_csrf_token")
		v = strings.Replace(v, "</form>", fmt.Sprintf(`<input type="hidden" name="%s" value="%s anticsrftoken . %s">
	     	</form>`, fieldName, eb.LeftDelim, eb.RightDelim), -1)
	}

	// _rt field
	if matches := eb.loginFormRegex.FindAllStringIndex(v, -1); len(matches) > 0 {
		log.Tracef("Adding field '_rt' into login form: %s", name)
		for _, m := range matches {
			ts := v[m[0]:m[1]]
			v = strings.Replace(v, ts, fmt.Sprintf(`%s
			<input type="hidden" value="{{ qparam . "_rt" }}" name="_rt">`, ts), 1)
		}
	}

	return v
}

// ParseFile method parses given single file.
func (eb *EngineBase) ParseFile(filename string) (*template.Template, error) {
	if !strings.HasPrefix(filename, eb.BaseDir) {
		filename = path.Join(eb.BaseDir, filename)
	}
	tmpl := eb.NewTemplate(StripPathPrefixAt(filepath.ToSlash(filename), "views/"))
	tstr, err := eb.Open(filename)
	if err != nil {
		return nil, err
	}
	return tmpl.Parse(tstr)
}

// ParseFiles method parses given files with given template instance.
func (eb *EngineBase) ParseFiles(t *template.Template, filenames ...string) (*template.Template, error) {
	for _, filename := range filenames {
		s, err := eb.Open(filename)
		if err != nil {
			return nil, err
		}

		name := filepath.Base(filename)
		var tmpl *template.Template
		if t == nil {
			t = eb.NewTemplate(name)
		}
		if name == t.Name() {
			tmpl = t
		} else {
			tmpl = t.New(name)
		}
		if _, err = tmpl.Parse(s); err != nil {
			return nil, err
		}
	}

	return t, nil
}

// Get method returns the template based given name if found, otherwise nil.
func (eb *EngineBase) Get(layout, tpath, tmplName string) (*template.Template, error) {
	if eb.hotReload && eb.Name == "go" {
		key := path.Join(tpath, tmplName)
		if !eb.CaseSensitive {
			key = strings.ToLower(key)
		}

		if ess.IsStrEmpty(layout) {
			return eb.ParseFile(path.Join(eb.BaseDir, key))
		}
		return eb.ParseFiles(eb.NewTemplate(key),
			path.Join(eb.BaseDir, "layouts", layout),
			path.Join(eb.BaseDir, key))
	}

	if ess.IsStrEmpty(layout) {
		layout = noLayout
	}

	if tmpls, found := eb.Templates[layout]; found {
		key := path.Join(tpath, tmplName)
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

// SetHotReload method set teh view engine mode into hot reload without watcher.
func (eb *EngineBase) SetHotReload(r bool) {
	eb.hotReload = r
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
	baseDir := path.Join(eb.BaseDir, "layouts")
	if !eb.VFS.IsExists(baseDir) {
		return nil, fmt.Errorf("%sviewengine: layouts base dir is not exists: %s", eb.Name, baseDir)
	}

	return eb.VFS.Glob(path.Join(baseDir, "*"+eb.FileExt))
}

// DirsPath method returns all sub directories from `<view-base-dir>/<sub-dir-name>`.
// if it not exists returns error.
func (eb *EngineBase) DirsPath(subDir string) ([]string, error) {
	baseDir := path.Join(eb.BaseDir, subDir)
	if !eb.VFS.IsExists(baseDir) {
		return nil, fmt.Errorf("%sviewengine: %s base dir is not exists: %s", eb.Name, subDir, baseDir)
	}
	return eb.VFS.Dirs(baseDir)
}

// FilesPath method returns all file path from `<view-base-dir>/<sub-dir-name>`.
// if it not exists returns error.
func (eb *EngineBase) FilesPath(subDir string) ([]string, error) {
	baseDir := path.Join(eb.BaseDir, subDir)
	if !eb.VFS.IsExists(baseDir) {
		return nil, fmt.Errorf("%sviewengine: %s base dir is not exists: %s", eb.Name, subDir, baseDir)
	}
	return eb.VFS.Files(baseDir)
}

// NewTemplate method return new instance on `template.Template` initialized with
// key, template funcs and delimiters.
func (eb *EngineBase) NewTemplate(key string) *template.Template {
	return template.New(key).Funcs(TemplateFuncMap).Delims(eb.LeftDelim, eb.RightDelim)
}
