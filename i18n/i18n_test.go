// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package i18n

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aahframework.org/aah/ahttp"
	"aahframework.org/test/assert"
)

func TestLoadMessage(t *testing.T) {
	wd, _ := os.Getwd()
	_ = Load(filepath.Join(wd, "testdata"))
	_ = Load(filepath.Join(wd, "testdata", "english", "messages.en"))
	_ = Load(filepath.Join(wd, "testdata", "english", "message-not-exists.en"))

	locales := Locales()

	// Assert all loaded locales fr-ca en en-gb en-us fr
	assert.True(t, isExists(locales, "en-us"))
	assert.True(t, isExists(locales, "en"))
	assert.True(t, isExists(locales, "fr-ca"))
	assert.True(t, isExists(locales, "fr"))
	assert.True(t, isExists(locales, "en-gb"))
}

func TestMsgRetrive_enUS(t *testing.T) {
	wd, _ := os.Getwd()
	_ = Load(filepath.Join(wd, "testdata"))

	locale := ahttp.Locale{Raw: "en-US", Language: "en", Region: "US"}

	homeLabel := Msg(&locale, "label.home")
	assert.Equal(t, "Home USA", homeLabel)

	prevLabel := Msg(&locale, "label.paginate.prev")
	assert.Equal(t, "Previous", prevLabel)

	addUserLabel := Msg(&locale, "label.add", "User")
	assert.Equal(t, "Add User", addUserLabel)
}

func TestMsgRetrive_enGB(t *testing.T) {
	wd, _ := os.Getwd()
	_ = Load(filepath.Join(wd, "testdata"))

	locale := ahttp.Locale{Raw: "en-GB", Language: "en", Region: "GB"}

	homeLabel := Msg(&locale, "label.home")
	assert.Equal(t, "Home UK", homeLabel)

	addUserLabel := Msg(&locale, "label.add", "User")
	assert.Equal(t, "Add User", addUserLabel)

	prevLabel := Msg(&locale, "label.paginate.prev")
	assert.Equal(t, "Previous", prevLabel)

	nfLabel := Msg(&locale, "label.paginate.notfound")
	assert.Equal(t, "", nfLabel)
}

func TestMsgRetrive_en(t *testing.T) {
	wd, _ := os.Getwd()
	_ = Load(filepath.Join(wd, "testdata"))

	locale := ahttp.Locale{Raw: "en", Language: "en"}

	homeLabel := Msg(&locale, "label.home")
	assert.Equal(t, "", homeLabel)

	addUserLabel := Msg(&locale, "label.add", "User")
	assert.Equal(t, "", addUserLabel)

	prevLabel := Msg(&locale, "label.paginate.prev")
	assert.Equal(t, "Previous", prevLabel)

	nextLabel := Msg(&locale, "label.paginate.next")
	assert.Equal(t, "Next", nextLabel)

	lastLabel := Msg(&locale, "label.paginate.last")
	assert.Equal(t, "Last", lastLabel)
}

func TestMsgRetrive_frCA(t *testing.T) {
	wd, _ := os.Getwd()
	_ = Load(filepath.Join(wd, "testdata"))

	locale := ahttp.Locale{Raw: "fr-CA", Language: "fr", Region: "CA"}

	homeLabel := Msg(&locale, "label.home")
	assert.Equal(t, "Accueil fr-CA", homeLabel)

	addUserLabel := Msg(&locale, "label.add", "Utilisateur")
	assert.Equal(t, "Ajouter Utilisateur", addUserLabel)

	showUserLabel := Msg(&locale, "label.show", "Utilisateur")
	assert.Equal(t, "Montrer Utilisateur", showUserLabel)
}

func TestMsgRetrive_fr(t *testing.T) {
	wd, _ := os.Getwd()
	_ = Load(filepath.Join(wd, "testdata"))

	locale := ahttp.Locale{Raw: "fr", Language: "fr"}

	prevLabel := Msg(&locale, "label.paginate.prev")
	assert.Equal(t, "Précédent", prevLabel)

	nextLabel := Msg(&locale, "label.paginate.next")
	assert.Equal(t, "Suivant", nextLabel)
}

func TestMsgRetrive_it(t *testing.T) {
	wd, _ := os.Getwd()
	_ = Load(filepath.Join(wd, "testdata"))

	locale := ahttp.Locale{Raw: "it-IT", Language: "it", Region: "IT"}

	prevLabel := Msg(&locale, "label.paginate.prev")
	assert.Equal(t, "Precedente", prevLabel)

	nextLabel := Msg(&locale, "label.paginate.next")
	assert.Equal(t, "Successivo", nextLabel)

}

func TestMsgRetriveNotFoundLocale(t *testing.T) {
	wd, _ := os.Getwd()
	_ = Load(filepath.Join(wd, "testdata"))

	locale := ahttp.Locale{Raw: "pl-PT", Language: "pl", Region: "PL"}

	notFoundStore := Msg(&locale, "store.not.exists")
	assert.Equal(t, "", notFoundStore)
}

func isExists(s []string, v string) bool {
	for _, iv := range s {
		if strings.EqualFold(iv, v) {
			return true
		}
	}
	return false
}
