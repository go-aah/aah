// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org/i18n source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

// Package i18n is internationalization and localization support for aah
// framework. Messages config format is `forge` config syntax (go-aah/config)
// which is similar to HOCON syntax aka typesafe config.
//
// Message filename format is `message.<Language-ID>`. Language ID is combination
// of `Language + Region` or `Language` value. aah framework implements Language
// code is as per two-letter `ISO 639-1` standard and Region code is as per two-letter
// `ISO 3166-1` standard.
//
// Supported message file extension formats are (incasesensitive)
// 	1) Language + Region => en-us | en-US
// 	2) Language          => en
//
// 	For Example:
// 		message.en-US or message.en-us
// 		message.en-GB or message.en-gb
// 		message.en-CA or message.en-ca
// 		message.en
// 		message.es
// 		message.zh
// 		message.nl
// 		etc.
//
// Note: Sub directories is supported, so you can organize message files.
package i18n

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"aahframe.work/aah/ahttp"
	"aahframe.work/aah/config"
	"aahframe.work/aah/essentials"
	"aahframe.work/aah/log"
	"aahframe.work/aah/vfs"
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package methods
//______________________________________________________________________________

// New method creates aah i18n message store
func New() *I18n {
	return NewWithVFS(nil)
}

// NewWithVFS method creates aah i18n message store with given Virtual FileSystem.
func NewWithVFS(fs *vfs.VFS) *I18n {
	return &I18n{
		Store:        make(map[string]*config.Config),
		fileExtRegex: `messages\.[a-z]{2}(\-[a-zA-Z]{2})?$`,
		vfs:          fs,
	}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// I18n methods
//______________________________________________________________________________

// I18n holds the message store and related information for internationalization
// and localization.
type I18n struct {
	Store         map[string]*config.Config
	DefaultLocale string

	fileExtRegex string
	vfs          *vfs.VFS
}

// Load processes the given message file or directory and adds to the
// message store
func (s *I18n) Load(paths ...string) error {
	for _, path := range paths {
		if !vfs.IsExists(s.vfs, path) {
			log.Warnf("Path: %v not exists, let's move on", path)
			continue
		}

		if s.isDir(path) {
			_ = vfs.Walk(s.vfs, path, func(fpath string, f os.FileInfo, _ error) error {
				if !f.IsDir() {
					match, err := regexp.MatchString(s.fileExtRegex, f.Name())
					if err == nil && match {
						s.processMsgFile(fpath)
					}
				}
				return nil
			})
		} else { // if it's a file
			s.processMsgFile(path)
		}
	}

	return nil
}

// Lookup returns value by given key, locale and it supports formatting a message
// before its return. If given message key or store doesn't exists for given locale;
// Lookup method returns empty string.
// 	Lookup(locale, "i.love.aah.framework", "yes")
// The sequence and fallback order of message fetch from store is -
// 	* language and region-id (e.g.: en-US)
// 	* language (e.g.: en)
func (s *I18n) Lookup(locale *ahttp.Locale, key string, args ...interface{}) string {
	// assign default locale if nil
	if locale == nil {
		locale = ahttp.NewLocale(s.DefaultLocale)
	}

	// Lookup by language and region-id. For eg.: en-us
	store := s.findStoreByLocale(locale.String())
	if store == nil {
		log.Tracef("Locale (%v) doesn't exists in message store", locale)
		goto langStore
	}
	log.Tracef("Message is retrieved from locale: %v, key: %v", locale, key)
	if msg, found := retriveValue(store, key, args...); found {
		return msg
	}

langStore:
	store = s.findStoreByLocale(locale.Language)
	if store == nil {
		log.Tracef("Locale (%v) doesn't exists in message store", locale.Language)
		goto defaultStore
	}
	log.Tracef("Message is retrieved from locale: %v, key: %v", locale.Language, key)
	if msg, found := retriveValue(store, key, args...); found {
		return msg
	}

defaultStore: // fallback to `i18n.default` config value.
	store = s.findStoreByLocale(s.DefaultLocale)
	if store == nil {
		goto notExists
	}
	log.Tracef("Message is retrieved with 'i18n.default': %v, key: %v", s.DefaultLocale, key)
	if msg, found := retriveValue(store, key, args...); found {
		return msg
	}

notExists:
	log.Warnf("i18n key not found: %s", key)
	return ""
}

// Locales returns all the loaded locales from message store
func (s *I18n) Locales() []string {
	var locales []string
	for l := range s.Store {
		locales = append(locales, l)
	}
	return locales
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// I18n Unexported methods
//______________________________________________________________________________

func (s *I18n) processMsgFile(file string) {
	key := strings.ToLower(filepath.Ext(file)[1:])
	msgFile, err := config.VFSLoadFile(s.vfs, file)
	if err != nil {
		log.Errorf("Unable to load message file: %v, error: %v", file, err)
	}

	// merge messages if key is already exists otherwise add it
	if ms, exists := s.Store[key]; exists {
		log.Tracef("Key[%v] is already exists, let's merge it", key)
		if err = ms.Merge(msgFile); err != nil {
			log.Errorf("Error while merging message file: %v", file)
		}
	} else {
		log.Tracef("Adding to message store [%v: %v]", key, file)
		s.Store[key] = msgFile
	}
}

func (s *I18n) findStoreByLocale(locale string) *config.Config {
	if store, exists := s.Store[strings.ToLower(locale)]; exists {
		return store
	}
	return nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package Unexported methods
//______________________________________________________________________________

func retriveValue(store *config.Config, key string, args ...interface{}) (string, bool) {
	if msg, found := store.String(key); found {
		if len(args) > 0 {
			return fmt.Sprintf(msg, args...), found
		}
		return msg, found
	}
	return "", false
}

func (s *I18n) isDir(name string) bool {
	if s.vfs == nil {
		return ess.IsDir(name)
	}

	fi, err := s.vfs.Lstat(name)
	if err != nil {
		return false
	}
	return fi.IsDir()
}
