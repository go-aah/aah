// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package view

import (
	"html/template"
	"path/filepath"
	"strings"

	"aahframe.work/log"
)

// tmplSafeHTML method outputs given HTML as-is, use it with care.
func (e *GoViewEngine) tmplSafeHTML(str string) template.HTML {
	return template.HTML(str)
}

// tmplInclude method renders given template with View Args and imports into
// current template.
func (e *GoViewEngine) tmplInclude(name string, viewArgs map[string]interface{}) template.HTML {
	if len(name) == 0 {
		log.Error("goviewengine: empty template filename suppiled to func 'include'")
		return e.tmplSafeHTML("")
	}
	if name[0] == '/' {
		if strings.HasPrefix(name, "/views") {
			name = strings.TrimPrefix(name, "/views")
			log.Warnf("goviewengine: use without prefix of '/views' with func 'include' [/views%s => %s]", name, name)
		}
	} else if !strings.HasPrefix(name, "common") {
		// TODO existing behaviour will be removed in the future release
		name = "common/" + name
	}
	var err error
	var tmpl *template.Template
	if e.hotReload || name[0] == '/' {
		if tmpl, err = e.ParseFile(name); err != nil {
			log.Errorf("goviewengine: %s", err)
			return e.tmplSafeHTML("")
		}
	} else {
		tmpl = commonTemplates.Lookup(filepath.ToSlash(name))
	}
	if tmpl == nil {
		log.Warnf("goviewengine: common template not found: %s", name)
		return e.tmplSafeHTML("")
	}
	buf := acquireBuilder()
	defer releaseBuilder(buf)
	if err = tmpl.Execute(buf, viewArgs); err != nil {
		log.Errorf("goviewengine: %s", err)
		return e.tmplSafeHTML("")
	}
	return e.tmplSafeHTML(buf.String())
}
