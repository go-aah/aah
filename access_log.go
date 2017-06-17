// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"bytes"
	"fmt"
	"net/http"
	"path/filepath"
	"sync"
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

	defaultRequestAccessLogPattern = "%clientip %reqid %reqtime %restime %resstatus %ressize %reqmethod %requrl"
	appStartTimeKey                = "appReqStartTime"
	appAccessLogBufPool            *sync.Pool
	appAccessLog                   *log.Logger
	appAccessLogFmtFlags           []ess.FmtFlagPart
	appAccessLogChan               chan *requestAccessLog
)

type (
	//requestAccessLog contains data about the current request
	requestAccessLog struct {
		StartTime       time.Time
		ElapsedDuration time.Duration
		RequestID       string
		Request         ahttp.Request
		ResStatus       int
		ResBytes        int
		ResHdr          http.Header
	}
)

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
	pattern := appCfg.StringDefault("request.access_log.pattern", defaultRequestAccessLogPattern)
	appAccessLogFmtFlags, err = ess.ParseFmtFlag(pattern, accessLogFmtFlags)
	if err != nil {
		return err
	}

	// initialize request access log channel
	appAccessLogChan = make(chan *requestAccessLog, cfg.IntDefault("runtime.pooling.global", defaultGlobalPoolSize))
	appAccessLogBufPool = &sync.Pool{New: func() interface{} { return &bytes.Buffer{} }}

	// start the listener
	go listenForAccessLog()

	return nil
}

func listenForAccessLog() {
	for {
		info := <-appAccessLogChan
		appAccessLog.Print(requestAccessLogFormatter(info))
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
			//there are two options here for the pattern we have to handle ;
			//%reqtime or %reqtime:
			if ess.IsStrEmpty(part.Format) || part.Format == "%v" {
				buf.WriteString(ral.StartTime.Format(time.RFC3339))
			} else {
				buf.WriteString(ral.StartTime.Format(part.Format))
			}
		case fmtFlagRequestURL:
			buf.WriteString(ral.Request.Path)
		case fmtFlagRequestMethod:
			buf.WriteString(ral.Request.Method)
		case fmtFlagRequestID:
			if ess.IsStrEmpty(ral.RequestID) {
				buf.WriteByte('-')
			} else {
				buf.WriteString(ral.RequestID)
			}
		case fmtFlagRequestHeader:
			hdr := ral.Request.Header.Get(http.CanonicalHeaderKey(part.Format))
			if ess.IsStrEmpty(hdr) {
				buf.WriteByte('-')
			} else {
				buf.WriteString(hdr)
			}
		case fmtFlagQueryString:
			queryStr := ral.Request.Raw.URL.Query().Encode()
			if ess.IsStrEmpty(queryStr) {
				buf.WriteByte('-')
			} else {
				buf.WriteString(queryStr)
			}
		case fmtFlagResponseStatus:
			buf.WriteString(fmt.Sprintf(part.Format, ral.ResStatus))
		case fmtFlagResponseSize:
			buf.WriteString(fmt.Sprintf(part.Format, ral.ResBytes))
		case fmtFlagResponseHeader:
			hdr := ral.ResHdr.Get(part.Format)
			if ess.IsStrEmpty(hdr) {
				buf.WriteByte('-')
			} else {
				buf.WriteString(hdr)
			}
		case fmtFlagResponseTime:
			buf.WriteString(fmt.Sprintf("%10v", ral.ElapsedDuration))
		}
		buf.WriteByte(' ')
	}
	return buf.String()
}

func getBuffer() *bytes.Buffer {
	return appAccessLogBufPool.Get().(*bytes.Buffer)
}

func putBuffer(buf *bytes.Buffer) {
	buf.Reset()
	appAccessLogBufPool.Put(buf)
}
