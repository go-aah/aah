// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0-unstable"
	"aahframework.org/pool.v0"
	"aahframework.org/test.v0/assert"
)

func TestRequestAccessLogFormatter(t *testing.T) {
	startTime := time.Now()
	req := httptest.NewRequest("GET", "/oops?me=human", nil)
	req.Header = make(http.Header)

	w := httptest.NewRecorder()

	ral := &requestAccessLog{
		StartTime:       startTime,
		ElapsedDuration: time.Now().Add(2 * time.Second).Sub(startTime),
		Request:         ahttp.Request{Raw: req, Header: req.Header, ClientIP: "[::1]"},
		ResStatus:       200,
		ResBytes:        63,
		ResHdr:          w.HeaderMap,
	}

	// Since we are not bootstrapping the framework's engine,
	// We need to manually set this
	ral.Request.Path = "/oops"
	ral.Request.Header.Set(ahttp.HeaderXRequestID, "5946ed129bf23409520736de")
	bufPool = pool.NewPool(2, func() interface{} { return &bytes.Buffer{} })

	// Testing for the default access log pattern first
	expectedDefaultFormat := fmt.Sprintf("%s %s %v %v %d %d %s %s",
		"[::1]", "5946ed129bf23409520736de", ral.StartTime.Format(time.RFC3339),
		fmt.Sprintf("%.4f", ral.ElapsedDuration.Seconds()*1e3), ral.ResStatus,
		ral.ResBytes, ral.Request.Method, ral.Request.Path)

	testFormatter(t, ral, appDefaultAccessLogPattern, expectedDefaultFormat)

	// Testing custom access log pattern
	ral.ResHdr.Add("content-type", "application/json")
	pattern := "%reqtime:2016-05-16 %reqhdr %querystr %reshdr:content-type"
	expected := fmt.Sprintf("%s %s %s %s", ral.StartTime.Format("2016-05-16"), "-", "me=human", ral.ResHdr.Get("Content-Type"))

	testFormatter(t, ral, pattern, expected)

	// Testing all available access log pattern
	ral.Request.Header = ral.Request.Raw.Header
	ral.Request.Header.Add(ahttp.HeaderAccept, "text/html")
	ral.Request.ClientIP = "127.0.0.1"
	allAvailablePatterns := "%clientip %reqid %reqtime %restime %resstatus %ressize %reqmethod %requrl %reqhdr:accept %querystr %reshdr"
	expectedForAllAvailablePatterns := fmt.Sprintf("%s %s %s %v %d %d %s %s %s %s %s",
		ral.Request.ClientIP, ral.Request.Header.Get(ahttp.HeaderXRequestID),
		ral.StartTime.Format(time.RFC3339), fmt.Sprintf("%.4f", ral.ElapsedDuration.Seconds()*1e3),
		ral.ResStatus, ral.ResBytes, ral.Request.Method,
		ral.Request.Path, "text/html", "me=human", "-")

	testFormatter(t, ral, allAvailablePatterns, expectedForAllAvailablePatterns)
}

func TestRequestAccessLogFormatterInvalidPattern(t *testing.T) {
	_, err := ess.ParseFmtFlag("%oops", accessLogFmtFlags)

	assert.NotNil(t, err)
}

func TestRequestAccessLogInitDefault(t *testing.T) {
	testAccessInit(t, `
		request {
			access_log {
		    # Default value is false
		    enable = true
		  }
		}
		`)
}

func TestEngineRequestAccessLog(t *testing.T) {
	// App Config
	cfgDir := filepath.Join(getTestdataPath(), appConfigDir())
	err := initConfig(cfgDir)
	assert.Nil(t, err)
	assert.NotNil(t, AppConfig())

	AppConfig().SetString("server.port", "8080")

	// Router
	err = initRoutes(cfgDir, AppConfig())
	assert.Nil(t, err)
	assert.NotNil(t, AppRouter())

	// Security
	err = initSecurity(AppConfig())
	assert.Nil(t, err)
	assert.True(t, AppSessionManager().IsStateful())

	// Controllers
	cRegistry = controllerRegistry{}

	AddController((*Site)(nil), []*MethodInfo{
		{
			Name:       "GetInvolved",
			Parameters: []*ParameterInfo{},
		},
	})

	AppConfig().SetBool("request.access_log.enable", true)

	e := newEngine(AppConfig())
	req := httptest.NewRequest("GET", "localhost:8080/get-involved.html", nil)
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)

	assert.True(t, e.isAccessLogEnabled)
}

func testFormatter(t *testing.T, ral *requestAccessLog, pattern, expected string) {
	var err error
	appAccessLogFmtFlags, err = ess.ParseFmtFlag(pattern, accessLogFmtFlags)

	assert.Nil(t, err)
	assert.Equal(t, expected, requestAccessLogFormatter(ral))
}

func testAccessInit(t *testing.T, cfgStr string) {
	buildTime := time.Now().Format(time.RFC3339)
	SetAppBuildInfo(&BuildInfo{
		BinaryName: "testapp",
		Date:       buildTime,
		Version:    "1.0.0",
	})

	cfg, _ := config.ParseString(cfgStr)
	logsDir := filepath.Join(getTestdataPath(), appLogsDir())
	err := initRequestAccessLog(logsDir, cfg)

	assert.Nil(t, err)
	assert.NotNil(t, appAccessLog)
}
