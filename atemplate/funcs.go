// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package atemplate

import "html/template"

// tmplSafeHTML method outputs given HTML as-is, use it with care.
func tmplSafeHTML(str string) template.HTML {
	return template.HTML(str)
}

func init() {
	AddTemplateFunc(template.FuncMap{
		"safeHTML": tmplSafeHTML,
	})
}
