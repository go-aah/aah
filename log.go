// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"aahframe.work/ahttp"
	"aahframe.work/config"
	"aahframe.work/essentials"
	"aahframe.work/internal/util"
	"aahframe.work/log"
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Access Logger Definitions
//______________________________________________________________________________

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

func (a *Application) initAccessLog() error {
	// log file configuration
	cfg := config.NewEmpty()
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

type accessLogger struct {
	a        *Application
	logger   *log.Logger
	fmtFlags []ess.FmtFlagPart
	logChan  chan *accessLog
	logPool  *sync.Pool
}

func (aal *accessLogger) Log(ctx *Context) {
	if ctx.IsStaticRoute() && !aal.a.settings.StaticAccessLogEnabled {
		return
	}
	al := aal.logPool.Get().(*accessLog)
	al.StartTime = ctx.Get(reqStartTimeKey).(time.Time)

	// All the bytes have been written on the wire
	// so calculate elapsed time
	al.ElapsedDuration = time.Since(al.StartTime)

	req := *ctx.Req
	al.Request = &req
	if h := req.Header[aal.a.settings.RequestIDHeaderKey]; len(h) > 0 {
		al.RequestID = h[0]
	} else {
		al.RequestID = "-"
	}
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
	buf := acquireBuilder()
	defer releaseBuilder(buf)

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
	if len(queryStr) == 0 {
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

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Dump logger Definitions
//______________________________________________________________________________

const (
	keyAahRequestBodyBuf  = "_aahRequestBodyBuf"
	keyAahResponseBodyBuf = "_aahResponseBodyBuf"
)

func (a *Application) initDumpLog() error {
	// log file configuration
	cfg := config.NewEmpty()
	file := a.Config().StringDefault("server.dump_log.file", "")

	cfg.SetString("log.receiver", "file")
	if ess.IsStrEmpty(file) {
		cfg.SetString("log.file", filepath.Join(a.logsDir(), a.binaryFilename()+"-dump.log"))
	} else {
		abspath, err := filepath.Abs(file)
		if err != nil {
			return err
		}
		cfg.SetString("log.file", abspath)
	}

	cfg.SetString("log.pattern", "%message")

	adLog, err := log.New(cfg)
	if err != nil {
		return err
	}

	a.dumpLog = &dumpLogger{
		a:               a,
		logger:          adLog,
		logRequestBody:  a.Config().BoolDefault("server.dump_log.request_body", false),
		logResponseBody: a.Config().BoolDefault("server.dump_log.response_body", false),
	}

	return nil
}

type dumpLogger struct {
	a               *Application
	logger          *log.Logger
	logRequestBody  bool
	logResponseBody bool
}

func (d *dumpLogger) Dump(ctx *Context) {
	buf := acquireBuffer()
	defer releaseBuffer(buf)

	// Request
	uri := fmt.Sprintf("%s://%s%s", ctx.Req.Scheme, ctx.Req.Host, ctx.Req.Path)
	if qs := ctx.Req.URL().RawQuery; len(qs) > 0 {
		uri += "?" + qs
	}

	buf.WriteString(fmt.Sprintf("\nURI: %s\n", uri))
	buf.WriteString(fmt.Sprintf("METHOD: %s\n", ctx.Req.Method))
	buf.WriteString(fmt.Sprintf("PROTO: %s\n", ctx.Req.Proto))
	buf.WriteString("HEADERS:\n")
	buf.WriteString(d.composeHeaders(ctx.Req.Header) + "\n")
	if d.logRequestBody {
		buf.WriteString("BODY:\n")
		d.writeBody(keyAahRequestBodyBuf, ctx.Req.ContentType().Mime, buf, ctx)
	}

	buf.WriteString("\n\n-----------------------------------------------------------------------\n\n")

	// Response
	buf.WriteString(fmt.Sprintf("STATUS: %d %s\n", ctx.Res.Status(), http.StatusText(ctx.Res.Status())))
	buf.WriteString(fmt.Sprintf("BYTES WRITTEN: %d\n", ctx.Res.BytesWritten()))
	buf.WriteString("HEADERS:\n")
	buf.WriteString(d.composeHeaders(ctx.Res.Header()) + "\n")
	if d.logResponseBody {
		buf.WriteString("BODY:\n")
		d.writeBody(keyAahResponseBodyBuf, ctx.Reply().ContType, buf, ctx)
	}

	buf.WriteString("\n\n=======================================================================")

	d.logger.Print(buf.String())
}

func (d *dumpLogger) writeBody(key, ct string, w *bytes.Buffer, ctx *Context) {
	cbuf := ctx.Get(key)
	if cbuf == nil {
		w.WriteString("    ***** NO CONTENT *****")
		return
	}

	b := cbuf.(*bytes.Buffer)
	switch util.OnlyMIME(ct) {
	case ahttp.ContentTypeHTML.Mime, ahttp.ContentTypeForm.Mime,
		ahttp.ContentTypeMultipartForm.Mime, ahttp.ContentTypePlainText.Mime:
		_, _ = b.WriteTo(w)
	case ahttp.ContentTypeJSON.Mime, ahttp.ContentTypeJSONText.Mime:
		_ = json.Indent(w, b.Bytes(), "", "    ")
	case ahttp.ContentTypeXML.Mime, ahttp.ContentTypeXMLText.Mime:
		// TODO XML formatting
		_, _ = b.WriteTo(w)
	}
	releaseBuffer(b)
}

func (d *dumpLogger) composeHeaders(hdrs http.Header) string {
	var str []string
	for _, k := range sortHeaderKeys(hdrs) {
		str = append(str, fmt.Sprintf("    %s: %s", k, strings.Join(hdrs[k], ", ")))
	}
	return strings.Join(str, "\n")
}

func sortHeaderKeys(hdrs http.Header) []string {
	var keys []string
	for key := range hdrs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
