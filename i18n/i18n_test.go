// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package i18n

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"aahframe.work/ahttp"
	"aahframe.work/config"
	"aahframe.work/essentials"
	"aahframe.work/log"
	"github.com/stretchr/testify/assert"
)

func TestNewStore(t *testing.T) {
	wd, _ := os.Getwd()

	store := New(
		logOption(),
		Dirs(filepath.Join(wd, "testdata")),
		Files(
			filepath.Join(wd, "testdata", "english", "messages.en"),
			filepath.Join(wd, "testdata", "english", "message-not-exists.en"),
		),
	)

	locales := store.Locales()

	// Assert all loaded locales fr-ca en en-gb en-us fr
	assert.True(t, ess.IsSliceContainsString(locales, "en-us"))
	assert.True(t, ess.IsSliceContainsString(locales, "en"))
	assert.True(t, ess.IsSliceContainsString(locales, "fr-ca"))
	assert.True(t, ess.IsSliceContainsString(locales, "fr"))
	assert.True(t, ess.IsSliceContainsString(locales, "en-gb"))
	assert.Equal(t, "en", store.DefaultLocale())
}

func TestMsgRetrive_enUS(t *testing.T) {
	wd, _ := os.Getwd()

	store := New(logOption(), Dirs(filepath.Join(wd, "testdata")))

	locale := ahttp.Locale{Raw: "en-US", Language: "en", Region: "US"}

	homeLabel := store.Lookup(&locale, "label.home")
	assert.Equal(t, "Home USA", homeLabel)

	prevLabel := store.Lookup(&locale, "label.paginate.prev")
	assert.Equal(t, "Previous", prevLabel)

	addUserLabel := store.Lookup(&locale, "label.add", "User")
	assert.Equal(t, "Add User", addUserLabel)
}

func TestMsgRetrive_enGB(t *testing.T) {
	wd, _ := os.Getwd()
	store := New(logOption(), DefaultLocale("en"), Dirs(filepath.Join(wd, "testdata")))
	assert.Equal(t, "en", store.DefaultLocale())

	locale := ahttp.Locale{Raw: "en-GB", Language: "en", Region: "GB"}

	homeLabel := store.Lookup(&locale, "label.home")
	assert.Equal(t, "Home UK", homeLabel)

	addUserLabel := store.Lookup(&locale, "label.add", "User")
	assert.Equal(t, "Add User", addUserLabel)

	prevLabel := store.Lookup(&locale, "label.paginate.prev")
	assert.Equal(t, "Previous", prevLabel)

	nfLabel := store.Lookup(&locale, "label.paginate.notfound")
	assert.Equal(t, "label.paginate.notfound", nfLabel)
}

func TestMsgRetrive_en(t *testing.T) {
	wd, _ := os.Getwd()
	store := New(logOption(), DefaultLocale("en"), Dirs(filepath.Join(wd, "testdata")))

	locale := ahttp.Locale{Raw: "en", Language: "en"}

	homeLabel := store.Lookup(nil, "label.home")
	assert.Equal(t, "label.home", homeLabel)

	addUserLabel := store.Lookup(&locale, "label.add", "User")
	assert.Equal(t, "label.add", addUserLabel)

	prevLabel := store.Lookup(&locale, "label.paginate.prev")
	assert.Equal(t, "Previous", prevLabel)

	nextLabel := store.Lookup(&locale, "label.paginate.next")
	assert.Equal(t, "Next", nextLabel)

	lastLabel := store.Lookup(nil, "label.paginate.last")
	assert.Equal(t, "Last", lastLabel)
}

func TestMsgRetrive_frCA(t *testing.T) {
	wd, _ := os.Getwd()
	store := New(logOption(), Dirs(filepath.Join(wd, "testdata")))

	locale := ahttp.Locale{Raw: "fr-CA", Language: "fr", Region: "CA"}

	homeLabel := store.Lookup(&locale, "label.home")
	assert.Equal(t, "Accueil fr-CA", homeLabel)

	addUserLabel := store.Lookup(&locale, "label.add", "Utilisateur")
	assert.Equal(t, "Ajouter Utilisateur", addUserLabel)

	showUserLabel := store.Lookup(&locale, "label.show", "Utilisateur")
	assert.Equal(t, "Montrer Utilisateur", showUserLabel)
}

func TestMsgRetrive_fr(t *testing.T) {
	wd, _ := os.Getwd()
	store := New(logOption(), Dirs(filepath.Join(wd, "testdata")))

	locale := ahttp.Locale{Raw: "fr", Language: "fr"}

	prevLabel := store.Lookup(&locale, "label.paginate.prev")
	assert.Equal(t, "Précédent", prevLabel)

	nextLabel := store.Lookup(&locale, "label.paginate.next")
	assert.Equal(t, "Suivant", nextLabel)
}

func TestMsgRetrive_it(t *testing.T) {
	wd, _ := os.Getwd()
	store := New(logOption(), Dirs(filepath.Join(wd, "testdata")), VFS(nil))

	locale := ahttp.Locale{Raw: "it-IT", Language: "it", Region: "IT"}

	prevLabel := store.Lookup(&locale, "label.paginate.prev")
	assert.Equal(t, "Precedente", prevLabel)

	nextLabel := store.Lookup(&locale, "label.paginate.next")
	assert.Equal(t, "Successivo", nextLabel)
}

func TestMsgRetriveNotFoundLocale(t *testing.T) {
	wd, _ := os.Getwd()
	store := New(logOption(), Dirs(filepath.Join(wd, "testdata")), VFS(nil))

	locale := ahttp.Locale{Raw: "pl-PT", Language: "pl", Region: "PL"}

	notFoundStore := store.Lookup(&locale, "store.not.exists")
	assert.Equal(t, "store.not.exists", notFoundStore)
}

func logOption() Option {
	l, _ := log.New(config.NewEmpty())
	l.SetWriter(ioutil.Discard)
	return Logger(l)
}
