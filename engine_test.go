package ws

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"aahframework.org/ahttp.v0"
	"aahframework.org/ainsp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/log.v0"
	"aahframework.org/router.v0"
	"aahframework.org/test.v0/assert"

	gws "github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
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

	cfg, _ := config.ParseString(cfgStr)
	wse := newEngine(t, cfg)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(ahttp.HeaderOrigin) == "" {
			r.Header.Set(ahttp.HeaderOrigin, fmt.Sprintf("http://%s", ahttp.Host(r)))
		}
		wse.Handle(w, r)
	}))
	assert.NotNil(t, ts)
	t.Logf("Test WS server running here : %s", ts.URL)

	wsURL := strings.Replace(ts.URL, "http", "ws", -1)

	// test cases
	testcases := []struct {
		label   string
		wsURL   string
		opCode  gws.OpCode
		content []byte
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
			label:   "WS XML msg test",
			wsURL:   fmt.Sprintf("%s/ws/xml", wsURL),
			opCode:  gws.OpText,
			content: []byte(`<Msg><Content>Hello JSON</Content><Value>23436723</Value></Msg>`),
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
			conn, _, _, err := gws.Dial(context.Background(), tc.wsURL)
			if err != nil {
				switch {
				case strings.HasSuffix(err.Error(), "401"),
					strings.HasSuffix(err.Error(), "404"):
					return
				}
			}
			assert.FailNowOnError(t, err, "connection failure")

			err = wsutil.WriteClientMessage(conn, tc.opCode, tc.content)
			if err != nil {
				if !strings.Contains(err.Error(), "broken pipe") {
					assert.FailNowOnError(t, err, "Unable to send msg to ws server")
				}
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

func newEngine(t *testing.T, cfg *config.Config) *Engine {
	l, err := log.New(cfg)
	assert.Nil(t, err)

	r := router.New(filepath.Join(testdataBaseDir(), "routes.conf"), config.NewEmptyConfig())
	err = r.Load()
	assert.Nil(t, err)

	wse, err := New(cfg, l, r)
	assert.Nil(t, err)
	assert.NotNil(t, wse.logger)
	assert.NotNil(t, wse.cfg)

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

	str := fmt.Sprintf("%s", e.Req)
	assert.True(t, str != "")

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

	str := fmt.Sprintf("%s", e.Req)
	assert.True(t, str != "")

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
