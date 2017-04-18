// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
)

func isBinDirExists() bool {
	return ess.IsFileExists(filepath.Join(getWorkingDir(), "bin"))
}

func isAppDirExists() bool {
	return ess.IsFileExists(filepath.Join(getWorkingDir(), "app"))
}

func getWorkingDir() string {
	wd, _ := os.Getwd()
	return wd
}

func isValidTimeUnit(str string, units ...string) bool {
	for _, v := range units {
		if strings.HasSuffix(str, v) {
			return true
		}
	}
	return false
}

func checkSSLConfigValues(isSSLEnabled, isLetsEncrypt bool, sslCert, sslKey string) error {
	log.Debugf("SSLCert: %v, SSLKey: %v", sslCert, sslKey)

	if isSSLEnabled {
		if !isLetsEncrypt && (ess.IsStrEmpty(sslCert) || ess.IsStrEmpty(sslKey)) {
			return errors.New("SSL config is incomplete; either enable 'server.ssl.lets_encrypt.enable' or provide 'server.ssl.cert' & 'server.ssl.key' value")
		}
	}

	if isLetsEncrypt && !isSSLEnabled {
		return errors.New("let's encrypt enabled, however SSL 'server.ssl.enable' is not enabled for application")
	}
	return nil
}
