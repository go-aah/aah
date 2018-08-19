// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"aahframework.org/essentials"
	"github.com/stretchr/testify/assert"
)

func TestServerStartHTTP(t *testing.T) {
	defer ess.DeleteFiles("webapp1.pid")

	importPath := filepath.Join(testdataBaseDir(), "webapp1")
	ts := newTestServer(t, importPath)
	defer ts.Close()

	t.Logf("Test Server URL [Server Start HTTP]: %s", ts.URL)

	go ts.app.Start()
	defer ts.app.Shutdown()

	time.Sleep(10 * time.Millisecond)
}

func TestServerStartUnix(t *testing.T) {
	defer ess.DeleteFiles("webapp1.pid")

	importPath := filepath.Join(testdataBaseDir(), "webapp1")
	ts := newTestServer(t, importPath)
	defer ts.Close()

	t.Logf("Test Server URL [Server Start Unix]: %s", ts.URL)

	ts.app.Config().SetString("server.address", "unix:/tmp/testserver")
	go ts.app.Start()
	defer ts.app.Shutdown()

	time.Sleep(10 * time.Millisecond)
}

func TestServerHTTPRedirect(t *testing.T) {
	defer ess.DeleteFiles("webapp1.pid")

	importPath := filepath.Join(testdataBaseDir(), "webapp1")
	ts1 := newTestServer(t, importPath)
	defer ts1.Close()

	t.Logf("Test Server URL [Redirect Server]: %s", ts1.URL)

	// redirect not enabled
	t.Log("redirect not enabled")
	ts1.app.startHTTPRedirect()

	// redirect enabled but port not provided
	t.Log("redirect enabled but port not provided")
	ts1.app.Config().SetBool("server.ssl.redirect_http.enable", true)
	ts1.app.Config().SetString("server.port", "8443")
	go ts1.app.startHTTPRedirect()
	defer ts1.app.shutdownRedirectServer()

	// redirect enabled with port
	ts2 := newTestServer(t, importPath)
	defer ts2.Close()

	t.Logf("Test Server URL [Redirect Server]: %s", ts2.URL)

	t.Log("redirect enabled with port")
	ts2.app.Config().SetString("server.ssl.redirect_http.port", "8080")
	go ts2.app.startHTTPRedirect()
	defer ts2.app.shutdownRedirectServer()

	// send request to redirect server
	t.Log("send request to redirect server")
	resp, err := http.Get("http://localhost:8080/contact-us.html?utm_source=footer")
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 307, resp.StatusCode)
	assert.True(t, strings.Contains(responseBody(resp), "Temporary Redirect"))
}
