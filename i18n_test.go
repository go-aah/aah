// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"path/filepath"
	"testing"

	"aahframework.org/ahttp.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/test.v0/assert"
)

func TestI18nAll(t *testing.T) {
	testdataPath := getTestdataPath()
	i18nDir := filepath.Join(testdataPath, appI18nDir())

	err := initI18n(i18nDir)
	assert.FailNowOnError(t, err, "")
	assert.NotNil(t, AppI18n())
	assert.True(t, ess.IsSliceContainsString(AppI18nLocales(), "en"))

	viewArgs := map[string]interface{}{}
	localeEnUS := ahttp.ToLocale(&ahttp.AcceptSpec{Value: "en-US", Raw: "en-US"})

	v1 := tmplI18n(viewArgs, "label.pages.site.get_involved.title")
	assert.Equal(t, "", string(v1))

	viewArgs[keyLocale] = localeEnUS
	v2 := tmplI18n(viewArgs, "label.pages.site.get_involved.title")
	assert.Equal(t, "en-US: Get Involved - aah web framework for Go", string(v2))

	v3 := tmplI18n(viewArgs, "label.pages.site.with_args.title", "My Page", 1)
	assert.Equal(t, "en-US: My Page no 1 - aah web framework for Go", string(v3))

	appI18n = nil
	assert.True(t, len(AppI18nLocales()) == 0)
	assert.Nil(t, initI18n(filepath.Join(i18nDir, "not-exists")))
}
