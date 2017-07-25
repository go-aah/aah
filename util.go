// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"aahframework.org/ahttp.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
)

var (
	charsetMap = map[string]string{
		"text/html":        "charset=utf-8",
		"application/json": "charset=utf-8",
		"application/xml":  "charset=utf-8",
		"text/json":        "charset=utf-8",
		"text/xml":         "charset=utf-8",
		"text/plain":       "charset=utf-8",
	}
)

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

// This method is similar to
// https://golang.org/src/net/http/transfer.go#bodyAllowedForStatus
func isResponseBodyAllowed(code int) bool {
	if (code >= http.StatusContinue && code < http.StatusOK) ||
		code == http.StatusNoContent || code == http.StatusNotModified {
		return false
	}
	return true
}

func resolveControllerName(ctx *Context) string {
	if ess.IsStrEmpty(ctx.controller.Namespace) {
		return ctx.controller.Name()
	}
	return path.Join(ctx.controller.Namespace, ctx.controller.Name())
}

func isCharsetExists(value string) bool {
	return strings.Contains(value, "charset")
}

// this method is candidate for essentials library
// move it when you get a time
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if !ess.IsStrEmpty(v) {
			return v
		}
	}
	return ""
}

func identifyContentType(ctx *Context) *ahttp.ContentType {
	// based on 'Accept' Header
	if !ess.IsStrEmpty(ctx.Req.AcceptContentType.Mime) &&
		ctx.Req.AcceptContentType.Mime != "*/*" {
		return ctx.Req.AcceptContentType
	}

	// as per 'render.default' in aah.conf or nil
	return defaultContentType()
}
