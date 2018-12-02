// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ws

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"aahframe.work/ahttp"
	"aahframe.work/ainsp"
	"aahframe.work/config"
	"aahframe.work/essentials"
	"aahframe.work/log"
	"aahframe.work/router"
	"aahframe.work/security"
	"aahframe.work/vfs"

	gws "github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/stretchr/testify/assert"
)

func TestEngineWSClient(t *testing.T) {
	cfgStr := `
    server {
      websocket {
        enable = true

        origin {
          check = true

          whitelist = [
            "localhost:8080"
          ]
        }
      }
    }
  `

	ts := createWSTestServer(t, cfgStr, "routes.conf")
	wsURL := strings.Replace(ts.ts.URL, "http", "ws", -1)

	// test cases
	testcases := []struct {
		label    string
		wsURL    string
		opCode   gws.OpCode
		content  []byte
		customID bool
	}{
		{
			label:   "WS Text msg test",
			wsURL:   fmt.Sprintf("%s/ws/text", wsURL),
			opCode:  gws.OpText,
			content: []byte("Hi welcome to aah ws test text msg"),
		},
		{
			label:   "WS Binary msg test",
			wsURL:   fmt.Sprintf("%s/ws/binary", wsURL),
			opCode:  gws.OpBinary,
			content: []byte("Hi welcome to aah ws test binary msg"),
		},
		{
			label:   "WS JSON msg test",
			wsURL:   fmt.Sprintf("%s/ws/json", wsURL),
			opCode:  gws.OpText,
			content: []byte(`{"content":"Hello JSON","value":23436723}`),
		},
		{
			label:    "WS XML msg test",
			wsURL:    fmt.Sprintf("%s/ws/xml", wsURL),
			opCode:   gws.OpText,
			content:  []byte(`<Msg><Content>Hello JSON</Content><Value>23436723</Value></Msg>`),
			customID: true,
		},
		{
			label: "WS preconnect abort test",
			wsURL: fmt.Sprintf("%s/ws/text?abort=true", wsURL),
		},
		{
			label: "WS disconnect test",
			wsURL: fmt.Sprintf("%s/ws/text?disconnect=true", wsURL),
		},
		{
			label: "WS not exists test",
			wsURL: fmt.Sprintf("%s/ws/notexists", wsURL),
		},
		{
			label: "WS no target test",
			wsURL: fmt.Sprintf("%s/ws/notarget", wsURL),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.label, func(t *testing.T) {
			if tc.customID {
				ts.wse.SetIDGenerator(func(ctx *Context) string {
					return ess.RandomString(32)
				})
			}
			conn, _, _, err := gws.Dial(context.Background(), tc.wsURL)
			if err != nil {
				switch {
				case strings.HasSuffix(err.Error(), "401"),
					strings.HasSuffix(err.Error(), "404"):
					return
				}
			}
			assert.Nil(t, err, "connection failure")

			err = wsutil.WriteClientMessage(conn, tc.opCode, tc.content)
			if err != nil {
				return
			}

			b, op, err := wsutil.ReadServerData(conn)
			if err != nil && (err == io.EOF || strings.Contains(err.Error(), "reset by peer")) {
				return
			}
			assert.Nil(t, err)
			assert.Equal(t, tc.opCode, op)
			assert.Equal(t, tc.content, b)
		})
	}

}

func TestEngineWSErrors(t *testing.T) {
	cfgStr := `
    server {
      websocket {
        enable = true
      }
    }
  `

	ts := createWSTestServer(t, cfgStr, "routes-multi.conf")

	resp, err := http.Get(ts.ts.URL + "/ws/text")
	assert.Nil(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	// 405 Method Not Allowed
	w := httptest.NewRecorder()
	r := httptest.NewRequest(ahttp.MethodPost, "http://localhost:8080/ws/text", strings.NewReader("error=notsupported"))
	ts.wse.Handle(w, r)
	assert.Equal(t, "405 Method Not Allowed", w.Body.String())
}

type testServer struct {
	ts  *httptest.Server
	wse *Engine
}

type app struct {
	cfg *config.Config
	r   *router.Router
	l   log.Loggerer
}

func (a *app) Config() *config.Config             { return a.cfg }
func (a *app) Router() *router.Router             { return a.r }
func (a *app) Log() log.Loggerer                  { return a.l }
func (a *app) VFS() *vfs.VFS                      { return nil }
func (a *app) SecurityManager() *security.Manager { return nil }

func createWSTestServer(t *testing.T, cfgStr, routeFile string) *testServer {
	cfg, _ := config.ParseString(cfgStr)
	wse := newEngine(t, cfg, routeFile)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(ahttp.HeaderOrigin) == "" {
			r.Header.Set(ahttp.HeaderOrigin, fmt.Sprintf("http://%s", ahttp.Host(r)))
		}
		wse.Handle(w, r)
	}))
	assert.NotNil(t, ts)

	t.Logf("Test WS server running here : %s", ts.URL)

	return &testServer{ts: ts, wse: wse}
}

func newEngine(t *testing.T, cfg *config.Config, routeFile string) *Engine {
	l, err := log.New(cfg)
	assert.Nil(t, err)

	l.SetWriter(ioutil.Discard)

	app := &app{cfg: cfg, l: l}
	r, err := router.NewWithApp(app, filepath.Join(testdataBaseDir(), routeFile))
	assert.Nil(t, err)

	app.r = r
	wse, err := New(app)
	assert.Nil(t, err)
	assert.NotNil(t, wse.app)

	// Adding events
	addWebSocketEvents(t, wse)

	// Add WebSocket
	addWebSocket(t, wse)

	return wse
}

func addWebSocketEvents(t *testing.T, wse *Engine) {
	wse.OnPreConnect(func(eventName string, ctx *Context) {
		t.Logf("Event: %s called", eventName)
		assert.Equal(t, EventOnPreConnect, eventName)
		assert.NotNil(t, ctx)

		if ctx.Req.QueryValue("abort") == "true" {
			ctx.Abort(http.StatusUnauthorized) // WS request stops there
		}
	})
	wse.OnPostConnect(func(eventName string, ctx *Context) {
		t.Logf("Event: %s called", eventName)
		assert.Equal(t, EventOnPostConnect, eventName)
		assert.NotNil(t, ctx)
		if ctx.Req.QueryValue("disconnect") == "true" {
			ctx.Disconnect()
		}
	})
	wse.OnPostDisconnect(func(eventName string, ctx *Context) {
		t.Logf("Event: %s called", eventName)
		assert.Equal(t, EventOnPostDisconnect, eventName)
		assert.NotNil(t, ctx)
	})
	wse.OnError(func(eventName string, ctx *Context) {
		t.Logf("Event: %s called: %s", eventName, ctx.ErrorReason())
		assert.Equal(t, EventOnError, eventName)
		assert.NotNil(t, ctx)
	})
}

func addWebSocket(t *testing.T, wse *Engine) {
	wse.AddWebSocket((*testWebSocket)(nil), []*ainsp.Method{
		{Name: "Text"},
		{Name: "Binary", Parameters: []*ainsp.Parameter{{Name: "encoding", Type: reflect.TypeOf((*string)(nil))}}},
		{Name: "JSON"},
		{Name: "XML"},
	})
}

type testWebSocket struct {
	*Context
}

func (e *testWebSocket) Text() {
	for {
		str, err := e.ReadText()
		if err != nil {
			e.Log().Error(err)
			IsDisconnected(err)
			return
		}

		if err := e.ReplyText(str); err != nil {
			e.Log().Error(err)
			return
		}
	}
}

func (e *testWebSocket) Binary(encoding string) {
	t := &testing.T{}
	ip := e.Req.ClientIP()
	assert.True(t, ip != "")

	assert.True(t, e.Req.String() != "")

	for {
		b, err := e.ReadBinary()
		if err != nil {
			e.Log().Error(err)
			return
		}

		if err := e.ReplyBinary(b); err != nil {
			e.Log().Error(err)
			return
		}
	}
}

func (e *testWebSocket) JSON() {
	t := &testing.T{}
	ip := e.Req.ClientIP()
	assert.True(t, ip != "")

	assert.True(t, e.Req.String() != "")

	type msg struct {
		Content string `json:"content"`
		Value   int    `json:"value"`
	}
	for {
		var m msg
		if err := e.ReadJSON(&m); err != nil {
			e.Log().Error(err)
			return
		}

		if err := e.ReplyJSON(m); err != nil {
			e.Log().Error(err)
			return
		}
	}
}

func (e *testWebSocket) XML() {
	t := &testing.T{}
	assert.Equal(t, "", e.Req.QueryValue("notexists"))
	assert.True(t, len(e.Req.QueryArrayValue("encoding")) == 0)
	assert.Equal(t, "", e.Req.PathValue("discussion"))

	type msg struct {
		XMLName xml.Name `xml:"Msg"`
		Content string
		Value   int
	}
	for {
		var m msg
		if err := e.ReadXML(&m); err != nil {
			e.Log().Error(err)
			return
		}

		if err := e.ReplyXML(m); err != nil {
			e.Log().Error(err)
			return
		}
	}
}

func testdataBaseDir() string {
	wd, _ := os.Getwd()
	if idx := strings.Index(wd, "testdata"); idx > 0 {
		wd = wd[:idx]
	}
	return filepath.Join(wd, "testdata")
}
