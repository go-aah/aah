package ws

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/http/httptest"
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

	// Adding events
	addWebSocketEvents(t, wse)

	// Adding Authentication
	setAuthCallback(t, wse, true)

	// Add WebSocket
	addWebSocket(t, wse)

	ts := createTestServer(t, wse)
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
			wsURL:   fmt.Sprintf("%s?encoding=text", wsURL),
			opCode:  gws.OpText,
			content: []byte("Hi welcome to aah ws test text msg"),
		},
		{
			label:   "WS Binary msg test",
			wsURL:   fmt.Sprintf("%s?encoding=binary", wsURL),
			opCode:  gws.OpBinary,
			content: []byte("Hi welcome to aah ws test binary msg"),
		},
		{
			label:   "WS JSON msg test",
			wsURL:   fmt.Sprintf("%s?encoding=json", wsURL),
			opCode:  gws.OpText,
			content: []byte(`{"content":"Hello JSON","value":23436723}`),
		},

		{
			label:   "WS XML msg test",
			wsURL:   fmt.Sprintf("%s?encoding=xml", wsURL),
			opCode:  gws.OpText,
			content: []byte(`<Msg><Content>Hello JSON</Content><Value>23436723</Value></Msg>`),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.label, func(t *testing.T) {
			conn, _, _, err := gws.Dial(context.Background(), tc.wsURL)
			assert.FailNowOnError(t, err, "connection failure")

			err = wsutil.WriteClientMessage(conn, tc.opCode, tc.content)
			assert.FailNowOnError(t, err, "Unable to send msg to ws server")
			b, op, err := wsutil.ReadServerData(conn)
			assert.Nil(t, err)
			assert.Equal(t, tc.opCode, op)
			assert.Equal(t, tc.content, b)
		})
	}
}

func newEngine(t *testing.T, cfg *config.Config) *Engine {
	l, err := log.New(cfg)
	assert.Nil(t, err)

	wse, err := New(cfg, l)
	assert.Nil(t, err)
	assert.NotNil(t, wse.logger)
	assert.NotNil(t, wse.cfg)

	return wse
}

func addWebSocketEvents(t *testing.T, wse *Engine) {
	wse.OnPreConnect(func(eventName string, ctx *Context) {
		t.Logf("Event: %s called", eventName)
		assert.Equal(t, EventOnPreConnect, eventName)
		assert.NotNil(t, ctx)
	})
	wse.OnPostConnect(func(eventName string, ctx *Context) {
		t.Logf("Event: %s called", eventName)
		assert.Equal(t, EventOnPostConnect, eventName)
		assert.NotNil(t, ctx)
	})
	wse.OnPostDisconnect(func(eventName string, ctx *Context) {
		t.Logf("Event: %s called", eventName)
		assert.Equal(t, EventOnPostDisconnect, eventName)
		assert.NotNil(t, ctx)
	})
	wse.OnError(func(eventName string, ctx *Context) {
		t.Logf("Event: %s called", eventName)
		assert.Equal(t, EventOnError, eventName)
		assert.NotNil(t, ctx)
	})
}

func setAuthCallback(t *testing.T, wse *Engine, mode bool) {
	wse.SetAuthCallback(func(ctx *Context) bool {
		assert.NotNil(t, ctx)
		t.Logf("Authentication callback called for %s", ctx.Req.Path)
		ctx.Header.Set("X-WS-Test-Auth", "Success")
		// success auth
		return mode
	})
}

func createTestServer(t *testing.T, wse *Engine) *httptest.Server {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route := &router.Route{Target: "testWebSocket"}
		switch r.URL.Query().Get("encoding") {
		case "text":
			route.Action = "Text"
		case "binary":
			route.Action = "Binary"
		case "json":
			route.Action = "JSON"
		case "xml":
			route.Action = "XML"
		}

		r.Header.Set(ahttp.HeaderOrigin, r.URL.String())

		ctx, err := wse.Connect(w, r, route, ahttp.PathParams{})
		if err != nil {
			if err == ErrWebSocketNotFound {
				wse.ReplyError(w, http.StatusNotFound)
			}
		}

		wse.CallAction(ctx)
	})

	return httptest.NewServer(handler)
}

func addWebSocket(t *testing.T, wse *Engine) {
	wse.AddWebSocket((*testWebSocket)(nil), []*ainsp.Method{
		{Name: "Text", Parameters: []*ainsp.Parameter{{Name: "encoding", Type: reflect.TypeOf((*string)(nil))}}},
		{Name: "Binary", Parameters: []*ainsp.Parameter{{Name: "encoding", Type: reflect.TypeOf((*string)(nil))}}},
		{Name: "JSON", Parameters: []*ainsp.Parameter{{Name: "encoding", Type: reflect.TypeOf((*string)(nil))}}},
		{Name: "XML"},
	})
}

type testWebSocket struct {
	*Context
}

func (e *testWebSocket) Text(encoding string) {
	for {
		str, err := e.ReadText()
		if err != nil {
			e.Log().Error(err)
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

func (e *testWebSocket) JSON(encoding string) {
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
	assert.Equal(t, "xml", e.Req.QueryValue("encoding"))
	assert.Equal(t, "", e.Req.QueryValue("notexists"))
	assert.True(t, len(e.Req.QueryArrayValue("encoding")) > 0)
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
