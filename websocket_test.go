package aah

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"aahframework.org/ainsp.v0"
	"aahframework.org/test.v0/assert"
	"aahframework.org/ws.v0"
	gws "github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

//
// Test WebSocket Engine
//

func TestWebSocketEngine(t *testing.T) {
	importPath := filepath.Join(testdataBaseDir(), "webapp1")
	ts, err := newTestServer(t, importPath)
	assert.Nil(t, err)
	defer ts.Close()

	t.Logf("Test Server URL [Engine Handling]: %s", ts.URL)

	ts.app.AddWebSocket((*testWebSocket)(nil), []*ainsp.Method{
		{Name: "Text", Parameters: []*ainsp.Parameter{{Name: "encoding", Type: reflect.TypeOf((*string)(nil))}}},
		{Name: "Binary", Parameters: []*ainsp.Parameter{{Name: "encoding", Type: reflect.TypeOf((*string)(nil))}}},
	})

	addWebSocketEvents(t, ts.app.wse)
	setAuthCallback(t, ts.app.wse, true)
	wsURL := strings.Replace(ts.URL, "http", "ws", -1)
	t.Logf("Test WebSocket URL: %s", wsURL)

	// test text msg
	t.Log("test text msg")
	conn, _, _, err := gws.Dial(context.Background(), fmt.Sprintf("%s/ws/text", wsURL))
	assert.FailNowOnError(t, err, "connection failure")

	testText1 := "Hi welcome to aah ws test 1"
	err = wsutil.WriteClientText(conn, []byte(testText1))
	assert.FailNowOnError(t, err, "Unable to send text msg to server")
	b, err := wsutil.ReadServerText(conn)
	assert.Nil(t, err)
	assert.Equal(t, testText1+" from server", string(b))
	_ = conn.Close()

	// test binary message
	t.Log("test binary message")
	conn, _, _, err = gws.Dial(context.Background(), fmt.Sprintf("%s/ws/binary", wsURL))
	assert.FailNowOnError(t, err, "connection failure")

	testBin1 := []byte("Hi welcome to aah ws test 1")
	err = wsutil.WriteClientBinary(conn, testBin1)
	assert.FailNowOnError(t, err, "Unable to send binary msg to server")
	b, err = wsutil.ReadServerBinary(conn)
	assert.Nil(t, err)
	assert.Equal(t, append(testBin1, []byte(" from server")...), b)
	_ = conn.Close()

	// ws route not found
	t.Log("ws route not found")
	_, _, _, err = gws.Dial(context.Background(), fmt.Sprintf("%s/ws/notexists", wsURL))
	assert.Equal(t, "unexpected HTTP response status: 404", err.Error())

	// ws no target found
	t.Log("ws no target found")
	_, _, _, err = gws.Dial(context.Background(), fmt.Sprintf("%s/ws/notarget", wsURL))
	assert.Equal(t, "unexpected HTTP response status: 404", err.Error())
}

func addWebSocketEvents(t *testing.T, wse *ws.Engine) {
	wse.OnPreConnect(func(eventName string, ctx *ws.Context) {
		t.Logf("Event: %s called", eventName)
		assert.Equal(t, ws.EventOnPreConnect, eventName)
		assert.NotNil(t, ctx)
	})
	wse.OnPostConnect(func(eventName string, ctx *ws.Context) {
		t.Logf("Event: %s called", eventName)
		assert.Equal(t, ws.EventOnPostConnect, eventName)
		assert.NotNil(t, ctx)
	})
	wse.OnPostDisconnect(func(eventName string, ctx *ws.Context) {
		t.Logf("Event: %s called", eventName)
		assert.Equal(t, ws.EventOnPostDisconnect, eventName)
		assert.NotNil(t, ctx)
	})
	wse.OnError(func(eventName string, ctx *ws.Context) {
		t.Logf("Event: %s called", eventName)
		assert.Equal(t, ws.EventOnError, eventName)
		assert.NotNil(t, ctx)
	})
}

func setAuthCallback(t *testing.T, wse *ws.Engine, mode bool) {
	wse.SetAuthCallback(func(ctx *ws.Context) bool {
		assert.NotNil(t, ctx)
		t.Logf("Authentication callback called for %s", ctx.Req.Path)
		ctx.Header.Set("X-WS-Test-Auth", "Success")
		// success auth
		return mode
	})
}

type testWebSocket struct {
	*ws.Context
}

func (e *testWebSocket) Text(encoding string) {
	for {
		str, err := e.ReadText()
		if err != nil {
			e.Log().Error(err)
			return
		}

		if err := e.ReplyText(str + " from server"); err != nil {
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

		b = append(b, []byte(" from server")...)

		if err := e.ReplyBinary(b); err != nil {
			e.Log().Error(err)
			return
		}
	}
}
