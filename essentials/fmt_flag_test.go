// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org/essentials source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ess

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFmtParseFlagLog(t *testing.T) {
	const (
		FmtFlagLevel FmtFlag = iota
		FmtFlagTime
		FmtFlagUTCTime
		FmtFlagLongfile
		FmtFlagShortfile
		FmtFlagLine
		FmtFlagMessage
		FmtFlagCustom
	)

	logFmtFlags := map[string]FmtFlag{
		"level":     FmtFlagLevel,
		"time":      FmtFlagTime,
		"utctime":   FmtFlagUTCTime,
		"longfile":  FmtFlagLongfile,
		"shortfile": FmtFlagShortfile,
		"line":      FmtFlagLine,
		"message":   FmtFlagMessage,
		"custom":    FmtFlagCustom,
	}

	flagParts, err := ParseFmtFlag("%time:2006-01-02 15:04:05.000 %level %custom:- %message", logFmtFlags)
	assert.Nil(t, err)

	assertFlagPart(t, "time", "2006-01-02 15:04:05.000", FmtFlag(1), flagParts[0])
	assertFlagPart(t, "level", "%v", FmtFlag(0), flagParts[1])
	assertFlagPart(t, "custom", "-", FmtFlag(7), flagParts[2])
	assertFlagPart(t, "message", "%v", FmtFlag(6), flagParts[3])

	// Unknown flag
	flagParts, err = ParseFmtFlag("%myflag", logFmtFlags)
	assert.NotNil(t, err)
	assert.Equal(t, "fmtflag: unknown flag 'myflag'", err.Error())
	assert.True(t, len(flagParts) == 0)
}

func TestFmtParseFlagAccessLog(t *testing.T) {
	const (
		fmtFlagClientIP FmtFlag = iota
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

	accessLogFmtFlags := map[string]FmtFlag{
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

	flagParts, err := ParseFmtFlag("%clientip %reqid %reqtime %restime %resstatus %ressize %reqmethod %requrl %reqhdr:Referer %reshdr:Server", accessLogFmtFlags)
	assert.Nil(t, err)

	assertFlagPart(t, "clientip", "%v", FmtFlag(0), flagParts[0])
	assertFlagPart(t, "reqid", "%v", FmtFlag(4), flagParts[1])
	assertFlagPart(t, "reqtime", "%v", FmtFlag(1), flagParts[2])
	assertFlagPart(t, "restime", "%v", FmtFlag(10), flagParts[3])
	assertFlagPart(t, "resstatus", "%v", FmtFlag(7), flagParts[4])
	assertFlagPart(t, "ressize", "%v", FmtFlag(8), flagParts[5])
	assertFlagPart(t, "reqmethod", "%v", FmtFlag(3), flagParts[6])
	assertFlagPart(t, "requrl", "%v", FmtFlag(2), flagParts[7])
	assertFlagPart(t, "reqhdr", "Referer", FmtFlag(5), flagParts[8])
	assertFlagPart(t, "reshdr", "Server", FmtFlag(9), flagParts[9])
}

func assertFlagPart(t *testing.T, name, format string, fflag FmtFlag, flagPart FmtFlagPart) {
	t.Logf("Fmt Flag: %v", format)
	assert.Equal(t, name, flagPart.Name)
	assert.Equal(t, format, flagPart.Format)
	assert.Equal(t, fflag, flagPart.Flag)
}
