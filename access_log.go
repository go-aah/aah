// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

// Package aah is A scalable, performant, rapid development Web framework for Go
// https://aahframework.org
package aah

import (
	"bytes"
	"strconv"
	"time"

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
)

type (

	//requestAccessLog contains data about the current request
	requestAccessLog struct {
		startTime   time.Time
		ctx         *Context
		requestID   string
		logPattern  string
		elapsedTime time.Duration
	}

	requestAccessLogChan chan requestAccessLog
)

func newRequestAccessLogChan() requestAccessLogChan {
	c := make(chan requestAccessLog)

	go listenForLogEntry(c)
	return c
}

func listenForLogEntry(c requestAccessLogChan) {

	for {

		select {

		case ral := <-c:

			flagParts, err := ess.ParseFmtFlag(ral.logPattern, accessLogFmtFlags)
			if err != nil {
				return
			}
			log.Debug(string(requestAccessLogFormatter(flagParts, ral)))
		default:
			//Nothing
		}
	}
}

func requestAccessLogFormatter(flags []ess.FmtFlagPart, ral requestAccessLog) []byte {

	var buf bytes.Buffer

	for _, part := range flags {
		switch part.Flag {
		case fmtFlagClientIP:
			buf.WriteString(ral.ctx.Req.Raw.RemoteAddr)
			buf.WriteString(" | ")
		case fmtFlagRequestTime:

			buf.WriteString(ral.startTime.String())

			buf.WriteString(" | ")
		case fmtFlagRequestURL:
			buf.WriteString(ral.ctx.Req.Raw.RequestURI)

			buf.WriteString(" | ")
		case fmtFlagRequestMethod:
			buf.WriteString(ral.ctx.Req.Method)

			buf.WriteString(" | ")
		case fmtFlagRequestID:
			rid := "-"

			if x := ral.requestID; x != "" {
				rid = x
			}
			buf.WriteString(rid)

			buf.WriteString(" | ")
		case fmtFlagRequestHeader:
			hdr := "-"
			if part.Format != "" {
				hdr = ral.ctx.Req.Header.Get(part.Format)
			}

			buf.WriteString(hdr)

			buf.WriteString(" | ")

		case fmtFlagQueryString:
			queryStr := "-"

			if x := ral.ctx.Req.Raw.URL.String(); x != "" {
				queryStr = x
			}
			buf.WriteString(queryStr)

			buf.WriteString(" | ")
		case fmtFlagResponseStatus:
			buf.WriteString(strconv.Itoa(ral.ctx.Res.Status()))

			buf.WriteString(" | ")
		case fmtFlagResponseSize:
			buf.WriteString(strconv.Itoa(ral.ctx.Res.BytesWritten()))

			buf.WriteString(" | ")
		case fmtFlagResponseHeader:

			hdr := "-"
			if part.Format != "" {
				hdr = ral.ctx.Res.Header().Get(part.Format)
			}
			buf.WriteString(hdr)

			buf.WriteString(" | ")
		case fmtFlagResponseTime:
			buf.WriteString(ral.elapsedTime.String())

			buf.WriteString(" | ")
		}
	}
	return buf.Bytes()
}
