// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"aahframe.work/ahttp"
	"aahframe.work/ainsp"
	"aahframe.work/config"
	"aahframe.work/essentials"
	"aahframe.work/log"
	"github.com/stretchr/testify/assert"
)

func TestAahApp(t *testing.T) {
	importPath := filepath.Join(testdataBaseDir(), "webapp1")
	ts := newTestServer(t, importPath)
	defer ts.Close()

	t.Logf("Test Server URL: %s", ts.URL)

	// Do not follow redirect
	if http.DefaultClient.CheckRedirect == nil {
		http.DefaultClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	// GET - /index.html or /
	t.Log("GET - /index.html or /")
	req, err := http.NewRequest(ahttp.MethodGet, ts.URL+"?lang=en", nil)
	assert.Nil(t, err)
	req.Header.Add(ahttp.HeaderAcceptEncoding, "gzip, deflate, sdch, br")
	result := fireRequest(t, req)
	assert.Equal(t, 200, result.StatusCode)
	assert.NotNil(t, result.Header)
	assert.Equal(t, "text/html; charset=utf-8", result.Header.Get(ahttp.HeaderContentType))
	assert.Equal(t, "SAMEORIGIN", result.Header.Get(ahttp.HeaderXFrameOptions))
	assert.Equal(t, "nosniff", result.Header.Get(ahttp.HeaderXContentTypeOptions))
	assert.Equal(t, "Before Called successfully", result.Header.Get("X-Before-Interceptor"))
	assert.Equal(t, "After Called successfully", result.Header.Get("X-After-Interceptor"))
	assert.Equal(t, "Finally Called successfully", result.Header.Get("X-Finally-Interceptor"))
	assert.True(t, strings.Contains(result.Body, "Test Application webapp1 Yes it works!!!"))
	assert.True(t, strings.Contains(result.Body, "aah framework web application"))

	// GET - /get-text.html
	t.Log("GET - /get-text.html")
	req, err = http.NewRequest(ahttp.MethodGet, ts.URL+"/get-text.html", nil)
	assert.Nil(t, err)
	req.Header.Add(ahttp.HeaderAcceptEncoding, "gzip, deflate, sdch, br")
	result = fireRequest(t, req)
	assert.Equal(t, 200, result.StatusCode)
	assert.NotNil(t, result.Header)
	assert.Equal(t, "SAMEORIGIN", result.Header.Get(ahttp.HeaderXFrameOptions))
	assert.Equal(t, "nosniff", result.Header.Get(ahttp.HeaderXContentTypeOptions))
	assert.Equal(t, "BeforeText Called successfully", result.Header.Get("X-Beforetext-Interceptor"))
	assert.Equal(t, "AfterText Called successfully", result.Header.Get("X-Aftertext-Interceptor"))
	assert.Equal(t, "FinallyText Called successfully", result.Header.Get("X-Finallytext-Interceptor"))
	assert.True(t, strings.Contains(result.Body, "This is text render response"))

	// Redirect - /test-redirect.html
	t.Log("Redirect - /test-redirect.html")
	req, err = http.NewRequest(ahttp.MethodGet, ts.URL+"/test-redirect.html", nil)
	assert.Nil(t, err)
	result = fireRequest(t, req)
	assert.Equal(t, 302, result.StatusCode)
	assert.NotNil(t, result.Header)
	assert.True(t, result.Header.Get(ahttp.HeaderLocation) != "")
	assert.True(t, strings.Contains(result.Body, "Found"))

	// Redirect - /test-redirect.html?mode=text_get
	t.Log("Redirect - /test-redirect.html?mode=text_get")
	req, err = http.NewRequest(ahttp.MethodGet, ts.URL+"/test-redirect.html?mode=text_get", nil)
	assert.Nil(t, err)
	result = fireRequest(t, req)
	assert.Equal(t, 302, result.StatusCode)
	assert.NotNil(t, result.Header)
	hdrLocation := result.Header.Get(ahttp.HeaderLocation)
	assert.True(t, hdrLocation != "")
	assert.True(t, strings.Contains(hdrLocation, "Param2"))
	assert.True(t, strings.Contains(result.Body, "Found"))

	// Redirect - /test-redirect.html?mode=status
	t.Log("Redirect - /test-redirect.html?mode=status")
	req, err = http.NewRequest(ahttp.MethodGet, ts.URL+"/test-redirect.html?mode=status", nil)
	assert.Nil(t, err)
	result = fireRequest(t, req)
	assert.Equal(t, 307, result.StatusCode)
	assert.NotNil(t, result.Header)
	assert.True(t, result.Header.Get(ahttp.HeaderLocation) != "")
	assert.True(t, strings.Contains(result.Body, "Temporary Redirect"))

	// Form Submit - /form-submit - Anti-CSRF nicely guarded the form request :)
	t.Log("Form Submit - /form-submit - Anti-CSRF nicely guarded the form request :)")
	form := url.Values{}
	form.Add("id", "1000001")
	form.Add("product_name", "Test Product")
	form.Add("username", "welcome")
	form.Add("email", "welcome@welcome.com")
	req, err = http.NewRequest(ahttp.MethodPost, ts.URL+"/form-submit", strings.NewReader(form.Encode()))
	assert.Nil(t, err)
	req.Header.Set(ahttp.HeaderContentType, ahttp.ContentTypeForm.String())
	result = fireRequest(t, req)
	assert.Equal(t, 403, result.StatusCode)
	assert.NotNil(t, result.Header)
	assert.True(t, strings.Contains(result.Body, "403 Forbidden"))

	// Form Submit - /form-submit with anti_csrf_token
	t.Log("Form Submit - /form-submit with anti_csrf_token")
	secret := ts.app.SecurityManager().AntiCSRF.GenerateSecret()
	secretstr := ts.app.SecurityManager().AntiCSRF.SaltCipherSecret(secret)
	form.Add("anti_csrf_token", secretstr)
	wt := httptest.NewRecorder()
	ts.app.SecurityManager().AntiCSRF.SetCookie(wt, secret)
	cookieValue := wt.Header().Get("Set-Cookie")
	req, err = http.NewRequest(ahttp.MethodPost, ts.URL+"/form-submit", strings.NewReader(form.Encode()))
	assert.Nil(t, err)
	req.Header.Set(ahttp.HeaderContentType, ahttp.ContentTypeForm.String())
	req.Header.Set(ahttp.HeaderCookie, cookieValue)
	result = fireRequest(t, req)
	assert.Equal(t, 200, result.StatusCode)
	assert.True(t, strings.Contains(result.Body, "Data recevied successfully"))
	assert.True(t, strings.Contains(result.Body, "welcome@welcome.com"))
	assert.True(t, strings.Contains(strings.Join(result.Header["Set-Cookie"], "||"), "aah_session="))

	// CreateRecord - /create-record - JSON post request
	// This is webapp test app, send request with anti_csrf_token on HTTP header
	t.Log("CreateRecord - /create-record - JSON post request\n" +
		"This is webapp test app, send request with anti_csrf_token on HTTP header")
	jsonStr := `{
		"first_name":"My firstname",
		"last_name": "My lastname",
		"email": "email@myemail.com",
		"number": 8253645635463
	}`
	req, err = http.NewRequest(ahttp.MethodPost, ts.URL+"/create-record", strings.NewReader(jsonStr))
	assert.Nil(t, err)
	req.Header.Set(ahttp.HeaderContentType, ahttp.ContentTypeJSON.String())
	req.Header.Set("X-Anti-CSRF-Token", secretstr)
	req.Header.Set(ahttp.HeaderCookie, cookieValue)
	req.Header.Set(ahttp.HeaderXRequestID, ess.NewGUID()+"jeeva")
	result = fireRequest(t, req)
	assert.Equal(t, 200, result.StatusCode)
	assert.True(t, strings.Contains(result.Body, "JSON Payload recevied successfully"))
	assert.True(t, strings.Contains(result.Body, "8253645635463"))
	assert.True(t, strings.Contains(result.Body, "email@myemail.com"))

}

func TestAppMisc(t *testing.T) {
	importPath := filepath.Join(testdataBaseDir(), "webapp1")
	ts := newTestServer(t, importPath)
	defer ts.Close()

	t.Logf("Test Server URL [App Misc]: %s", ts.URL)

	a := ts.app

	assert.Equal(t, "web", a.Type())
	assert.Equal(t, "aah framework web application", a.Desc())
	assert.False(t, a.IsPackaged())
	a.SetPackaged(true)
	assert.True(t, a.IsPackaged())
	assert.True(t, a.IsProfileDev())
	assert.True(t, strings.Contains(strings.Join(a.AllProfiles(), " "), "prod"))

	ll := a.NewChildLogger(log.Fields{"key1": "value1"})
	assert.NotNil(t, ll)

	// simualate CLI call
	t.Log("simualate CLI call")
	a.SetBuildInfo(nil)
	a.settings.PackagedMode = false
	err := a.Init(importPath)
	assert.Nil(t, err)

	// SSL
	t.Log("SSL")
	a.SetTLSConfig(nil)
	a.Config().SetBool("server.ssl.enable", true)
	a.Config().SetBool("server.ssl.lets_encrypt.enable", true)
	err = a.settings.Refresh(a.Config())
	assert.Nil(t, err)

	// simulate import path
	t.Log("simulate import path")
	a.settings.ImportPath = "github.com/jeevatkm/noapp"
	err = a.initPath()
	// assert.True(t, strings.HasPrefix(err.Error(), "import path does not exists:"))

	// App packaged mode
	t.Log("App packaged mode")
	pa := newApp()
	l, _ := log.New(config.NewEmpty())
	pa.logger = l
	pa.SetPackaged(true)
	pa.initPath()

	// App embedded mode
	assert.False(t, pa.IsEmbeddedMode())
	pa.SetEmbeddedMode()
	assert.True(t, pa.IsEmbeddedMode())
	pa.initPath()

	// App WS engine
	assert.Nil(t, pa.WSEngine())

	// App Parse port
	assert.Equal(t, "80", pa.parsePort(""))
}

func TestAppRecover(t *testing.T) {
	importPath := filepath.Join(testdataBaseDir(), "webapp1")
	a := newTestApp(t, importPath)
	a.Log().(*log.Logger).SetWriter(ioutil.Discard)
	panicTest(a)
}

func panicTest(a *app) {
	defer a.aahRecover()
	panic("test panic")
}

func fireRequest(t *testing.T, req *http.Request) *testResult {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Errorf("Request failed %s", err)
		t.FailNow()
		return nil
	}

	return &testResult{
		StatusCode: resp.StatusCode,
		Header:     resp.Header,
		Body:       responseBody(resp),
		Raw:        resp,
	}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Test Server
//______________________________________________________________________________

func newTestServer(t *testing.T, importPath string) *testServer {
	ts := &testServer{
		app: newTestApp(t, importPath),
	}

	ts.server = httptest.NewServer(ts.app)
	ts.URL = ts.server.URL

	// Manually do it here here, for aah CLI test no issue `aah test` :)
	ts.manualInit()

	ts.DiscordLog()

	return ts
}

func newTestApp(t *testing.T, importPath string) *app {
	a := newApp()
	a.SetBuildInfo(&BuildInfo{
		BinaryName: filepath.Base(importPath),
		Date:       time.Now().Format(time.RFC3339),
		Version:    "1.0.0",
	})

	err := a.VFS().AddMount(a.VirtualBaseDir(), importPath)
	assert.Nil(t, err, "not expecting any error")

	err = a.Init(importPath)
	assert.Nil(t, err, "app init failure")

	return a
}

type testResult struct {
	StatusCode int
	Header     http.Header
	Body       string
	Raw        *http.Response
}

// TestServer provides capabilities to test aah application end-to-end.
//
// Note: after sometime I will expose this test server, I'm not fully satisfied with
// the implementation yet! Because there are short comings in the test server....
type testServer struct {
	URL    string
	app    *app
	server *httptest.Server
}

func (ts *testServer) Close() {
	ts.server.Close()
}

func (ts *testServer) DiscordLog() {
	ts.app.Log().(*log.Logger).SetWriter(ioutil.Discard)
}

func (ts *testServer) UndiscordLog() {
	ts.app.Log().(*log.Logger).SetWriter(os.Stdout)
}

// It a workaround to init required things for application, since test `webapp1`
// residing in `aahframework.org/aah.v0/testdata/webapp1`.
//
// This is not required for actual application residing in $GOPATH :)
func (ts *testServer) manualInit() {
	// adding middlewares
	ts.app.he.Middlewares(
		RouteMiddleware,
		CORSMiddleware,
		BindMiddleware,
		AntiCSRFMiddleware,
		AuthcAuthzMiddleware,
		ActionMiddleware,
	)

	// adding controller
	ts.app.AddController((*testSiteController)(nil), []*ainsp.Method{
		{Name: "Index"},
		{Name: "Text"},
		{
			Name: "Redirect",
			Parameters: []*ainsp.Parameter{
				{Name: "mode", Type: reflect.TypeOf((*string)(nil))},
			},
		},
		{
			Name: "FormSubmit",
			Parameters: []*ainsp.Parameter{
				{Name: "id", Type: reflect.TypeOf((*int)(nil))},
				{Name: "info", Type: reflect.TypeOf((**sample)(nil))},
			},
		},
		{
			Name: "CreateRecord",
			Parameters: []*ainsp.Parameter{
				{Name: "info", Type: reflect.TypeOf((**sampleJSON)(nil))},
			},
		},
		{Name: "XML"},
		{
			Name: "JSONP",
			Parameters: []*ainsp.Parameter{
				{Name: "callback", Type: reflect.TypeOf((*string)(nil))},
			},
		},
		{Name: "SecureJSON"},
		{Name: "TriggerPanic"},
		{Name: "BinaryBytes"},
		{Name: "SendFile"},
		{Name: "Cookies"},
	})

	// reset controller namespace and key
	cregistry := &ainsp.TargetRegistry{Registry: make(map[string]*ainsp.Target), SearchType: ctxPtrType}
	for k, v := range ts.app.he.registry.Registry {
		v.Namespace = ""
		cregistry.Registry[path.Base(k)] = v
	}
	ts.app.he.registry = cregistry
}

// Test types
type sample struct {
	ProductID   int    `bind:"id"`
	ProductName string `bind:"product_name"`
	Username    string `bind:"username"`
	Email       string `bind:"email"`
	Page        int    `bind:"page"`
	Count       string `bind:"count"`
}

type sampleJSON struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Number    int    `json:"number"`
}

// Test Controller

type testSiteController struct {
	*Context
}

func (s *testSiteController) Index() {
	s.Reply().HTML(Data{
		"Message":     "Welcome to aah framework - Test Application webapp1",
		"IsSubDomain": s.Subdomain(),
		"StaticRoute": s.IsStaticRoute(),
	})
}

func (s *testSiteController) Text() {
	s.Reply().Text(s.Msg("test.text.msg.render"))
}

func (s *testSiteController) Redirect(mode string) {
	switch mode {
	case "status":
		s.Reply().RedirectWithStatus(s.RouteURL("text_get"), 307)
	case "text_get":
		s.Reply().Redirect(s.RouteURLNamedArgs("text_get", map[string]interface{}{
			"param1": "param1value",
			"Param2": "Param2Value",
		}))
	default:
		s.Reply().Redirect(s.RouteURL("index"))
	}
}

func (s *testSiteController) FormSubmit(id int, info *sample) {
	s.Session().Set("session_val1", "This is my session 1 value")
	s.Reply().JSON(Data{
		"message": "Data recevied successfully",
		"success": true,
		"id":      id,
		"data":    info,
	})
}

func (s *testSiteController) CreateRecord(info *sampleJSON) {
	s.Reply().JSON(Data{
		"message": "JSON Payload recevied successfully",
		"success": true,
		"data":    info,
	})
}

func (s *testSiteController) XML() {
	s.Reply().XML(Data{
		"message": "This is XML payload result",
		"success": true,
	})
}

func (s *testSiteController) JSONP(callback string) {
	s.Reply().JSONP(sample{
		Username:    "myuser_name",
		ProductName: "JSONP product",
		ProductID:   190398398,
		Email:       "email@email.com",
		Page:        2,
		Count:       "1000",
	}, callback)
}

func (s *testSiteController) SecureJSON() {
	s.Reply().JSONSecure(sample{
		Username:    "myuser_name",
		ProductName: "JSONP product",
		ProductID:   190398398,
		Email:       "email@email.com",
		Page:        2,
		Count:       "1000",
	})
}

func (s *testSiteController) TriggerPanic() {
	if s.Req.AcceptContentType().IsEqual("application/json") {
		s.Reply().ContentType(ahttp.ContentTypeJSON.String())
	}
	panic("This panic flow test and recovery")
}

func (s *testSiteController) BinaryBytes() {
	s.Reply().
		HeaderAppend(ahttp.HeaderContentType, ahttp.ContentTypePlainText.String()).
		Binary([]byte("This is my Binary Bytes"))
}

func (s *testSiteController) SendFile() {
	s.Reply().
		Header("X-Before-Interceptor", "").
		Header(ahttp.HeaderContentType, ""). // this is just invoke the method
		Header(ahttp.HeaderContentType, "text/css").
		FileInline(filepath.Join("static", "css", "aah.css"), "aah.css")
	s.Reply().IsContentTypeSet()
}

func (s *testSiteController) Cookies() {
	s.Reply().Cookie(&http.Cookie{
		Name:     "test_cookie_1",
		Value:    "This is test cookie value 1",
		Path:     "/",
		Expires:  time.Now().AddDate(1, 0, 0),
		HttpOnly: true,
	}).
		Cookie(&http.Cookie{
			Name:     "test_cookie_2",
			Value:    "This is test cookie value 2",
			Path:     "/",
			Expires:  time.Now().AddDate(1, 0, 0),
			HttpOnly: true,
		}).Text("Hey I'm sending cookies for you :)")
}

func (s *testSiteController) HandleError(err *Error) bool {
	s.Log().Infof("we got the callbakc from error handler: %s", err)
	s.Reply().Header("X-Cntrl-ErrorHandler", "true")
	return false
}

func (s *testSiteController) Before() {
	s.Reply().Header("X-Before-Interceptor", "Before Called successfully")
	s.Log().Info("Before controller interceptor")
}

func (s *testSiteController) After() {
	s.Reply().Header("X-After-Interceptor", "After Called successfully")
	s.Log().Info("After controller interceptor")
}

func (s *testSiteController) Finally() {
	s.Reply().Header("X-Finally-Interceptor", "Finally Called successfully")
	s.Log().Info("Finally controller interceptor")
}

func (s *testSiteController) BeforeText() {
	s.Reply().Header("X-BeforeText-Interceptor", "BeforeText Called successfully")
	s.Log().Info("Before action Text interceptor")
}

func (s *testSiteController) AfterText() {
	s.Reply().Header("X-AfterText-Interceptor", "AfterText Called successfully")
	s.Log().Info("After action Text interceptor")
}

func (s *testSiteController) FinallyText() {
	s.Reply().Header("X-FinallyText-Interceptor", "FinallyText Called successfully")
	s.Log().Info("Finally action Text interceptor")
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Test util methods
//______________________________________________________________________________

func testdataBaseDir() string {
	wd, _ := os.Getwd()
	if idx := strings.Index(wd, "testdata"); idx > 0 {
		wd = wd[:idx]
	}
	return filepath.Join(wd, "testdata")
}

func responseBody(res *http.Response) string {
	body := res.Body
	defer ess.CloseQuietly(body)
	if strings.Contains(res.Header.Get(ahttp.HeaderContentEncoding), "gzip") {
		body, _ = gzip.NewReader(body)
	}
	buf := new(bytes.Buffer)
	io.Copy(buf, body)
	return buf.String()
}
