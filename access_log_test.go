// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

// Package aah is A scalable, performant, rapid development Web framework for Go
// https://aahframework.org
package aah

import (
	"bytes"
	"fmt"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"aahframework.org/ahttp.v0"
	"aahframework.org/essentials.v0-unstable"
	"aahframework.org/test.v0/assert"
)

func TestRequestAccessLogFormatter(t *testing.T) {
	startTime := time.Now()

	req := httptest.NewRequest("GET", "/oops?me=human", nil)

	resRec := httptest.NewRecorder()

	ral := &requestAccessLog{
		StartTime:       startTime,
		ElapsedDuration: time.Now().Add(2 * time.Second).Sub(startTime),
		RequestID:       "req-id:12345",
		Request:         ahttp.Request{Raw: req},
		ResStatus:       200,
		ResBytes:        63,
		ResHdr:          resRec.HeaderMap,
	}

	appAccessLogBufPool = &sync.Pool{New: func() interface{} { return &bytes.Buffer{} }}

	//We test for the default format first
	expectedDefaultFormat := fmt.Sprintf(" %s %v %v %d %d %s %s ",
		ral.RequestID, ral.StartTime.Format(time.RFC3339),
		ral.ElapsedDuration, ral.ResStatus, ral.ResBytes, ral.Request.Method, ral.Request.Raw.RequestURI)

	testFormatter(t, ral, defaultRequestAccessLogPattern, expectedDefaultFormat)

	ral.ResHdr.Add("content-type", "application/json")

	//Then for something much more diffrent
	pattern := "%reqtime:2016-05-16 %reqhdr %querystr %reshdr:content-type"

	expected := fmt.Sprintf("%s %s %s %s ",
		ral.StartTime.Format("2016-05-16"),
		"-", "me=human", ral.ResHdr.Get("content-type"),
	)
	testFormatter(t, ral, pattern, expected)

	//Test for equest access log format
	ral.Request.Header = ral.Request.Raw.Header

	ral.Request.Header.Add(ahttp.HeaderAccept, "text/html")

	ral.Request.ClientIP = "127.0.0.1"

	allAvailablePatterns := "%clientip %reqid %reqtime %restime %resstatus %ressize %reqmethod %requrl %reqhdr:accept %querystr %reshdr"

	expectedForAllAvailablePatterns := fmt.Sprintf("%s %s %s %v %d %d %s %s %s %s %s",
		ral.Request.ClientIP, ral.RequestID,
		ral.StartTime.Format(time.RFC3339), ral.ElapsedDuration,
		ral.ResStatus, ral.ResBytes, ral.Request.Method,
		ral.Request.Raw.RequestURI, "text/html", "me=human", "- ")

	testFormatter(t, ral, allAvailablePatterns, expectedForAllAvailablePatterns)
}

func TestRequestAccessLogFormatter_invalidPattern(t *testing.T) {

	var err error

	_, err = ess.ParseFmtFlag("%oops", accessLogFmtFlags)

	assert.NotNil(t, err)
}

func testFormatter(t *testing.T, ral *requestAccessLog, pattern, expected string) {

	var err error

	appAccessLogFmtFlags, err = ess.ParseFmtFlag(pattern, accessLogFmtFlags)

	assert.Nil(t, err)

	got := string(requestAccessLogFormatter(ral))

	assert.Equal(t, expected, got)
}
