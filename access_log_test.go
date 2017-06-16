package aah

import (
	"bytes"
	"fmt"
	"net/http/httptest"
	"path/filepath"
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

	//Since we are not bootstrapping the framework's engine,
	//We need to manually set this
	ral.Request.Path = "/oops"
	appAccessLogBufPool = &sync.Pool{New: func() interface{} { return &bytes.Buffer{} }}

	//We test for the default format first
	expectedDefaultFormat := fmt.Sprintf(" %s %v %v %d %d %s %s ",
		ral.RequestID, ral.StartTime.Format(time.RFC3339),
		ral.ElapsedDuration, ral.ResStatus, ral.ResBytes, ral.Request.Method, ral.Request.Path)

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
		ral.Request.Path, "text/html", "me=human", "- ")

	testFormatter(t, ral, allAvailablePatterns, expectedForAllAvailablePatterns)
}

func TestRequestAccessLogFormatterInvalidPattern(t *testing.T) {

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

func TestInitRequestAccessLog(t *testing.T) {

	cfgDir := filepath.Join(getTestdataPath(), appConfigDir())
	_ = initConfig(cfgDir)

	err := initRequestAccessLog(appLogsDir(), AppConfig())

	assert.Nil(t, err)

	assert.NotNil(t, appAccessLog)
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
	err = initSecurity(cfgDir, AppConfig())
	assert.Nil(t, err)
	assert.True(t, AppSessionManager().IsStateful())

	// Controllers
	cRegistry = controllerRegistry{}

	AddController((*Site)(nil), []*MethodInfo{
		{
			Name:       "GetInvolved",
			Parameters: []*ParameterInfo{},
		},
		{
			Name:       "ContributeCode",
			Parameters: []*ParameterInfo{},
		},
		{
			Name:       "Credits",
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
