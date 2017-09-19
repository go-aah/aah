// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0-unstable"
	"aahframework.org/log.v0-unstable"
	"aahframework.org/test.v0/assert"
)

type (
	Site struct {
		*Context
	}

	sample struct {
		ProductID   int    `bind:"id"`
		ProductName string `bind:"product_name"`
		Username    string `bind:"username"`
		Email       string `bind:"email"`
		Page        int    `bind:"page"`
		Count       string `bind:"count"`
	}

	sampleJSON struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Email     string `json:"email"`
		Number    int    `json:"number"`
	}
)

func (s *Site) GetInvolved() {
	s.Session().Set("test1", "test1value")
	s.Reply().Text("GetInvolved action")
}

func (s *Site) Credits() {
	s.Log().Info("Credits action called")
	s.Reply().
		Header("X-Custom-Header", "custom value").
		DisableGzip().
		JSON(Data{
			"message": "This is credits page",
			"code":    1000001,
		})
}

func (s *Site) ContributeCode() {
	s.Log().Info("ContributeCode action called")
	panic("panic flow testing")
}

func (s *Site) Before() {
	log.Info("Before interceptor")
}

func (s *Site) After() {
	log.Info("After interceptor")
}

func (s *Site) BeforeGetInvolved() {
	log.Info("Before GetInvolved interceptor")
}

func (s *Site) AfterGetInvolved() {
	log.Info("After GetInvolved interceptor")
}

func (s *Site) AutoBind(id int, info *sample) {
	s.Log().Info("AutoBind action called")
	log.Info("ID:", id)
	log.Infof("Info: %+v", info)
	s.Reply().Text("Data have been recevied successfully")
}

func (s *Site) JSONRequest(info *sampleJSON) {
	s.Log().Info("JSONRequest action called")
	log.Infof("JSON Info: %+v", info)
	s.Reply().JSON(Data{
		"success": true,
		"data":    info,
	})
}

func testEngineMiddleware(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("X-Custom-Name", "test engine middleware")
}

func TestEngineNew(t *testing.T) {
	cfgDir := filepath.Join(getTestdataPath(), appConfigDir())
	err := initConfig(cfgDir)
	assert.Nil(t, err)
	assert.NotNil(t, AppConfig())

	AppConfig().SetInt("render.gzip.level", 5)
	AppConfig().SetString("request.id.header", "X-Test-Request-Id")

	e := newEngine(AppConfig())
	assert.Equal(t, "X-Test-Request-Id", e.requestIDHeader)
	assert.True(t, e.isRequestIDEnabled)
	assert.True(t, e.isGzipEnabled)

	ctx := acquireContext()
	req := httptest.NewRequest("GET", "http://localhost:8080/doc/v0.3/mydoc.html", nil)
	ctx.Req = ahttp.AcquireRequest(req)
	assert.NotNil(t, ctx)
	assert.NotNil(t, ctx.Req)
	releaseContext(ctx)

	buf := acquireBuffer()
	assert.NotNil(t, buf)
	releaseBuffer(buf)

	appLogFatal = func(v ...interface{}) { t.Log(v) }
	AppConfig().SetInt("render.gzip.level", 10)
	e = newEngine(AppConfig())
	assert.NotNil(t, e)
}

func TestEngineServeHTTP(t *testing.T) {
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

	err = initLogs(getTestdataPath(), AppConfig())
	assert.Nil(t, err)

	err = initAccessLog(getTestdataPath(), AppConfig())
	assert.Nil(t, err)
	appAccessLog.SetWriter(os.Stdout)

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
		{
			Name: "AutoBind",
			Parameters: []*ParameterInfo{
				&ParameterInfo{Name: "id", Type: reflect.TypeOf((*int)(nil))},
				&ParameterInfo{Name: "info", Type: reflect.TypeOf((**sample)(nil))},
			},
		},
		{
			Name: "JSONRequest",
			Parameters: []*ParameterInfo{
				&ParameterInfo{Name: "info", Type: reflect.TypeOf((**sampleJSON)(nil))},
			},
		},
	})

	// Middlewares
	Middlewares(ToMiddleware(testEngineMiddleware))

	AppConfig().SetBool("server.access_log.enable", true)

	// Engine
	e := newEngine(AppConfig())

	// Request 1
	r1 := httptest.NewRequest("GET", "http://localhost:8080/doc/v0.3/mydoc.html", nil)
	w1 := httptest.NewRecorder()
	e.ServeHTTP(w1, r1)

	resp1 := w1.Result()
	assert.Equal(t, 404, resp1.StatusCode)
	assert.True(t, strings.Contains(resp1.Status, "Not Found"))
	assert.Equal(t, "aah-go-server", resp1.Header.Get(ahttp.HeaderServer))

	// Request 2
	r2 := httptest.NewRequest("GET", "http://localhost:8080/get-involved.html", nil)
	r2.Header.Add(ahttp.HeaderAcceptEncoding, "gzip, deflate, sdch, br")
	w2 := httptest.NewRecorder()
	e.ServeHTTP(w2, r2)

	resp2 := w2.Result()
	body2 := getResponseBody(resp2)
	assert.Equal(t, 200, resp2.StatusCode)
	assert.True(t, strings.Contains(resp2.Status, "OK"))
	assert.Equal(t, "GetInvolved action", body2)
	assert.Equal(t, "test engine middleware", resp2.Header.Get("X-Custom-Name"))

	// Request 3
	r3 := httptest.NewRequest("GET", "http://localhost:8080/contribute-to-code.html", nil)
	w3 := httptest.NewRecorder()
	e.ServeHTTP(w3, r3)

	resp3 := w3.Result()
	body3 := getResponseBody(resp3)
	assert.Equal(t, 500, resp3.StatusCode)
	assert.True(t, strings.Contains(resp3.Status, "Internal Server Error"))
	assert.True(t, strings.Contains(body3, "Internal Server Error"))

	// Request 4 static
	r4 := httptest.NewRequest("GET", "http://localhost:8080/assets/logo.png", nil)
	w4 := httptest.NewRecorder()
	e.ServeHTTP(w4, r4)

	resp4 := w4.Result()
	assert.NotNil(t, resp4)

	// Request 5 RedirectTrailingSlash - 302 status
	wd, _ := os.Getwd()
	appBaseDir = wd
	r5 := httptest.NewRequest("GET", "http://localhost:8080/testdata", nil)
	w5 := httptest.NewRecorder()
	e.ServeHTTP(w5, r5)

	resp5 := w5.Result()
	assert.Equal(t, 302, resp5.StatusCode)
	assert.True(t, strings.Contains(resp5.Status, "Found"))
	assert.Equal(t, "http://localhost:8080/testdata/", resp5.Header.Get(ahttp.HeaderLocation))

	// Request 6 Directory Listing
	appIsSSLEnabled = true
	appIsProfileProd = true
	AppSecurityManager().SecureHeaders.CSP = "default-erc 'self'"
	r6 := httptest.NewRequest("GET", "http://localhost:8080/testdata/", nil)
	r6.Header.Add(e.requestIDHeader, "D9391509-595B-4B92-BED7-F6A9BE0DFCF2")
	r6.Header.Add(ahttp.HeaderAcceptEncoding, "gzip, deflate, sdch, br")
	w6 := httptest.NewRecorder()
	e.ServeHTTP(w6, r6)

	resp6 := w6.Result()
	body6 := getResponseBody(resp6)
	assert.True(t, strings.Contains(body6, "Listing of /testdata/"))
	assert.True(t, strings.Contains(body6, "config/"))
	AppSecurityManager().SecureHeaders.CSP = ""
	appIsSSLEnabled = false
	appIsProfileProd = false

	// Request 7 Custom Headers
	r7 := httptest.NewRequest("GET", "http://localhost:8080/credits", nil)
	r7.Header.Add(ahttp.HeaderAcceptEncoding, "gzip, deflate, sdch, br")
	w7 := httptest.NewRecorder()
	e.ServeHTTP(w7, r7)

	resp7 := w7.Result()
	body7 := getResponseBody(resp7)
	assert.Equal(t, `{"code":1000001,"message":"This is credits page"}`, body7)
	assert.Equal(t, "custom value", resp7.Header.Get("X-Custom-Header"))

	// Request 8
	r8 := httptest.NewRequest("POST", "http://localhost:8080/credits", nil)
	r8.Header.Add(ahttp.HeaderAcceptEncoding, "gzip, deflate, sdch, br")
	r8.Header.Add(ahttp.HeaderAccept, ahttp.ContentTypeJSON.String())
	w8 := httptest.NewRecorder()
	e.ServeHTTP(w8, r8)

	// Method Not Allowed 405 response
	resp8 := w8.Result()
	body8 := getResponseBody(resp8)
	assert.Equal(t, 405, resp8.StatusCode)
	assert.Equal(t, `{"code":405,"message":"Method Not Allowed"}`, body8)
	assert.Equal(t, "GET, OPTIONS", resp8.Header.Get("Allow"))

	// Request 9 Auto Options
	r9 := httptest.NewRequest("OPTIONS", "http://localhost:8080/credits", nil)
	r9.Header.Add(ahttp.HeaderAcceptEncoding, "gzip, deflate, sdch, br")
	w9 := httptest.NewRecorder()
	e.ServeHTTP(w9, r9)

	resp9 := w9.Result()
	assert.Equal(t, 200, resp9.StatusCode)
	assert.Equal(t, "GET, OPTIONS", resp9.Header.Get("Allow"))

	// Request 10 Auto Bind request
	autobindPriority = []string{"Q", "F", "P"}
	requestParsers[ahttp.ContentTypeMultipartForm.Mime] = multipartFormParser
	requestParsers[ahttp.ContentTypeForm.Mime] = formParser
	secret := AppSecurityManager().AntiCSRF.GenerateSecret()
	secretstr := AppSecurityManager().AntiCSRF.SaltCipherSecret(secret)
	form := url.Values{}
	form.Add("product_name", "Test Product")
	form.Add("username", "welcome")
	form.Add("email", "welcome@welcome.com")
	form.Add("anti_csrf_token", secretstr)
	r10 := httptest.NewRequest("POST", "http://localhost:8080/products/100002?page=10&count=20",
		strings.NewReader(form.Encode()))
	r10.Header.Set(ahttp.HeaderContentType, ahttp.ContentTypeForm.String())

	w10 := httptest.NewRecorder()
	_ = AppSecurityManager().AntiCSRF.SetCookie(w10, secret)
	cookieValue := w10.Header().Get("Set-Cookie")
	r10.Header.Set(ahttp.HeaderCookie, cookieValue)

	e.ServeHTTP(w10, r10)

	resp10 := w10.Result()
	body10 := getResponseBody(resp10)
	assert.NotNil(t, resp10)
	assert.Equal(t, http.StatusOK, resp10.StatusCode)
	assert.Equal(t, "text/plain; charset=utf-8", resp10.Header.Get(ahttp.HeaderContentType))
	assert.Equal(t, "Data have been recevied successfully", body10)

	// Request 11 multipart
	r11 := httptest.NewRequest("POST", "http://localhost:8080/products/100002?page=10&count=20",
		strings.NewReader(form.Encode()))
	r11.Header.Set(ahttp.HeaderContentType, ahttp.ContentTypeForm.String())
	w11 := httptest.NewRecorder()
	r11.Header.Set(ahttp.HeaderCookie, cookieValue)

	e.ServeHTTP(w11, r11)

	resp11 := w11.Result()
	body11 := getResponseBody(resp11)
	assert.NotNil(t, resp11)
	assert.Equal(t, http.StatusOK, resp11.StatusCode)
	assert.Equal(t, "text/plain; charset=utf-8", resp11.Header.Get(ahttp.HeaderContentType))
	assert.Equal(t, "Data have been recevied successfully", body11)

	// Request 12 JSON request
	jsonBytes := []byte(`{
	"first_name":"My firstname",
	"last_name": "My lastname",
	"email": "email@myemail.com",
	"number": 8253645635463
}`)

	r12 := httptest.NewRequest("POST", "http://localhost:8080/json-submit",
		bytes.NewReader(jsonBytes))
	r12.Header.Set(ahttp.HeaderContentType, ahttp.ContentTypeJSON.String())
	r12.Header.Set("X-Anti-CSRF-Token", secretstr)
	r12.Header.Set(ahttp.HeaderCookie, cookieValue)
	w12 := httptest.NewRecorder()
	e.ServeHTTP(w12, r12)

	resp12 := w12.Result()
	body12 := getResponseBody(resp12)
	assert.NotNil(t, resp12)
	assert.Equal(t, http.StatusOK, resp12.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp12.Header.Get(ahttp.HeaderContentType))
	assert.True(t, strings.Contains(body12, `"success":true`))

	// Request 13 domain not found
	r13 := httptest.NewRequest("GET", "http://localhost:7070/index.html", nil)
	w13 := httptest.NewRecorder()
	e.ServeHTTP(w13, r13)

	resp13 := w13.Result()
	assert.Equal(t, 404, resp13.StatusCode)
	assert.True(t, strings.Contains(resp13.Status, "Not Found"))

	appBaseDir = ""
}

func TestEngineGzipHeaders(t *testing.T) {
	cfg, _ := config.ParseString("")
	e := newEngine(cfg)

	req := httptest.NewRequest("GET", "http://localhost:8080/doc/v0.3/mydoc.html", nil)
	req.Header.Add(ahttp.HeaderAcceptEncoding, "gzip")
	ctx := e.prepareContext(httptest.NewRecorder(), req)
	e.wrapGzipWriter(ctx)

	assert.True(t, ctx.Req.IsGzipAccepted)
	assert.Equal(t, "gzip", ctx.Res.Header().Get(ahttp.HeaderContentEncoding))
	assert.Equal(t, "Accept-Encoding", ctx.Res.Header().Get(ahttp.HeaderVary))
	assert.False(t, isResponseBodyAllowed(199))
	assert.False(t, isResponseBodyAllowed(304))
	assert.False(t, isResponseBodyAllowed(100))
}

func getResponseBody(res *http.Response) string {
	r := res.Body
	defer ess.CloseQuietly(r)
	if strings.Contains(res.Header.Get("Content-Encoding"), "gzip") {
		r, _ = gzip.NewReader(r)
	}
	body, _ := ioutil.ReadAll(r)
	return string(body)
}
