// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

// Package i18n is internationalization and localization support for aah
// framework. Messages store config format is same as aah configuration.
// Refer to https://docs.aahframework.org/configuration.html.
//
// Message filename format is `message.<Language-ID>`. Language ID is combination
// of `Language + Region` or `Language` value. aah framework implements Language
// code is as per two-letter `ISO 639-1` standard and Region code is as per two-letter
// `ISO 3166-1` standard.
//
// Supported message file extension formats are (incasesensitive)
//
// 	1) Language + Region => en-us | en-US
//
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
	"sync"

	"aahframe.work/ahttp"
	"aahframe.work/config"
	"aahframe.work/log"
	"aahframe.work/vfs"
)

// I18ner interface is used to implement i18n message store.
type I18ner interface {
	Lookup(locale *ahttp.Locale, key string, args ...interface{}) string
	DefaultLocale() string
	Locales() []string
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package methods
//______________________________________________________________________________

// New method creates aah i18n message store with given options.
func New(l log.Loggerer, opts ...Option) *I18n {
	msgStore := &I18n{
		RWMutex:       sync.RWMutex{},
		store:         make(map[string]*config.Config),
		fileExtRegex:  regexp.MustCompile(`messages\.[a-z]{2}(\-[a-zA-Z]{2})?$`),
		defaultLocale: "en",
		files:         make([]string, 0),
		log:           l,
	}
	for _, opt := range opts {
		opt(msgStore)
	}
	return msgStore
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// I18n options type and methods
//______________________________________________________________________________

// Option type to provide configuration options
// to create i18n message store.
type Option func(*I18n)

// DefaultLocale option func is to message store default locale.
func DefaultLocale(locale string) Option {
	return func(i *I18n) {
		i.Lock()
		defer i.Unlock()
		i.defaultLocale = locale
	}
}

// Dirs option func is to set aah VFS instance.
func VFS(fs vfs.FileSystem) Option {
	return func(i *I18n) {
		i.Lock()
		defer i.Unlock()
		i.fs = fs
	}
}

// Dirs option func is to supply n no. of directory path.
func Dirs(dirs ...string) Option {
	return func(i *I18n) {
		i.Lock()
		defer i.Unlock()
		for _, d := range dirs {
			if !vfs.IsDir(i.fs, d) {
				i.log.Warnf("i18n: %v not exists or error, let's move on", d)
				continue
			}
			_ = vfs.Walk(i.fs, d, func(fpath string, fi os.FileInfo, _ error) error {
				if !fi.IsDir() {
					if i.fileExtRegex.MatchString(fi.Name()) {
						i.files = append(i.files, fpath)
					}
				}
				return nil
			})
		}
	}
}

// Files option func is to supply n no. of file path.
func Files(files ...string) Option {
	return func(i *I18n) {
		i.Lock()
		defer i.Unlock()
		for _, f := range files {
			if !vfs.IsExists(i.fs, f) {
				i.log.Warnf("i18n: %v not exists, let's move on", f)
				continue
			}
			i.files = append(i.files, f)
		}
	}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// I18n methods
//______________________________________________________________________________

// I18n holds the message store and related information for internationalization
// and localization.
type I18n struct {
	sync.RWMutex
	store         map[string]*config.Config
	defaultLocale string
	files         []string
	fileExtRegex  *regexp.Regexp
	fs            vfs.FileSystem
	log           log.Loggerer
}

// interface check
var _ I18ner = (*I18n)(nil)

// Lookup returns value by given key, locale and it supports formatting a message
// before its return. If given message key or store doesn't exists for given locale;
// Lookup method returns empty string.
// 	Lookup(locale, "i.love.aah.framework", "yes")
// The sequence and fallback order of message fetch from store is -
// 	* language and region-id (e.g.: en-US)
// 	* language (e.g.: en)
func (s *I18n) Lookup(locale *ahttp.Locale, key string, args ...interface{}) string {
	s.RLock()
	defer s.RUnlock()
	// assign default locale if nil
	if locale == nil {
		locale = ahttp.NewLocale(s.defaultLocale)
	}

	// Lookup by language and region-id. For eg.: en-us
	store := s.findStoreByLocale(locale.String())
	if store == nil {
		goto langStore
	}
	if msg, found := retriveValue(store, key, args...); found {
		s.log.Tracef("i18n message is retrieved from locale: %v, key: %v", locale, key)
		return msg
	}

langStore:
	// Lookup by language. For eg.: en
	store = s.findStoreByLocale(locale.Language)
	if store == nil {
		goto defaultStore
	}
	if msg, found := retriveValue(store, key, args...); found {
		s.log.Tracef("i18n message is retrieved from locale: %v, key: %v", locale.Language, key)
		return msg
	}

defaultStore: // fallback to `i18n.default` config value.
	store = s.findStoreByLocale(s.defaultLocale)
	if store == nil {
		goto notExists
	}
	if msg, found := retriveValue(store, key, args...); found {
		s.log.Tracef("i18n message is retrieved from locale: %v, key: %v", s.defaultLocale, key)
		return msg
	}

notExists:
	return key
}

// DefaultLocale method returns the i18n store's default locale.
func (s *I18n) DefaultLocale() string {
	s.RLock()
	defer s.RUnlock()
	return s.defaultLocale
}

// Locales returns all the loaded locales from message store
func (s *I18n) Locales() []string {
	s.RLock()
	defer s.RUnlock()
	var locales []string
	for l := range s.store {
		locales = append(locales, l)
	}
	return locales
}

// Load method loads message files into message store.
// Returns error for any failures.
func (s *I18n) Init() error {
	for _, f := range s.files {
		if err := s.add2Store(f); err != nil {
			return err
		}
	}
	return nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// I18n Unexported methods
//______________________________________________________________________________

func (s *I18n) add2Store(file string) error {
	key := strings.ToLower(filepath.Ext(file)[1:])
	s.log.Tracef("Adding into i18n message store [%v: %v]", key, file)
	msgFile, err := config.LoadFile(file)
	if err != nil {
		return fmt.Errorf("i18n: unable to process message file: %v, error: %v", file, err)
	}

	// merge messages if key is already exists otherwise add it
	s.Lock()
	defer s.Unlock()
	if ms, exists := s.store[key]; exists {
		if err = ms.Merge(msgFile); err != nil {
			return fmt.Errorf("i18n: error while adding into message store file: %v", file)
		}
	} else {
		s.store[key] = msgFile
	}
	return nil
}

func (s *I18n) findStoreByLocale(locale string) *config.Config {
	if store, exists := s.store[strings.ToLower(locale)]; exists {
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
	return key, false
}
