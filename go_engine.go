// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/view source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package view

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"path/filepath"
	"strings"
	"sync"

	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
)

const noLayout = "nolayout"

var (
	commonTemplates *Templates
	bufPool         *sync.Pool
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// type GoViewEngine and its method
//___________________________________

// GoViewEngine implements the partial inheritance support with Go templates.
type GoViewEngine struct {
	cfg             *config.Config
	baseDir         string
	layouts         map[string]*Templates
	viewFileExt     string
	caseSensitive   bool
	isLayoutEnabled bool
	leftDelim       string
	rightDelim      string
	antiCSRFField   *AntiCSRFField
}

// Init method initialize a template engine with given aah application config
// and application views base path.
func (e *GoViewEngine) Init(appCfg *config.Config, baseDir string) error {
	if !ess.IsFileExists(baseDir) {
		return fmt.Errorf("goviewengine: views base dir is not exists: %s", baseDir)
	}

	e.cfg = appCfg
	e.baseDir = baseDir
	e.layouts = make(map[string]*Templates)
	e.viewFileExt = e.cfg.StringDefault("view.ext", ".html")
	e.caseSensitive = e.cfg.BoolDefault("view.case_sensitive", false)
	e.isLayoutEnabled = e.cfg.BoolDefault("view.default_layout", true)

	delimiter := strings.Split(e.cfg.StringDefault("view.delimiters", DefaultDelimiter), ".")
	if len(delimiter) != 2 || ess.IsStrEmpty(delimiter[0]) || ess.IsStrEmpty(delimiter[1]) {
		return fmt.Errorf("goviewengine: config 'view.delimiters' value is invalid")
	}
	e.leftDelim, e.rightDelim = delimiter[0], delimiter[1]

	e.layouts = make(map[string]*Templates)

	// Anti CSRF
	e.antiCSRFField = NewAntiCSRFField("go", e.leftDelim, e.rightDelim)

	// Add template func
	AddTemplateFunc(template.FuncMap{
		"import":  tmplImport,
		"include": tmplImport, // alias for import
	})

	// load common templates
	if err := e.loadCommonTemplates(); err != nil {
		return err
	}

	// collect all layouts
	layouts, err := e.findLayouts()
	if err != nil {
		return err
	}

	// load layout templates
	if err = e.loadLayoutTemplates(layouts); err != nil {
		return err
	}

	if !e.isLayoutEnabled {
		// since pages directory processed above, no error expected here
		_ = e.loadNonLayoutTemplates("pages")
	}

	if ess.IsFileExists(filepath.Join(e.baseDir, "errors")) {
		if err = e.loadNonLayoutTemplates("errors"); err != nil {
			return err
		}
	}

	return nil
}

// Get method returns the template based given name if found, otherwise nil.
func (e *GoViewEngine) Get(layout, path, tmplName string) (*template.Template, error) {
	if ess.IsStrEmpty(layout) {
		layout = noLayout
	}

	if l, found := e.layouts[layout]; found {
		key := filepath.Join(path, tmplName)
		if layout == noLayout {
			key = noLayout + "-" + key
		}

		if !e.caseSensitive {
			key = strings.ToLower(key)
		}

		if t := l.Lookup(key); t != nil {
			return t, nil
		}
	}

	return nil, ErrTemplateNotFound
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// GoViewEngine unexported methods
//___________________________________

func (e *GoViewEngine) loadCommonTemplates() error {
	baseDir := filepath.Join(e.baseDir, "common")
	if !ess.IsFileExists(baseDir) {
		return fmt.Errorf("goviewengine: common base dir is not exists: %s", baseDir)
	}

	commons, err := ess.FilesPath(baseDir, true)
	if err != nil {
		return err
	}

	commonTemplates = &Templates{}
	bufPool = &sync.Pool{New: func() interface{} { return &bytes.Buffer{} }}
	prefix := filepath.Dir(e.baseDir)
	for _, file := range commons {
		if !strings.HasSuffix(file, e.viewFileExt) {
			log.Warnf("goviewengine: not a valid template extension[%s]: %s", e.viewFileExt, TrimPathPrefix(prefix, file))
			continue
		}

		tmplKey := StripPathPrefixAt(filepath.ToSlash(file), "views/")
		tmpl := template.New(tmplKey).Funcs(TemplateFuncMap).Delims(e.leftDelim, e.rightDelim)

		tbytes, err := ioutil.ReadFile(file)
		if err != nil {
			return err
		}

		tstr := e.antiCSRFField.InsertOnString(string(tbytes))
		if tmpl, err = tmpl.Parse(tstr); err != nil {
			return err
		}

		if err = commonTemplates.Add(tmplKey, tmpl); err != nil {
			return err
		}
	}

	return nil
}

func (e *GoViewEngine) findLayouts() ([]string, error) {
	baseDir := filepath.Join(e.baseDir, "layouts")
	if !ess.IsFileExists(baseDir) {
		return nil, fmt.Errorf("goviewengine: layouts base dir is not exists: %s", baseDir)
	}

	return filepath.Glob(filepath.Join(baseDir, "*"+e.viewFileExt))
}

func (e *GoViewEngine) loadLayoutTemplates(layouts []string) error {
	baseDir := filepath.Join(e.baseDir, "pages")
	if !ess.IsFileExists(baseDir) {
		return fmt.Errorf("goviewengine: pages base dir is not exists: %s", baseDir)
	}

	dirs, err := ess.DirsPath(baseDir, true)
	if err != nil {
		return err
	}

	prefix := filepath.Dir(e.baseDir)
	var errs []error
	for _, layout := range layouts {
		layoutKey := strings.ToLower(filepath.Base(layout))
		if e.layouts[layoutKey] == nil {
			e.layouts[layoutKey] = &Templates{}
		}

		for _, dir := range dirs {
			files, err := filepath.Glob(filepath.Join(dir, "*"+e.viewFileExt))
			if err != nil {
				errs = append(errs, err)
				continue
			}

			for _, file := range files {
				tfiles := []string{file, layout}
				tmplKey := StripPathPrefixAt(filepath.ToSlash(file), "views/")
				tmpl := template.New(tmplKey).Funcs(TemplateFuncMap).Delims(e.leftDelim, e.rightDelim)
				tmplfiles := e.antiCSRFField.InsertOnFiles(tfiles...)

				log.Tracef("Parsing files: %s", TrimPathPrefix(prefix, tfiles...))
				if tmpl, err = tmpl.ParseFiles(tmplfiles...); err != nil {
					errs = append(errs, err)
					continue
				}

				if err = e.layouts[layoutKey].Add(tmplKey, tmpl); err != nil {
					errs = append(errs, err)
					continue
				}
			}
		}
	}

	return handleParseError(errs)
}

func (e *GoViewEngine) loadNonLayoutTemplates(scope string) error {
	baseDir := filepath.Join(e.baseDir, scope)
	if !ess.IsFileExists(baseDir) {
		return fmt.Errorf("goviewengine: %s base dir is not exists: %s", scope, baseDir)
	}

	dirs, err := ess.DirsPath(baseDir, true)
	if err != nil {
		return err
	}

	if e.layouts[noLayout] == nil {
		e.layouts[noLayout] = &Templates{}
	}

	prefix := filepath.Dir(e.baseDir)
	var errs []error
	for _, dir := range dirs {
		files, err := filepath.Glob(filepath.Join(dir, "*"+e.viewFileExt))
		if err != nil {
			errs = append(errs, err)
			continue
		}

		for _, file := range files {
			tmplKey := noLayout + "-" + StripPathPrefixAt(filepath.ToSlash(file), "views/")
			tmpl := template.New(tmplKey).Funcs(TemplateFuncMap).Delims(e.leftDelim, e.rightDelim)
			fileBytes, _ := ioutil.ReadFile(file)
			fileStr := e.antiCSRFField.InsertOnString(string(fileBytes))

			log.Tracef("Parsing file: %s", TrimPathPrefix(prefix, file))
			if tmpl, err = tmpl.Parse(fileStr); err != nil {
				errs = append(errs, err)
				continue
			}

			if err = e.layouts[noLayout].Add(tmplKey, tmpl); err != nil {
				errs = append(errs, err)
				continue
			}
		}
	}

	return handleParseError(errs)
}

func handleParseError(errs []error) error {
	if len(errs) > 0 {
		var msg []string
		for _, e := range errs {
			msg = append(msg, e.Error())
		}
		log.Errorf("View templates parsing error(s):\n    %s", strings.Join(msg, "\n    "))
		return errors.New("goviewengine: error processing templates, please check the log")
	}
	return nil
}

func init() {
	_ = AddEngine("go", &GoViewEngine{})

	// Add template func
	AddTemplateFunc(template.FuncMap{
		"safeHTML": tmplSafeHTML,
	})
}
