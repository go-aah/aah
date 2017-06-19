// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0-unstable"
	"aahframework.org/log.v0"
)

const (
	fmtFlagClientIP ess.FmtFlag = iota
	fmtFlagRequestTime
	fmtFlagRequestURL
	fmtFlagRequestMethod
	fmtFlagRequestID
	fmtFlagRequestHeader
	fmtFlagQueryString
	fmtFlagResponseStatus
	fmtFlagResponseSize
	fmtFlagResponseHeader
	fmtFlagResponseTime
)

var (
	accessLogFmtFlags = map[string]ess.FmtFlag{
		"clientip":  fmtFlagClientIP,
		"reqtime":   fmtFlagRequestTime,
		"requrl":    fmtFlagRequestURL,
		"reqmethod": fmtFlagRequestMethod,
		"reqid":     fmtFlagRequestID,
		"reqhdr":    fmtFlagRequestHeader,
		"querystr":  fmtFlagQueryString,
		"resstatus": fmtFlagResponseStatus,
		"ressize":   fmtFlagResponseSize,
		"reshdr":    fmtFlagResponseHeader,
		"restime":   fmtFlagResponseTime,
	}

	appDefaultAccessLogPattern = "%clientip %reqid %reqtime %restime %resstatus %ressize %reqmethod %requrl"
	appReqStartTimeKey         = "appReqStartTimeKey"
	appReqIDHdrKey             = ahttp.HeaderXRequestID
	appAccessLog               *log.Logger
	appAccessLogFmtFlags       []ess.FmtFlagPart
	appAccessLogChan           chan *requestAccessLog
)

type (
	//requestAccessLog contains data about the current request
	requestAccessLog struct {
		StartTime       time.Time
		ElapsedDuration time.Duration
		Request         ahttp.Request
		ResStatus       int
		ResBytes        int
		ResHdr          http.Header
	}
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// requestAccessLog methods
//___________________________________

// FmtRequestTime method returns the formatted request time. There are three
// possibilities to handle, `%reqtime`, `%reqtime:` and `%reqtime:<format>`.
func (al *requestAccessLog) FmtRequestTime(format string) string {
	if format == "%v" || ess.IsStrEmpty(format) {
		return al.StartTime.Format(time.RFC3339)
	}
	return al.StartTime.Format(format)
}

func (al *requestAccessLog) GetRequestHdr(hdrKey string) string {
	hdrValues := al.Request.Header[http.CanonicalHeaderKey(hdrKey)]
	if len(hdrValues) == 0 {
		return "-"
	}
	return strings.Join(hdrValues, ", ")
}

func (al *requestAccessLog) GetResponseHdr(hdrKey string) string {
	hdrValues := al.ResHdr[http.CanonicalHeaderKey(hdrKey)]
	if len(hdrValues) == 0 {
		return "-"
	}
	return strings.Join(hdrValues, ", ")
}

func (al *requestAccessLog) GetQueryString() string {
	queryStr := al.Request.Raw.URL.Query().Encode()
	if ess.IsStrEmpty(queryStr) {
		return "-"
	}
	return queryStr
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

func initRequestAccessLog(logsDir string, appCfg *config.Config) error {
	// log file configuration
	cfg, _ := config.ParseString("")
	file := appCfg.StringDefault("request.access_log.file", "")
	if ess.IsStrEmpty(file) {
		cfg.SetString("log.file", filepath.Join(logsDir, getBinaryFileName()+"-access.log"))
	} else if !filepath.IsAbs(file) {
		cfg.SetString("log.file", filepath.Join(logsDir, file))
	} else {
		cfg.SetString("log.file", file)
	}

	cfg.SetString("log.pattern", "%message")

	var err error

	// initialize request access log file
	appAccessLog, err = log.New(cfg)
	if err != nil {
		return err
	}

	// parse request access log pattern
	pattern := appCfg.StringDefault("request.access_log.pattern", appDefaultAccessLogPattern)
	appAccessLogFmtFlags, err = ess.ParseFmtFlag(pattern, accessLogFmtFlags)
	if err != nil {
		return err
	}

	// initialize request access log channel
	if appAccessLogChan == nil {
		appAccessLogChan = make(chan *requestAccessLog, cfg.IntDefault("runtime.pooling.global", defaultGlobalPoolSize))
		go listenForAccessLog()
	}

	appReqIDHdrKey = cfg.StringDefault("request.id.header", ahttp.HeaderXRequestID)

	return nil
}

func listenForAccessLog() {
	for {
		appAccessLog.Print(requestAccessLogFormatter(<-appAccessLogChan))
	}
}

func requestAccessLogFormatter(ral *requestAccessLog) string {
	buf := getBuffer()
	defer putBuffer(buf)

	for _, part := range appAccessLogFmtFlags {
		switch part.Flag {
		case fmtFlagClientIP:
			buf.WriteString(ral.Request.ClientIP)
		case fmtFlagRequestTime:
			buf.WriteString(ral.FmtRequestTime(part.Format))
		case fmtFlagRequestURL:
			buf.WriteString(ral.Request.Path)
		case fmtFlagRequestMethod:
			buf.WriteString(ral.Request.Method)
		case fmtFlagRequestID:
			buf.WriteString(ral.GetRequestHdr(appReqIDHdrKey))
		case fmtFlagRequestHeader:
			buf.WriteString(ral.GetRequestHdr(part.Format))
		case fmtFlagQueryString:
			buf.WriteString(ral.GetQueryString())
		case fmtFlagResponseStatus:
			buf.WriteString(fmt.Sprintf(part.Format, ral.ResStatus))
		case fmtFlagResponseSize:
			buf.WriteString(fmt.Sprintf(part.Format, ral.ResBytes))
		case fmtFlagResponseHeader:
			buf.WriteString(ral.GetResponseHdr(part.Format))
		case fmtFlagResponseTime:
			buf.WriteString(fmt.Sprintf(part.Format, ral.ElapsedDuration.Nanoseconds()))
		}
		buf.WriteByte(' ')
	}
	return strings.TrimSpace(buf.String())
}
