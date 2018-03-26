// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/view source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package view

import (
	"html/template"
	"path/filepath"
	"strings"

	"aahframework.org/log.v0"
)

// tmplSafeHTML method outputs given HTML as-is, use it with care.
func tmplSafeHTML(str string) template.HTML {
	return template.HTML(str)
}

// tmplInclude method renders given template with View Args and imports into
// current template.
func tmplInclude(name string, viewArgs map[string]interface{}) template.HTML {
	if !strings.HasPrefix(name, "common") {
		name = "common/" + name
	}
	name = filepath.ToSlash(name)

	tmpl := commonTemplates.Lookup(name)
	if tmpl == nil {
		log.Warnf("goviewengine: common template not found: %s", name)
		return tmplSafeHTML("")
	}

	buf := acquireBuffer()
	defer releaseBuffer(buf)
	if err := tmpl.Execute(buf, viewArgs); err != nil {
		log.Error(err)
		return template.HTML("")
	}

	return tmplSafeHTML(buf.String())
}
