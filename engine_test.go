// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/log.v0"
	"aahframework.org/test.v0/assert"
)

type (
	Site struct {
		*Context
	}
)

func (s *Site) GetInvolved() {
	s.Reply().Text("GetInvolved action")
}

func (s *Site) ContributeCode() {
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
	assert.Equal(t, 5, e.gzipLevel)
	assert.True(t, e.isRequestIDEnabled)
	assert.True(t, e.isGzipEnabled)
	assert.NotNil(t, e.ctxPool)
	assert.NotNil(t, e.bufPool)
	assert.NotNil(t, e.reqPool)

	req := e.getRequest()
	ctx := e.getContext()
	ctx.Req = req
	assert.NotNil(t, ctx)
	assert.NotNil(t, req)
	assert.NotNil(t, ctx.Req)
	e.putContext(ctx)

	buf := e.getBuffer()
	assert.NotNil(t, buf)
	e.putBuffer(buf)
}

func TestEngineServeHTTP(t *testing.T) {
	// App Config
	cfgDir := filepath.Join(getTestdataPath(), appConfigDir())
	err := initConfig(cfgDir)
	assert.Nil(t, err)
	assert.NotNil(t, AppConfig())

	// Router
	err = initRoutes(cfgDir, AppConfig())
	assert.Nil(t, err)
	assert.NotNil(t, AppRouter())

	// Security
	err = initSecurity(cfgDir, AppConfig())
	assert.Nil(t, err)

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
	})

	// Middlewares
	Middlewares(ToMiddleware(testEngineMiddleware))

	// Engine
	e := newEngine(AppConfig())

	// Request 1
	r1 := httptest.NewRequest("GET", "http://localhost:8080/doc/v0.3/mydoc.html", nil)
	w1 := httptest.NewRecorder()
	e.ServeHTTP(w1, r1)

	resp1 := w1.Result()
	assert.Equal(t, 404, resp1.StatusCode)
	assert.Equal(t, "Not Found", resp1.Status)
	assert.Equal(t, "aah-go-server", resp1.Header.Get(ahttp.HeaderServer))

	// Request 2
	r2 := httptest.NewRequest("GET", "http://localhost:8080/get-involved.html", nil)
	w2 := httptest.NewRecorder()
	e.ServeHTTP(w2, r2)

	resp2 := w2.Result()
	body2, _ := ioutil.ReadAll(resp2.Body)
	assert.Equal(t, 200, resp2.StatusCode)
	assert.Equal(t, "OK", resp2.Status)
	assert.Equal(t, "GetInvolved action", string(body2))
	assert.Equal(t, "test engine middleware", resp2.Header.Get("X-Custom-Name"))

	// Request 3
	r3 := httptest.NewRequest("GET", "http://localhost:8080/contribute-to-code.html", nil)
	w3 := httptest.NewRecorder()
	e.ServeHTTP(w3, r3)

	resp3 := w3.Result()
	body3, _ := ioutil.ReadAll(resp3.Body)
	assert.Equal(t, 500, resp3.StatusCode)
	assert.Equal(t, "Internal Server Error", resp3.Status)
	assert.True(t, strings.Contains(string(body3), "panic flow testing"))
}

func TestEngineGzipHeaders(t *testing.T) {
	cfg, _ := config.ParseString("")
	e := newEngine(cfg)

	req := httptest.NewRequest("GET", "http://localhost:8080/doc/v0.3/mydoc.html", nil)
	req.Header.Add(ahttp.HeaderAcceptEncoding, "gzip")
	ctx := e.prepareContext(httptest.NewRecorder(), req)

	assert.True(t, ctx.Req.IsGzipAccepted)

	e.writeGzipHeaders(ctx, false)

	assert.Equal(t, "gzip", ctx.Res.Header().Get(ahttp.HeaderContentEncoding))
	assert.Equal(t, "Accept-Encoding", ctx.Res.Header().Get(ahttp.HeaderVary))

	e.writeGzipHeaders(ctx, true)
	assert.Equal(t, "", ctx.Res.Header().Get(ahttp.HeaderContentEncoding))
	assert.Equal(t, "Accept-Encoding", ctx.Res.Header().Get(ahttp.HeaderVary))
}
