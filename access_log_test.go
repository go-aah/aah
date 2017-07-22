// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0-unstable"
	"aahframework.org/test.v0/assert"
)

func TestAccessLogFormatter(t *testing.T) {
	al := createTestAccessLog()

	// Since we are not bootstrapping the framework's engine,
	// We need to manually set this
	al.Request.Path = "/oops"
	al.Request.Header.Set(ahttp.HeaderXRequestID, "5946ed129bf23409520736de")

	// Testing for the default access log pattern first
	expectedDefaultFormat := fmt.Sprintf("%v - %v %v %v %v %v %v %v %v",
		"[::1]", al.StartTime.Format(time.RFC3339),
		al.Request.Method, al.Request.Path, al.Request.Raw.Proto, al.ResStatus,
		al.ResBytes, fmt.Sprintf("%.4f", al.ElapsedDuration.Seconds()*1e3), "-")

	testFormatter(t, al, appDefaultAccessLogPattern, expectedDefaultFormat)

	// Testing custom access log pattern
	al = createTestAccessLog()
	al.ResHdr.Add("content-type", "application/json")
	pattern := "%reqtime:2016-05-16 %reqhdr %querystr %reshdr:content-type"
	expected := fmt.Sprintf(`%s %s "%s" "%s"`, al.StartTime.Format("2016-05-16"), "-", "me=human", al.ResHdr.Get("Content-Type"))

	testFormatter(t, al, pattern, expected)

	// Testing all available access log pattern
	al = createTestAccessLog()
	al.Request.Header = al.Request.Raw.Header
	al.Request.Header.Add(ahttp.HeaderAccept, "text/html")
	al.Request.Header.Set(ahttp.HeaderXRequestID, "5946ed129bf23409520736de")
	al.Request.ClientIP = "127.0.0.1"
	al.ResHdr.Add("content-type", "application/json")
	allAvailablePatterns := "%clientip %reqid %reqtime %restime %resstatus %ressize %reqmethod %requrl %reqhdr:accept %querystr %reshdr"
	expectedForAllAvailablePatterns := fmt.Sprintf(`%s "%s" %s %v %d %d %s %s "%s" "%s" %s`,
		al.Request.ClientIP, al.Request.Header.Get(ahttp.HeaderXRequestID),
		al.StartTime.Format(time.RFC3339), fmt.Sprintf("%.4f", al.ElapsedDuration.Seconds()*1e3),
		al.ResStatus, al.ResBytes, al.Request.Method,
		al.Request.Path, "text/html", "me=human", "-")

	testFormatter(t, al, allAvailablePatterns, expectedForAllAvailablePatterns)
}

func TestAccessLogFormatterInvalidPattern(t *testing.T) {
	_, err := ess.ParseFmtFlag("%oops", accessLogFmtFlags)

	assert.NotNil(t, err)
}

func TestAccessLogInitDefault(t *testing.T) {
	testAccessInit(t, `
		request {
			access_log {
		    # Default value is false
		    enable = true
		  }
		}
		`)

	testAccessInit(t, `
		request {
			access_log {
		    # Default value is false
		    enable = true

				file = "testdata/test-access.log"
		  }
		}
		`)

	testAccessInit(t, `
		request {
			access_log {
		    # Default value is false
		    enable = true

				file = "/tmp/test-access.log"
		  }
		}
		`)
}

func TestEngineAccessLog(t *testing.T) {
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

func testFormatter(t *testing.T, al *accessLog, pattern, expected string) {
	var err error
	appAccessLogFmtFlags, err = ess.ParseFmtFlag(pattern, accessLogFmtFlags)

	assert.Nil(t, err)
	assert.Equal(t, expected, accessLogFormatter(al))
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
	err := initAccessLog(logsDir, cfg)

	assert.Nil(t, err)
	assert.NotNil(t, appAccessLog)
}

func createTestAccessLog() *accessLog {
	startTime := time.Now()
	req := httptest.NewRequest("GET", "/oops?me=human", nil)
	req.Header = http.Header{}

	w := httptest.NewRecorder()

	al := acquireAccessLog()
	al.StartTime = startTime
	al.ElapsedDuration = time.Now().Add(2 * time.Second).Sub(startTime)
	al.Request = &ahttp.Request{Raw: req, Header: req.Header, ClientIP: "[::1]"}
	al.ResStatus = 200
	al.ResBytes = 63
	al.ResHdr = w.HeaderMap

	return al
}
