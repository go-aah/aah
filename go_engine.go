// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/view source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package view

import (
	"bytes"
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
	*EngineBase
}

// Init method initialize a template engine with given aah application config
// and application views base path.
func (e *GoViewEngine) Init(appCfg *config.Config, baseDir string) error {
	if !ess.IsFileExists(baseDir) {
		return fmt.Errorf("goviewengine: views base dir is not exists: %s", baseDir)
	}

	if e.EngineBase == nil {
		e.EngineBase = &EngineBase{}
	}

	if err := e.EngineBase.Init(appCfg, baseDir, "go", ".html"); err != nil {
		return err
	}

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
	layouts, err := e.LayoutFiles()
	if err != nil {
		return err
	}

	// load layout templates
	if err = e.loadLayoutTemplates(layouts); err != nil {
		return err
	}

	if !e.IsLayoutEnabled {
		// since pages directory processed above, no error expected here
		_ = e.loadNonLayoutTemplates("pages")
	}

	if ess.IsFileExists(filepath.Join(e.BaseDir, "errors")) {
		if err = e.loadNonLayoutTemplates("errors"); err != nil {
			return err
		}
	}

	return nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// GoViewEngine unexported methods
//___________________________________

func (e *GoViewEngine) loadCommonTemplates() error {
	commons, err := e.FilesPath("common")
	if err != nil {
		return err
	}

	commonTemplates = &Templates{}
	bufPool = &sync.Pool{New: func() interface{} { return &bytes.Buffer{} }}
	prefix := filepath.Dir(e.BaseDir)
	for _, file := range commons {
		if !strings.HasSuffix(file, e.FileExt) {
			log.Warnf("goviewengine: not a valid template extension[%s]: %s", e.FileExt, TrimPathPrefix(prefix, file))
			continue
		}

		tmplKey := StripPathPrefixAt(filepath.ToSlash(file), "views/")
		tmpl := e.NewTemplate(tmplKey)

		tbytes, err := ioutil.ReadFile(file)
		if err != nil {
			return err
		}

		tstr := e.AntiCSRFField.InsertOnString(string(tbytes))
		if tmpl, err = tmpl.Parse(tstr); err != nil {
			return err
		}

		if err = commonTemplates.Add(tmplKey, tmpl); err != nil {
			return err
		}
	}

	return nil
}

func (e *GoViewEngine) loadLayoutTemplates(layouts []string) error {
	dirs, err := e.DirsPath("pages")
	if err != nil {
		return err
	}

	prefix := filepath.Dir(e.BaseDir)
	var errs []error
	for _, layout := range layouts {
		layoutKey := strings.ToLower(filepath.Base(layout))

		for _, dir := range dirs {
			files, err := filepath.Glob(filepath.Join(dir, "*"+e.FileExt))
			if err != nil {
				errs = append(errs, err)
				continue
			}

			for _, file := range files {
				tfiles := []string{file, layout}
				tmplKey := StripPathPrefixAt(filepath.ToSlash(file), "views/")
				tmpl := e.NewTemplate(tmplKey)
				tmplfiles := e.AntiCSRFField.InsertOnFiles(tfiles...)

				log.Tracef("Parsing files: %s", TrimPathPrefix(prefix, tfiles...))
				if tmpl, err = tmpl.ParseFiles(tmplfiles...); err != nil {
					errs = append(errs, err)
					continue
				}

				if err = e.AddTemplate(layoutKey, tmplKey, tmpl); err != nil {
					errs = append(errs, err)
					continue
				}
			}
		}
	}

	return e.ParseErrors(errs)
}

func (e *GoViewEngine) loadNonLayoutTemplates(scope string) error {
	dirs, err := e.DirsPath(scope)
	if err != nil {
		return err
	}

	prefix := filepath.Dir(e.BaseDir)
	var errs []error
	for _, dir := range dirs {
		files, err := filepath.Glob(filepath.Join(dir, "*"+e.FileExt))
		if err != nil {
			errs = append(errs, err)
			continue
		}

		for _, file := range files {
			tmplKey := noLayout + "-" + StripPathPrefixAt(filepath.ToSlash(file), "views/")
			tmpl := e.NewTemplate(tmplKey)
			fileBytes, _ := ioutil.ReadFile(file)
			fileStr := e.AntiCSRFField.InsertOnString(string(fileBytes))

			log.Tracef("Parsing file: %s", TrimPathPrefix(prefix, file))
			if tmpl, err = tmpl.Parse(fileStr); err != nil {
				errs = append(errs, err)
				continue
			}

			if err = e.AddTemplate(noLayout, tmplKey, tmpl); err != nil {
				errs = append(errs, err)
				continue
			}
		}
	}

	return e.ParseErrors(errs)
}

func init() {
	_ = AddEngine("go", &GoViewEngine{})

	// Add template func
	AddTemplateFunc(template.FuncMap{
		"safeHTML": tmplSafeHTML,
	})
}
