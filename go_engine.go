// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/view source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package view

import (
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"path/filepath"
	"strings"

	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0-unstable"
)

const noLayout = "nolayout"

// GoViewEngine implements the partial inheritance support with Go templates.
type GoViewEngine struct {
	cfg                    *config.Config
	baseDir                string
	layouts                map[string]*Templates
	viewFileExt            string
	caseSensitive          bool
	isDefaultLayoutEnabled bool
	delimiters             []string
	antiCSRFField          string
	antiCSRFInserter       *strings.Replacer
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
	ge.viewFileExt = ge.cfg.StringDefault("view.ext", ".html")
	ge.caseSensitive = ge.cfg.BoolDefault("view.case_sensitive", false)
	ge.isDefaultLayoutEnabled = ge.cfg.BoolDefault("view.default_layout", true)

	ge.delimiters = strings.Split(ge.cfg.StringDefault("view.delimiters", "{{.}}"), ".")
	if len(ge.delimiters) != 2 || ess.IsStrEmpty(ge.delimiters[0]) || ess.IsStrEmpty(ge.delimiters[1]) {
		return fmt.Errorf("goviewengine: config 'view.delimiters' value is invalid")
	}

	// anti CSRF
	ge.antiCSRFField = `	<input type="hidden" name="anti_csrf_token" value="{{ anitcsrftoken . }}">
	</form>`
	ge.antiCSRFField = strings.Replace(ge.antiCSRFField, "{{", ge.delimiters[0], -1)
	ge.antiCSRFField = strings.Replace(ge.antiCSRFField, "}}", ge.delimiters[1], -1)
	ge.antiCSRFInserter = strings.NewReplacer("</form>", ge.antiCSRFField)

	layoutsBaseDir := filepath.Join(ge.baseDir, "layouts")
	if !ess.IsFileExists(layoutsBaseDir) {
		return fmt.Errorf("goviewengine: layouts base dir is not exists: %s", layoutsBaseDir)
	}

	pagesBaseDir := filepath.Join(ge.baseDir, "pages")
	if !ess.IsFileExists(pagesBaseDir) {
		return fmt.Errorf("goviewengine: pages base dir is not exists: %s", pagesBaseDir)
	}

	layouts, err := filepath.Glob(filepath.Join(layoutsBaseDir, "*"+ge.viewFileExt))
	if err != nil {
		return err
	}

	pageDirs, err := ess.DirsPath(pagesBaseDir, true)
	if err != nil {
		return err
	}

	if err = ge.processLayoutTemplates(layouts, pageDirs); err != nil {
		return err
	}

	if !ge.isDefaultLayoutEnabled {
		if err = ge.processNolayoutTemplates(pageDirs); err != nil {
			return err
		}
	}

	errorPagesDir := filepath.Join(ge.baseDir, "errors")
	if ess.IsFileExists(errorPagesDir) {
		if err = ge.processNolayoutTemplates([]string{errorPagesDir}); err != nil {
			return err
		}
	}

	return nil
}

// Get method returns the template based given name if found, otherwise nil.
func (ge *GoViewEngine) Get(layout, path, tmplName string) (*template.Template, error) {
	if ess.IsStrEmpty(layout) {
		layout = noLayout
	}

	if l, found := ge.layouts[layout]; found {
		key := parseKey("", filepath.Join(path, tmplName))
		if layout == noLayout {
			key = noLayout + key
		}
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

func (ge *GoViewEngine) processLayoutTemplates(layouts, dirs []string) error {
	var errs []error
	for _, layout := range layouts {
		lTemplate := &Templates{
			Template:      make(map[string]*template.Template),
			TemplateLower: make(map[string]*template.Template),
		}

		for _, dir := range dirs {
			files, err := filepath.Glob(filepath.Join(dir, "*"+ge.viewFileExt))
			if err != nil {
				errs = append(errs, err)
				continue
			}

			for _, file := range files {
				tfiles := []string{file, layout}
				tmplKey := parseKey(ge.baseDir, file)
				tmpl := template.New(tmplKey).Funcs(TemplateFuncMap)
				tmpl.Delims(ge.delimiters[0], ge.delimiters[1])
				tmplfiles := ge.processAntiCSRFField(tfiles...)

				log.Tracef("Parsing files [%s]: %s", tmplKey, ge.trimAppBaseDir(tfiles...))
				if tmpl, err = tmpl.ParseFiles(tmplfiles...); err != nil {
					errs = append(errs, err)
					continue
				}

				lTemplate.Template[tmplKey] = tmpl
				lTemplate.TemplateLower[strings.ToLower(tmplKey)] = tmpl
			}
		}

		ge.layouts[strings.ToLower(filepath.Base(layout))] = lTemplate
	}

	return handleParseError(errs)
}

func (ge *GoViewEngine) processNolayoutTemplates(dirs []string) error {
	if _, found := ge.layouts[noLayout]; !found {
		ge.layouts[noLayout] = &Templates{
			Template:      make(map[string]*template.Template),
			TemplateLower: make(map[string]*template.Template),
		}
	}

	var errs []error
	prefix := strings.TrimSuffix(ge.baseDir, "views")
	for _, dir := range dirs {
		files, err := filepath.Glob(filepath.Join(dir, "*"+ge.viewFileExt))
		if err != nil {
			errs = append(errs, err)
			continue
		}

		for _, file := range files {
			tmplKey := parseKey(ge.baseDir, file)
			tmpl := template.New(tmplKey).Funcs(TemplateFuncMap)
			tmpl.Delims(ge.delimiters[0], ge.delimiters[1])
			fileBytes, _ := ioutil.ReadFile(file)
			fileContent := ge.antiCSRFInserter.Replace(string(fileBytes))

			log.Tracef("Parsing file [%s]: %s", tmplKey, trimPathPrefix(file, prefix))
			if tmpl, err = tmpl.Parse(fileContent); err != nil {
				errs = append(errs, err)
				continue
			}

			ge.layouts[noLayout].Template[noLayout+tmplKey] = tmpl
			ge.layouts[noLayout].TemplateLower[strings.ToLower(noLayout+tmplKey)] = tmpl
		}
	}

	return handleParseError(errs)
}

func (ge *GoViewEngine) trimAppBaseDir(files ...string) string {
	var fs []string
	prefix := strings.TrimSuffix(ge.baseDir, "views")
	for _, f := range files {
		fs = append(fs, trimPathPrefix(f, prefix))
	}
	return strings.Join(fs, ", ")
}

func (ge *GoViewEngine) processAntiCSRFField(filenames ...string) []string {
	var files []string
	tmpDir, _ := ioutil.TempDir("", "anti_csrf")

	for _, f := range filenames {
		fileBytes, err := ioutil.ReadFile(f)
		if err != nil {
			files = append(files, f)
			continue
		}

		file := string(fileBytes)
		rfile := ge.trimAppBaseDir(f)
		if strings.Contains(file, "</form>") {
			log.Tracef("Adding Anti-CSRF field into %s", rfile)
			file = ge.antiCSRFInserter.Replace(file)
			fpath := filepath.Join(tmpDir, rfile)
			_ = ess.MkDirAll(filepath.Dir(fpath), 0755)
			_ = ioutil.WriteFile(fpath, []byte(file), 0755)
			files = append(files, fpath)
			continue
		}

		files = append(files, f)
	}
	return files
}

func handleParseError(errs []error) error {
	if len(errs) > 0 {
		for _, e := range errs {
			log.Error(e)
		}
		return errors.New("goviewengine: error processing templates, check the log")
	}
	return nil
}

func init() {
	_ = AddEngine("go", &GoViewEngine{})
}
