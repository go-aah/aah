// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
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

	"aahframework.org/aah/ahttp"
	"aahframework.org/config"
	"aahframework.org/essentials"
	"aahframework.org/log"
)

var (
	msgFileExtRegex = `messages\.[a-z]{2}(\-[a-zA-Z]{2})?$`
	messageStore    = make(map[string]*config.Config, 0)
)

// Load processes the given message file or directory and adds to the
// message store
func Load(paths ...string) error {
	for _, path := range paths {
		if !ess.IsFileExists(path) {
			log.Warnf("Path: %v not exists, let's move on", path)
			continue
		}

		if ess.IsDir(path) {
			_ = ess.Walk(path, func(fpath string, f os.FileInfo, _ error) error {
				if !f.IsDir() {
					match, err := regexp.MatchString(msgFileExtRegex, f.Name())
					if err == nil && match {
						processMsgFile(fpath)
					}
				}
				return nil
			})
		} else { // if it's a file
			processMsgFile(path)
		}
	}

	return nil
}

// Msg returns message value by given locale and it supports formatting a message
// before its return. If given message key or store doesn't exists for given locale;
// Msg method returns empty string.
// 	Msg(locale, "i.love.aah.framework", "yes")
// The sequence and fallback order of message fetch from store is -
// 	* language and language-id (e.g.: en-US)
// 	* language (e.g.: en)
func Msg(locale *ahttp.Locale, key string, args ...interface{}) string {
	// Lookup by language and language-id. For eg.: en-us
	store := findMsgStoreByLocale(locale.String())
	if store == nil {
		// Lookup by language. for eg.: en
		store = findMsgStoreByLocale(locale.Language)
		if store == nil {
			log.Warnf("Locale (%v, %v) doesn't exists in message store", locale.String(), locale.Language)
			return ""
		}

		log.Tracef("Message is retrieved from locale: %v", locale.Language)
		return retriveMsg(store, key, args...)
	}

	log.Tracef("Message is retrieved from locale: %v", locale.String())
	msg := retriveMsg(store, key, args...)
	if ess.IsStrEmpty(msg) {
		// If return value is empty then lookup by language. for eg.: en
		return retriveMsg(findMsgStoreByLocale(locale.Language), key, args...)
	}

	return msg
}

// Locales returns all the loaded locales from message store
func Locales() []string {
	var locales []string
	for l := range messageStore {
		locales = append(locales, l)
	}
	return locales
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

func processMsgFile(file string) {
	key := strings.ToLower(filepath.Ext(file)[1:])
	msgFile, err := config.LoadFile(file)
	if err != nil {
		log.Errorf("Unable to load message file: %v, error: %v", file, err)
	}

	// merge messages if key is already exists otherwise add it
	if ms, exists := messageStore[key]; exists {
		log.Tracef("Key[%v] is already exists, let's merge it", key)
		if err = ms.Merge(msgFile); err != nil {
			log.Errorf("Error while merging message file: %v", file)
		}
	} else {
		log.Tracef("Adding to message store [%v: %v]", key, file)
		messageStore[key] = msgFile
	}
}

func findMsgStoreByLocale(locale string) *config.Config {
	if store, exists := messageStore[strings.ToLower(locale)]; exists {
		return store
	}
	return nil
}

func retriveMsg(store *config.Config, key string, args ...interface{}) string {
	if msg, found := store.String(key); found {
		if len(args) > 0 {
			return fmt.Sprintf(msg, args...)
		}
		return msg
	}
	return ""
}
