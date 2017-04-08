// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/view source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package atemplate

import (
	"errors"
	"fmt"
	"html/template"
	"path/filepath"
	"strings"

	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
)

// GoViewEngine implements the partial inheritance support with Go templates.
type GoViewEngine struct {
	cfg           *config.Config
	baseDir       string
	layouts       map[string]*Templates
	caseSensitive bool
}

// Init method initialize a template engine with given aah application config
// and application views base path.
func (ge *GoViewEngine) Init(appCfg *config.Config, baseDir string) error {
	if !ess.IsFileExists(baseDir) {
		return fmt.Errorf("goviewengine: views base dir is not exists: %s", baseDir)
	}

	// initialize common templates
	if err := commonTemplate.Init(appCfg, baseDir); err != nil {
		return err
	}

	ge.cfg = appCfg
	ge.baseDir = baseDir
	ge.layouts = make(map[string]*Templates)
	ge.caseSensitive = ge.cfg.BoolDefault("view.case_sensitive", false)

	layoutsBaseDir := filepath.Join(ge.baseDir, "layouts")
	if !ess.IsFileExists(layoutsBaseDir) {
		return fmt.Errorf("goviewengine: layouts base dir is not exists: %s", layoutsBaseDir)
	}

	pagesBaseDir := filepath.Join(ge.baseDir, "pages")
	if !ess.IsFileExists(pagesBaseDir) {
		return fmt.Errorf("goviewengine: pages base dir is not exists: %s", pagesBaseDir)
	}

	viewFileExt := ge.cfg.StringDefault("view.ext", ".html")
	layouts, err := filepath.Glob(filepath.Join(layoutsBaseDir, "*"+viewFileExt))
	if err != nil {
		return err
	}

	pageDirs, err := ess.DirsPath(pagesBaseDir, true)
	if err != nil {
		return err
	}

	return ge.processTemplates(layouts, pageDirs, "*"+viewFileExt)
}

// Get method returns the template based given name if found, otherwise nil.
func (ge *GoViewEngine) Get(layout, path, tmplName string) (*template.Template, error) {
	if l, found := ge.layouts[layout]; found {
		key := TemplateKey(filepath.Join(path, tmplName))
		if ge.caseSensitive {
			if t, found := l.Template[key]; found {
				return t, nil
			}
		} else {
			if t, found := l.TemplateLower[strings.ToLower(key)]; found {
				return t, nil
			}
		}
	}

	return nil, ErrTemplateNotFound
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// GoViewEngine unexported methods
//___________________________________

// processTemplates method process the layouts and pages dir wise.
func (ge *GoViewEngine) processTemplates(layouts, pageDirs []string, filePattern string) error {
	errorOccurred := false
	for _, layout := range layouts {
		lTemplate := &Templates{
			Template:      make(map[string]*template.Template),
			TemplateLower: make(map[string]*template.Template),
		}

		for _, dir := range pageDirs {
			files, err := filepath.Glob(filepath.Join(dir, filePattern))
			if err != nil {
				log.Error(err)
				errorOccurred = true
				continue
			}

			if len(files) == 0 {
				continue
			}

			for _, tmplFile := range files {
				files := append([]string{}, tmplFile, layout)

				// create key and init template with funcs
				tmplKey := TemplateKey(tmplFile)
				log.Tracef("Template Key: %s", tmplKey)

				tmpl := template.New(tmplKey).Funcs(TemplateFuncMap)

				// Set custom delimiters from aah.conf
				if ge.cfg.IsExists("view.delimiters") {
					delimiters := strings.Split(ge.cfg.StringDefault("view.delimiters", "{{.}}"), ".")
					if len(delimiters) == 2 {
						tmpl.Delims(delimiters[0], delimiters[1])
					} else {
						log.Error("goviewengine: config 'view.delimiters' value is not valid")
					}
				}

				log.Tracef("Parsing Templates[%s]: %s", tmplKey, files)
				if _, err = tmpl.ParseFiles(files...); err != nil {
					log.Error(err)
					errorOccurred = true
					continue
				}

				lTemplate.Template[tmplKey] = tmpl
				lTemplate.TemplateLower[strings.ToLower(tmplKey)] = tmpl
			}
		}
		ge.layouts[strings.ToLower(filepath.Base(layout))] = lTemplate
	}

	if errorOccurred {
		return errors.New("goviewengine: error processing templates, check the log")
	}

	return nil
}

func init() {
	_ = AddEngine("go", &GoViewEngine{})
}
