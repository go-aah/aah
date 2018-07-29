// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package session

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
	"aahframework.org/security.v0/cookie"
)

// Storer interface comply
var _ Storer = (*FileStore)(nil)

// FileStore is the aah framework session store implementation.
type FileStore struct {
	path       string
	filePrefix string
	m          sync.RWMutex
}

// Init method initialize the file store using given application config.
func (f *FileStore) Init(cfg *config.Config) error {
	storePath, found := cfg.String("security.session.store.filepath")
	if !found {
		return errors.New("session: file store storage path is not provided")
	}
	f.path = storePath

	// resolve relative path to absolute path
	if !filepath.IsAbs(f.path) {
		absPath, err := filepath.Abs(f.path)
		if err != nil {
			return err
		}
		f.path = absPath
	}

	// create a directory if not exists
	if !ess.IsFileExists(f.path) {
		if err := ess.MkDirAll(f.path, 0755); err != nil {
			return err
		}
	}

	// session file prefix
	f.filePrefix = cfg.StringDefault("security.session.prefix", "aah") + "_session"
	f.m = sync.RWMutex{}

	log.Infof("Session file store is initialized at path: %v", filepath.FromSlash(f.path))
	return nil
}

// Read method reads the encoded cookie value from file.
func (f *FileStore) Read(id string) string {
	f.m.RLock()
	defer f.m.RUnlock()
	sessionFile := filepath.Join(f.path, f.filePrefix+"_"+id)
	sdata, err := ioutil.ReadFile(sessionFile)
	if err != nil {
		log.Errorf("session: file store - read error: %v", err)
		return ""
	}

	return string(sdata)
}

// Save method saves the given session id with encoded cookie value.
func (f *FileStore) Save(id, value string) error {
	f.m.Lock()
	defer f.m.Unlock()
	sessionFile := filepath.Join(f.path, f.filePrefix+"_"+id)
	return ioutil.WriteFile(sessionFile, []byte(value), 0600)
}

// Delete method deletes the session file for given id.
func (f *FileStore) Delete(id string) error {
	f.m.Lock()
	defer f.m.Unlock()
	sessionFile := filepath.Join(f.path, f.filePrefix+"_"+id)
	if err := os.Remove(sessionFile); !os.IsNotExist(err) {
		return err
	}
	return nil
}

// IsExists method returns true if the session file exists otherwise false.
func (f *FileStore) IsExists(id string) bool {
	return ess.IsFileExists(filepath.Join(f.path, f.filePrefix+"_"+id))
}

// Cleanup method deletes the expired session file.
func (f *FileStore) Cleanup(m *Manager) {
	files, err := ess.FilesPath(f.path, false)
	if err != nil {
		log.Error(err)
		return
	}

	log.Infof("%v session files found", len(files))
	cnt := 0
	for _, sfile := range files {
		if sdata, err := ioutil.ReadFile(sfile); err == nil {
			if _, err := m.DecodeToSession(string(sdata)); err == cookie.ErrCookieTimestampIsExpired {
				f.m.Lock()
				if err := os.Remove(sfile); !os.IsNotExist(err) {
					log.Error(err)
				} else {
					cnt++
				}
				f.m.Unlock()
			}
		}
	}

	log.Infof("%v expired session files cleaned up", cnt)
}

func init() {
	_ = AddStore("file", &FileStore{})
}
