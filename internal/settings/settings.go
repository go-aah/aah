// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package settings

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"aahframe.work/ahttp"
	"aahframe.work/config"
	"aahframe.work/essentials"
	"aahframe.work/internal/util"
	"aahframe.work/log"
	"golang.org/x/crypto/acme/autocert"
)

// Constants
const (
	DefaultEnvProfile       = "dev"
	DefaultHTTPPort         = "8080"
	DefaultSecureJSONPrefix = ")]}',\n"
	ProfilePrefix           = "env."
)

// Settings represents parsed and inferred config values for the application.
type Settings struct {
	PhysicalPathMode       bool
	PackagedMode           bool
	ServerHeaderEnabled    bool
	RequestIDEnabled       bool
	SSLEnabled             bool
	LetsEncryptEnabled     bool
	GzipEnabled            bool
	SecureHeadersEnabled   bool
	AccessLogEnabled       bool
	StaticAccessLogEnabled bool
	DumpLogEnabled         bool
	Initialized            bool
	HotReload              bool
	HotReloadEnabled       bool
	AuthSchemeExists       bool
	Redirect               bool
	Pid                    int
	HTTPMaxHdrBytes        int
	ImportPath             string
	BaseDir                string
	VirtualBaseDir         string
	Type                   string
	EnvProfile             string
	SSLCert                string
	SSLKey                 string
	ServerHeader           string
	RequestIDHeaderKey     string
	SecureJSONPrefix       string
	ShutdownGraceTimeStr   string
	DefaultContentType     string
	HotReloadSignalStr     string
	HTTPReadTimeout        time.Duration
	HTTPWriteTimeout       time.Duration
	ShutdownGraceTimeout   time.Duration
	Autocert               *autocert.Manager

	cfg *config.Config
}

// Refresh method to parse/infer config values and populate settings instance.
func (s *Settings) Refresh(cfg *config.Config) error {
	s.cfg = cfg

	var err error
	if err = s.setEnvProfile(s.cfg.StringDefault("env.active", DefaultEnvProfile)); err != nil {
		return err
	}
	s.SSLEnabled = s.cfg.BoolDefault("server.ssl.enable", false)
	s.LetsEncryptEnabled = s.cfg.BoolDefault("server.ssl.lets_encrypt.enable", false)
	s.Redirect = s.cfg.BoolDefault("server.redirect.enable", false)

	readTimeout := s.cfg.StringDefault("server.timeout.read", "90s")
	writeTimeout := s.cfg.StringDefault("server.timeout.write", "90s")
	if !util.IsValidTimeUnit(readTimeout, "s", "m") || !util.IsValidTimeUnit(writeTimeout, "s", "m") {
		return errors.New("'server.timeout.{read|write}' value is not a valid time unit")
	}

	if s.HTTPReadTimeout, err = time.ParseDuration(readTimeout); err != nil {
		return fmt.Errorf("'server.timeout.read': %s", err)
	}

	if s.HTTPWriteTimeout, err = time.ParseDuration(writeTimeout); err != nil {
		return fmt.Errorf("'server.timeout.write': %s", err)
	}

	maxHdrBytesStr := s.cfg.StringDefault("server.max_header_bytes", "1mb")
	if maxHdrBytes, er := ess.StrToBytes(maxHdrBytesStr); er == nil {
		s.HTTPMaxHdrBytes = int(maxHdrBytes)
	} else {
		return errors.New("'server.max_header_bytes' value is not a valid size unit")
	}

	s.SSLCert = s.cfg.StringDefault("server.ssl.cert", "")
	s.SSLKey = s.cfg.StringDefault("server.ssl.key", "")
	if err = s.checkSSLConfigValues(); err != nil {
		return err
	}
	if s.SSLEnabled && s.LetsEncryptEnabled {
		cfgKeyPrefix := "server.ssl.lets_encrypt"
		hostPolicy, found := s.cfg.StringList(cfgKeyPrefix + ".host_policy")
		if !found || len(hostPolicy) == 0 {
			return errors.New("'server.ssl.lets_encrypt.host_policy' is empty, provide at least one hostname")
		}

		renewBefore := time.Duration(s.cfg.IntDefault(cfgKeyPrefix+".renew_before", 10))
		s.Autocert = &autocert.Manager{
			Prompt:      autocert.AcceptTOS,
			HostPolicy:  autocert.HostWhitelist(hostPolicy...),
			RenewBefore: time.Hour * (24 * renewBefore),
			Email:       s.cfg.StringDefault(cfgKeyPrefix+".email", ""),
		}

		cacheDir := s.cfg.StringDefault(cfgKeyPrefix+".cache_dir", filepath.Join(s.BaseDir, "autocert"))
		s.Autocert.Cache = autocert.DirCache(cacheDir)
	}

	s.Type = s.cfg.StringDefault("type", "")
	if s.Type != "websocket" {
		if _, err = ess.StrToBytes(s.cfg.StringDefault("request.max_body_size", "5mb")); err != nil {
			return errors.New("'request.max_body_size' value is not a valid size unit")
		}

		s.ServerHeader = s.cfg.StringDefault("server.header", "")
		s.ServerHeaderEnabled = !ess.IsStrEmpty(s.ServerHeader)
		s.RequestIDEnabled = s.cfg.BoolDefault("request.id.enable", true)
		s.RequestIDHeaderKey = s.cfg.StringDefault("request.id.header", ahttp.HeaderXRequestID)
		s.SecureHeadersEnabled = s.cfg.BoolDefault("security.http_header.enable", true)
		s.GzipEnabled = s.cfg.BoolDefault("render.gzip.enable", true)
		s.AccessLogEnabled = s.cfg.BoolDefault("server.access_log.enable", false)
		s.StaticAccessLogEnabled = s.cfg.BoolDefault("server.access_log.static_file", true)
		s.DumpLogEnabled = s.cfg.BoolDefault("server.dump_log.enable", false)
		if rd := s.cfg.StringDefault("render.default", ""); len(rd) > 0 {
			s.DefaultContentType = util.MimeTypeByExtension("some." + rd)
		}

		s.SecureJSONPrefix = s.cfg.StringDefault("render.secure_json.prefix", DefaultSecureJSONPrefix)

		ahttp.GzipLevel = s.cfg.IntDefault("render.gzip.level", 4)
		if !(ahttp.GzipLevel >= 1 && ahttp.GzipLevel <= 9) {
			return fmt.Errorf("'render.gzip.level' is not a valid level value: %v", ahttp.GzipLevel)
		}
	}

	s.HotReloadEnabled = s.cfg.BoolDefault("runtime.config_hotreload.enable", true)
	s.HotReloadSignalStr = strings.ToUpper(s.cfg.StringDefault("runtime.config_hotreload.signal", "SIGHUP"))

	s.ShutdownGraceTimeStr = s.cfg.StringDefault("server.timeout.grace_shutdown", "60s")
	if !util.IsValidTimeUnit(s.ShutdownGraceTimeStr, "s", "m") {
		log.Warn("'server.timeout.grace_shutdown' value is not a valid time unit, assigning default value 60s")
		s.ShutdownGraceTimeStr = "60s"
	}
	s.ShutdownGraceTimeout, _ = time.ParseDuration(s.ShutdownGraceTimeStr)

	return nil
}

// SetImportPath method process import path and sets it into settings instance.
func (s *Settings) SetImportPath(args []string) {
	for i, arg := range args {
		if arg == "--importpath" {
			s.ImportPath = args[i+1]
			break
		}
	}
}

// SetEnvProfile method is to set application environment profile value.
func (s *Settings) setEnvProfile(p string) error {
	if !strings.HasPrefix(p, ProfilePrefix) {
		p = ProfilePrefix + p
	}
	if err := s.cfg.SetProfile(p); err != nil {
		return err
	}
	s.EnvProfile = strings.TrimPrefix(p, ProfilePrefix)
	return nil
}

func (s *Settings) checkSSLConfigValues() error {
	if s.SSLEnabled {
		if !s.LetsEncryptEnabled && (ess.IsStrEmpty(s.SSLCert) || ess.IsStrEmpty(s.SSLKey)) {
			return errors.New("SSL config is incomplete; either enable 'server.ssl.lets_encrypt.enable' or provide 'server.ssl.cert' & 'server.ssl.key' value")
		} else if !s.LetsEncryptEnabled {
			if !ess.IsFileExists(s.SSLCert) {
				return fmt.Errorf("SSL cert file not found: %s", s.SSLCert)
			}

			if !ess.IsFileExists(s.SSLKey) {
				return fmt.Errorf("SSL key file not found: %s", s.SSLKey)
			}
		}
	}

	if s.LetsEncryptEnabled && !s.SSLEnabled {
		return errors.New("let's encrypt enabled, however SSL 'server.ssl.enable' is not enabled for application")
	}
	return nil
}
