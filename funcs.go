// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package atemplate

import (
	"bytes"
	"html/template"
)

// tmplSafeHTML method outputs given HTML as-is, use it with care.
func tmplSafeHTML(str string) template.HTML {
	return template.HTML(str)
}

// tmplImport method renders given template with View Args and imports into
// current template.
func tmplImport(name string, viewArgs map[string]interface{}) template.HTML {
	tmpl := commonTemplates.Lookup(name)
	if tmpl != nil {
		buf := bufPool.Get().(*bytes.Buffer)
		defer func() {
			buf.Reset()
			bufPool.Put(buf)
		}()

		if err := tmpl.Execute(buf, viewArgs); err == nil {
			return tmplSafeHTML(buf.String())
		}
	}
	return tmplSafeHTML("")
}
