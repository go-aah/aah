// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aahframework.org/ahttp.v0"
	"aahframework.org/test.v0/assert"
)

// TestHandler ...
type TestHandler struct {
}

func (th *TestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(ahttp.HeaderContentType, "text/html; charset=utf-8")
	_, _ = w.Write([]byte("TestHandler.ServeHTTP\n"))
	_, _ = w.Write([]byte(r.Method + "--" + r.URL.Path + "\n"))
}

func TestMiddlewareToHandler(t *testing.T) {
	a := newApp()
	e := a.he

	e.Middlewares(
		ToMiddleware(thirdPartyMiddleware3),
		ToMiddleware(http.HandlerFunc(thirdPartyMiddleware2)),
		ToMiddleware(&TestHandler{}),
		ToMiddleware(thirdPartyMiddleware1),
		ToMiddleware(invaildHandlerType),
	)

	req := httptest.NewRequest(ahttp.MethodGet, "http://localhost:8080/doc/v0.3/mydoc.html", nil)
	ctx := newContext(httptest.NewRecorder(), req)

	// Execute the middleware
	e.mwChain[0].Next(ctx)

	w := ctx.Res.Unwrap().(*httptest.ResponseRecorder)
	resp := w.Result()
	body := responseBody(resp)
	t.Log(body)

	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	assert.Equal(t, "text/html; charset=utf-8", resp.Header.Get(ahttp.HeaderContentType))
	assert.True(t, strings.Contains(body, "localhost:8080--GET--/doc/v0.3/mydoc.html"))
}

func thirdPartyMiddleware1(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("thirdPartyMiddleware1\n"))
	_, _ = w.Write([]byte(r.Method + "--" + r.URL.Path + "\n"))
}

func thirdPartyMiddleware2(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("thirdPartyMiddleware2\n"))
	_, _ = w.Write([]byte(r.Host + "--" + r.Method + "--" + r.URL.Path + "\n"))
}

func thirdPartyMiddleware3(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(ahttp.HeaderContentType, "text/html; charset=utf-8")

	// doesn't make sense right!!!
	// just for testing; to differentiate the default 200 code
	w.WriteHeader(http.StatusAccepted)

	_, _ = w.Write([]byte("thirdPartyMiddleware3\n"))
	_, _ = w.Write([]byte(r.Method + "--" + r.URL.Path + "\n"))
}

func invaildHandlerType(e *Event) {
	fmt.Println("This is invaild handler type")
}
