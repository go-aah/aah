// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
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
	if isSSLEnabled {
		if !isLetsEncrypt && (ess.IsStrEmpty(sslCert) || ess.IsStrEmpty(sslKey)) {
			return errors.New("SSL config is incomplete; either enable 'server.ssl.lets_encrypt.enable' or provide 'server.ssl.cert' & 'server.ssl.key' value")
		} else if !isLetsEncrypt {
			if !ess.IsFileExists(sslCert) {
				return fmt.Errorf("SSL cert file not found: %s", sslCert)
			}

			if !ess.IsFileExists(sslKey) {
				return fmt.Errorf("SSL key file not found: %s", sslKey)
			}
		}
	}

	if isLetsEncrypt && !isSSLEnabled {
		return errors.New("let's encrypt enabled, however SSL 'server.ssl.enable' is not enabled for application")
	}
	return nil
}

func writePID(appBinaryName, appBaseDir string) {
	appPID = os.Getpid()
	pidfile := filepath.Join(appBaseDir, appBinaryName+".pid")
	if err := ioutil.WriteFile(pidfile, []byte(strconv.Itoa(appPID)), 0644); err != nil {
		log.Error(err)
	}
}

func getBinaryFileName() string {
	return ess.StripExt(AppBuildInfo().BinaryName)
}

func isNoGzipStatusCode(code int) bool {
	for _, c := range noGzipStatusCodes {
		if c == code {
			return true
		}
	}
	return false
}
