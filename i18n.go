// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/i18n source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

// Package i18n is internationalization and localization support for aah
// framework. Messages config format is `forge` config syntax (go-aah/config)
// which is similar to HOCON syntax aka typesafe config.
//
// Message filename format is `message.<Language-ID>`. Language is combination
// of Language + Region value. aah framework implements Language code is as per
// two-letter `ISO 639-1` standard and Region code is as per two-letter
// `ISO 3166-1` standard.
//
// Supported message file extension formats are (incasesensitive)
// 	* Priority 1: Language + Region => en-us | en-US
// 	* Priority 2: Language          => en
//
// 	For Example:
// 		message.en-US or message.en-us
// 		message.en-GB or message.en-gb
// 		message.en-CA or message.en-ca
// 		message.en
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

	"aahframework.org/ahttp.v0-unstable"
	"aahframework.org/config.v0-unstable"
	"aahframework.org/essentials.v0-unstable"
	"aahframework.org/log.v0-unstable"
)

// Version no. of aah framework i18n library
const Version = "0.1"

// I18n holds the message store and related information for internationalization
// and localization.
type I18n struct {
	Store        map[string]*config.Config
	fileExtRegex string
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Global methods
//___________________________________

// New method creates aah i18n message store
func New() *I18n {
	return &I18n{
		Store:        make(map[string]*config.Config, 0),
		fileExtRegex: `messages\.[a-z]{2}(\-[a-zA-Z]{2})?$`,
	}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// I18n methods
//___________________________________

// Load processes the given message file or directory and adds to the
// message store
func (s *I18n) Load(paths ...string) error {
	for _, path := range paths {
		if !ess.IsFileExists(path) {
			log.Warnf("Path: %v not exists, let's move on", path)
			continue
		}

		if ess.IsDir(path) {
			_ = ess.Walk(path, func(fpath string, f os.FileInfo, _ error) error {
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
// 	* language and language-id (e.g.: en-US)
// 	* language (e.g.: en)
func (s *I18n) Lookup(locale *ahttp.Locale, key string, args ...interface{}) string {
	// Lookup by language and language-id. For eg.: en-us
	store := s.findStoreByLocale(locale.String())
	if store == nil {
		// Lookup by language. for eg.: en
		store = s.findStoreByLocale(locale.Language)
		if store == nil {
			log.Warnf("Locale (%v, %v) doesn't exists in message store", locale.String(), locale.Language)
			return ""
		}

		log.Tracef("Message is retrieved from locale: %v", locale.Language)
		if msg, found := retriveValue(store, key, args...); found {
			return msg
		}
	}

	log.Tracef("Message is retrieved from locale: %v", locale.String())
	if msg, found := retriveValue(store, key, args...); found {
		return msg
	}

	log.Tracef("Message is retrieved from locale: %v", locale.Language)
	if msg, found := retriveValue(s.findStoreByLocale(locale.Language), key, args...); found {
		return msg
	}

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

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// I18n Unexported methods
//___________________________________

func (s *I18n) processMsgFile(file string) {
	key := strings.ToLower(filepath.Ext(file)[1:])
	msgFile, err := config.LoadFile(file)
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

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

func retriveValue(store *config.Config, key string, args ...interface{}) (string, bool) {
	if msg, found := store.String(key); found {
		if len(args) > 0 {
			return fmt.Sprintf(msg, args...), found
		}
		return msg, found
	}
	return "", false
}
