// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org/view source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package view

import (
	"html/template"
	"path/filepath"
	"strings"

	"aahframework.org/log.v0"
)

// tmplSafeHTML method outputs given HTML as-is, use it with care.
func (e *GoViewEngine) tmplSafeHTML(str string) template.HTML {
	return template.HTML(str)
}

// tmplInclude method renders given template with View Args and imports into
// current template.
func (e *GoViewEngine) tmplInclude(name string, viewArgs map[string]interface{}) template.HTML {
	if !strings.HasPrefix(name, "common") {
		name = "common/" + name
	}

	name = filepath.ToSlash(name)
	var err error
	var tmpl *template.Template
	if e.hotReload {
		if tmpl, err = e.ParseFile(name); err != nil {
			log.Errorf("goviewengine: %s", err)
			return e.tmplSafeHTML("")
		}
	} else {
		tmpl = commonTemplates.Lookup(name)
	}

	if tmpl == nil {
		log.Warnf("goviewengine: common template not found: %s", name)
		return e.tmplSafeHTML("")
	}

	buf := acquireBuffer()
	defer releaseBuffer(buf)
	if err = tmpl.Execute(buf, viewArgs); err != nil {
		log.Errorf("goviewengine: %s", err)
		return e.tmplSafeHTML("")
	}

	return e.tmplSafeHTML(buf.String())
}
