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
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
)

const (
	fmtFlagClientIP ess.FmtFlag = iota
	fmtFlagRequestTime
	fmtFlagRequestURL
	fmtFlagRequestMethod
	fmtFlagRequestProto
	fmtFlagRequestID
	fmtFlagRequestHeader
	fmtFlagQueryString
	fmtFlagResponseStatus
	fmtFlagResponseSize
	fmtFlagResponseHeader
	fmtFlagResponseTime
	fmtFlagCustom
)

var (
	accessLogFmtFlags = map[string]ess.FmtFlag{
		"clientip":  fmtFlagClientIP,
		"reqtime":   fmtFlagRequestTime,
		"requrl":    fmtFlagRequestURL,
		"reqmethod": fmtFlagRequestMethod,
		"reqproto":  fmtFlagRequestProto,
		"reqid":     fmtFlagRequestID,
		"reqhdr":    fmtFlagRequestHeader,
		"querystr":  fmtFlagQueryString,
		"resstatus": fmtFlagResponseStatus,
		"ressize":   fmtFlagResponseSize,
		"reshdr":    fmtFlagResponseHeader,
		"restime":   fmtFlagResponseTime,
		"custom":    fmtFlagCustom,
	}

	defaultAccessLogPattern = "%clientip %custom:- %reqtime %reqmethod %requrl %reqproto %resstatus %ressize %restime %reqhdr:referer"
	reqStartTimeKey         = "_appReqStartTimeKey"
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// app Unexported methods
//______________________________________________________________________________

func (a *app) initAccessLog() error {
	// log file configuration
	cfg, _ := config.ParseString("")
	file := a.Config().StringDefault("server.access_log.file", "")

	cfg.SetString("log.receiver", "file")
	if ess.IsStrEmpty(file) {
		cfg.SetString("log.file", filepath.Join(a.logsDir(), a.binaryFilename()+"-access.log"))
	} else {
		abspath, err := filepath.Abs(file)
		if err != nil {
			return err
		}
		cfg.SetString("log.file", abspath)
	}

	cfg.SetString("log.pattern", "%message")

	// initialize request access log file
	aaLog, err := log.New(cfg)
	if err != nil {
		return err
	}

	aaLogger := &accessLogger{
		a:       a,
		logger:  aaLog,
		logPool: &sync.Pool{New: func() interface{} { return new(accessLog) }},
	}

	// parse request access log pattern
	pattern := a.Config().StringDefault("server.access_log.pattern", defaultAccessLogPattern)
	aaLogFmtFlags, err := ess.ParseFmtFlag(pattern, accessLogFmtFlags)
	if err != nil {
		return err
	}
	aaLogger.fmtFlags = aaLogFmtFlags

	// initialize request access log channel
	aaLogger.logChan = make(chan *accessLog, a.Config().IntDefault("server.access_log.channel_buffer_size", 500))

	a.accessLog = aaLogger
	go a.accessLog.listenToLogChan()

	return nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// accessLogger
//______________________________________________________________________________

type accessLogger struct {
	a        *app
	logger   *log.Logger
	fmtFlags []ess.FmtFlagPart
	logChan  chan *accessLog
	logPool  *sync.Pool
}

func (aal *accessLogger) Log(ctx *Context) {
	al := aal.logPool.Get().(*accessLog)
	al.StartTime = ctx.Get(reqStartTimeKey).(time.Time)

	// All the bytes have been written on the wire
	// so calculate elapsed time
	al.ElapsedDuration = time.Since(al.StartTime)

	req := *ctx.Req
	al.Request = &req
	al.RequestID = firstNonZeroString(req.Header.Get(aal.a.requestIDHeaderKey), "-")
	al.ResStatus = ctx.Res.Status()
	al.ResBytes = ctx.Res.BytesWritten()
	al.ResHdr = ctx.Res.Header()

	aal.logChan <- al
}

func (aal *accessLogger) listenToLogChan() {
	for al := range aal.logChan {
		aal.logger.Print(aal.accessLogFormatter(al))
	}
}

func (aal *accessLogger) accessLogFormatter(al *accessLog) string {
	defer aal.releaseAccessLog(al)
	buf := acquireBuffer()
	defer releaseBuffer(buf)

	for _, part := range aal.fmtFlags {
		switch part.Flag {
		case fmtFlagClientIP:
			buf.WriteString(al.Request.ClientIP())
		case fmtFlagRequestTime:
			buf.WriteString(al.FmtRequestTime(part.Format))
		case fmtFlagRequestURL:
			buf.WriteString(al.Request.Path)
		case fmtFlagRequestMethod:
			buf.WriteString(al.Request.Method)
		case fmtFlagRequestProto:
			buf.WriteString(al.Request.Unwrap().Proto)
		case fmtFlagRequestID:
			buf.WriteString(al.RequestID)
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
		case fmtFlagCustom:
			buf.WriteString(part.Format)
		}
		buf.WriteByte(' ')
	}
	return strings.TrimSpace(buf.String())
}

func (aal *accessLogger) releaseAccessLog(al *accessLog) {
	al.Reset()
	aal.logPool.Put(al)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// accessLog
//______________________________________________________________________________

//accessLog contains data about the current request
type accessLog struct {
	StartTime       time.Time
	ElapsedDuration time.Duration
	Request         *ahttp.Request
	RequestID       string
	ResStatus       int
	ResBytes        int
	ResHdr          http.Header
}

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
	return `"` + strings.Join(hdrValues, ", ") + `"`
}

func (al *accessLog) GetResponseHdr(hdrKey string) string {
	hdrValues := al.ResHdr[http.CanonicalHeaderKey(hdrKey)]
	if len(hdrValues) == 0 {
		return "-"
	}
	return `"` + strings.Join(hdrValues, ", ") + `"`
}

func (al *accessLog) GetQueryString() string {
	queryStr := al.Request.URL().Query().Encode()
	if ess.IsStrEmpty(queryStr) {
		return "-"
	}
	return `"` + queryStr + `"`
}

func (al *accessLog) Reset() {
	al.StartTime = time.Time{}
	al.ElapsedDuration = 0
	al.Request = nil
	al.ResStatus = 0
	al.ResBytes = 0
	al.ResHdr = nil
}
