// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/view source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package atemplate

import (
	"html/template"

	"aahframework.org/log.v0"
)

// tmplSafeHTML method outputs given HTML as-is, use it with care.
func tmplSafeHTML(str string) template.HTML {
	return template.HTML(str)
}

// tmplImport method renders given template with View Args and imports into
// current template.
func tmplImport(name string, viewArgs map[string]interface{}) template.HTML {
	tmplStr, err := commonTemplate.Execute(name, viewArgs)
	if err != nil {
		log.Error(err)
		return template.HTML("")
	}

	return tmplSafeHTML(tmplStr)
}
