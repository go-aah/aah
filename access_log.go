// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
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

	appDefaultAccessLogPattern = "%clientip %reqtime %restime %resstatus %ressize %reqmethod %requrl"
	appReqStartTimeKey         = "_appReqStartTimeKey"
	appReqIDHdrKey             = ahttp.HeaderXRequestID
	appAccessLog               *log.Logger
	appAccessLogFmtFlags       []ess.FmtFlagPart
	appAccessLogChan           chan *accessLog
	accessLogPool              = &sync.Pool{New: func() interface{} { return &accessLog{} }}
)

type (
	//accessLog contains data about the current request
	accessLog struct {
		StartTime       time.Time
		ElapsedDuration time.Duration
		Request         *ahttp.Request
		ResStatus       int
		ResBytes        int
		ResHdr          http.Header
	}
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// accessLog methods
//___________________________________

// FmtRequestTime method returns the formatted request time. There are three
// possibilities to handle, `%reqtime`, `%reqtime:` and `%reqtime:<format>`.
func (al *accessLog) FmtRequestTime(format string) string {
	if format == "%v" || ess.IsStrEmpty(format) {
		return al.StartTime.Format(time.RFC3339)
	}
	return al.StartTime.Format(format)
}

func (al *accessLog) GetRequestHdr(hdrKey string) string {
	hdrValues := al.Request.Header[http.CanonicalHeaderKey(hdrKey)]
	if len(hdrValues) == 0 {
		return "-"
	}
	return strings.Join(hdrValues, ", ")
}

func (al *accessLog) GetResponseHdr(hdrKey string) string {
	hdrValues := al.ResHdr[http.CanonicalHeaderKey(hdrKey)]
	if len(hdrValues) == 0 {
		return "-"
	}
	return strings.Join(hdrValues, ", ")
}

func (al *accessLog) GetQueryString() string {
	queryStr := al.Request.Raw.URL.Query().Encode()
	if ess.IsStrEmpty(queryStr) {
		return "-"
	}
	return queryStr
}

func (al *accessLog) Reset() {
	al.StartTime = time.Time{}
	al.ElapsedDuration = 0
	al.Request = nil
	al.ResStatus = 0
	al.ResBytes = 0
	al.ResHdr = nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

func initAccessLog(logsDir string, appCfg *config.Config) error {
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
		appAccessLogChan = make(chan *accessLog, cfg.IntDefault("request.access_log.channel_buffer_size", 500))
		go listenForAccessLog()
	}

	appReqIDHdrKey = cfg.StringDefault("request.id.header", ahttp.HeaderXRequestID)

	return nil
}

func listenForAccessLog() {
	for {
		appAccessLog.Print(accessLogFormatter(<-appAccessLogChan))
	}
}

func accessLogFormatter(al *accessLog) string {
	defer releaseAccessLog(al)
	buf := acquireBuffer()
	defer releaseBuffer(buf)

	for _, part := range appAccessLogFmtFlags {
		switch part.Flag {
		case fmtFlagClientIP:
			buf.WriteString(al.Request.ClientIP)
		case fmtFlagRequestTime:
			buf.WriteString(al.FmtRequestTime(part.Format))
		case fmtFlagRequestURL:
			buf.WriteString(al.Request.Path)
		case fmtFlagRequestMethod:
			buf.WriteString(al.Request.Method)
		case fmtFlagRequestID:
			buf.WriteString(al.GetRequestHdr(appReqIDHdrKey))
		case fmtFlagRequestHeader:
			buf.WriteString(al.GetRequestHdr(part.Format))
		case fmtFlagQueryString:
			buf.WriteString(al.GetQueryString())
		case fmtFlagResponseStatus:
			buf.WriteString(fmt.Sprintf(part.Format, al.ResStatus))
		case fmtFlagResponseSize:
			buf.WriteString(fmt.Sprintf(part.Format, al.ResBytes))
		case fmtFlagResponseHeader:
			buf.WriteString(al.GetResponseHdr(part.Format))
		case fmtFlagResponseTime:
			buf.WriteString(fmt.Sprintf("%.4f", al.ElapsedDuration.Seconds()*1e3))
		}
		buf.WriteByte(' ')
	}
	return strings.TrimSpace(buf.String())
}

func acquireAccessLog() *accessLog {
	return accessLogPool.Get().(*accessLog)
}

func releaseAccessLog(al *accessLog) {
	if al != nil {
		al.Reset()
		accessLogPool.Put(al)
	}
}
