// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org/ws source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ws

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"aahframework.org/ahttp.v0"
	"aahframework.org/ainsp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
	"aahframework.org/router.v0"

	gws "github.com/gobwas/ws"
)

const (
	// EventOnPreConnect event published right before the aah tries establish a
	// WebSocket connection with client.
	EventOnPreConnect = "OnPreConnect"

	// EventOnPostConnect event published right after the successful WebSocket
	// connection have been established with aah server.
	EventOnPostConnect = "OnPostConnect"

	// EventOnPostDisconnect event published when client disconnects either
	// gracefully, gone, network connection error, etc.
	EventOnPostDisconnect = "OnPostDisconnect"

	// EventOnError event published when any errors, auth error, failures while
	// establishing WebSocket connection, etc.
	EventOnError = "OnError"
)

// WebSocket errors
var (
	ErrOriginMismatch         = errors.New("aahws: origin mismatch")
	ErrParameterParseFailed   = errors.New("aahws: parameter parse failed")
	ErrAuthenticationFailed   = errors.New("aahws: authentication failed")
	ErrWebSocketNotFound      = errors.New("aahws: not found")
	ErrWebSocketConnectFailed = errors.New("aahws: connect failed")
)

// EventCallbackFunc func type used for all WebSocket event callback.
type EventCallbackFunc func(eventName string, ctx *Context)

// AuthCallbackFunc func type used for WebSocket authentication.
type AuthCallbackFunc func(ctx *Context) bool

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Engine type and its methods
//______________________________________________________________________________

// Engine struct holds the implementation of managing WebSocket for aah
// framework.
type Engine struct {
	checkOrigin      bool
	originWhitelist  []*url.URL
	cfg              *config.Config
	registry         *ainsp.TargetRegistry
	doAuth           AuthCallbackFunc
	onPreConnect     EventCallbackFunc
	onPostConnect    EventCallbackFunc
	onPostDisconnect EventCallbackFunc
	onError          EventCallbackFunc
	logger           log.Loggerer
}

// AddWebSocket method adds the given WebSocket implementation into engine.
func (e *Engine) AddWebSocket(t interface{}, methods []*ainsp.Method) {
	e.registry.Add(t, methods)
}

// OnPreConnect method sets WebSocket `OnPreConnect` event callback into
// WebSocket engine.
//
// Event published before each WebSocket connection been established.
func (e *Engine) OnPreConnect(ecf EventCallbackFunc) {
	e.onPreConnect = ecf
}

// OnPostConnect method sets WebSocket `OnPostConnect` event callback into
// WebSocket engine.
//
// Event published after each WebSocket connection successfully established.
func (e *Engine) OnPostConnect(ecf EventCallbackFunc) {
	e.onPostConnect = ecf
}

// OnPostDisconnect method sets WebSocket `OnPostDisconnect` event callback into
// WebSocket engine.
//
// Event published after each WebSocket connection is disconncted from aah server
// such as client disconnct, connection interrupted, etc.
func (e *Engine) OnPostDisconnect(ecf EventCallbackFunc) {
	e.onPostDisconnect = ecf
}

// OnError method sets WebSocket `OnError` event callback into
// WebSocket engine.
//
// Event published for mismatch origin, action parameter parse error,
// authentication failure, websocket initial connection failure,
// websocket not found.
func (e *Engine) OnError(ecf EventCallbackFunc) {
	e.onError = ecf
}

// SetAuthCallback method sets the WebSocket authentication callback. It gets
// called for every WebSocket connection.
//
// Authentication callback function should return true for success otherwise false.
func (e *Engine) SetAuthCallback(ac AuthCallbackFunc) {
	e.doAuth = ac
}

// Connect method primarly does upgrades HTTP connection into WebSocket
// connection.
//
// Along with Check Origin, Authentication Callback and aah WebSocket events
// such as `OnPreConnect`, `OnPostConnect`, `OnPostDisconnect` and `OnError`.
func (e *Engine) Connect(w http.ResponseWriter, r *http.Request, route *router.Route, pathParams ahttp.PathParams) (*Context, error) {
	ctx := e.newContext(r, route, pathParams)

	// Check Origin
	if e.checkOrigin && !e.isSameOrigin(ctx) {
		ctx.Log().Error("WS: Origin mismatch")
		ctx.reason = ErrOriginMismatch
		e.publishOnErrorEvent(ctx)
		e.ReplyError(w, http.StatusBadRequest)
		return nil, ErrOriginMismatch
	}

	// Check WebSocket exists and prepare it.
	if err := ctx.setTarget(route.Target, route.Action); err != nil {
		ctx.reason = err
		e.publishOnErrorEvent(ctx)
		return nil, err
	}

	// Parse action parameters
	if err := ctx.parseParameters(); err != nil {
		ctx.Log().Errorf("WS: Parameters error %v", err)
		ctx.reason = ErrParameterParseFailed
		e.publishOnErrorEvent(ctx)
		e.ReplyError(w, http.StatusBadRequest)
		return nil, ErrParameterParseFailed
	}

	// Do authentication
	if e.doAuth != nil {
		if !e.doAuth(ctx) {
			ctx.Log().Errorf("WS: Authentication failed for WebSocket connection: %s", ctx.Req)
			ctx.reason = ErrAuthenticationFailed
			e.publishOnErrorEvent(ctx)
			e.ReplyError(w, http.StatusUnauthorized)
			return nil, ErrAuthenticationFailed
		}
	}

	if e.onPreConnect != nil {
		e.onPreConnect(EventOnPreConnect, ctx)
	}

	r.Method = ahttp.MethodGet // back to GET for upgrade
	conn, _, hs, err := gws.UpgradeHTTP(r, w, ctx.Header)
	if err != nil {
		ctx.Log().Errorf("WS: Unable establish a WebSocket connection for '%s'", ctx.Req.Path)
		ctx.reason = ErrWebSocketConnectFailed
		e.publishOnErrorEvent(ctx)
		return nil, err
	}

	// WebSocket connection successful
	ctx.hs = hs
	ctx.Conn = conn

	if e.onPostConnect != nil {
		e.onPostConnect(EventOnPostConnect, ctx)
	}

	return ctx, nil
}

// CallAction method calls the defined action for the WebSocket.
func (e *Engine) CallAction(ctx *Context) {
	ctx.callAction()

	if e.onPostDisconnect != nil {
		e.onPostDisconnect(EventOnPostDisconnect, ctx)
	}
}

// ReplyError method writes HTTP error response.
func (e *Engine) ReplyError(w http.ResponseWriter, errCode int) {
	writeHTTPError(w, errCode, fmt.Sprintf("%d %s", errCode, http.StatusText(errCode)))
}

// Log method provides logging methods at WebSocket engine.
func (e *Engine) Log() log.Loggerer {
	return e.logger
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Engine Unexported methods
//______________________________________________________________________________

func (e *Engine) newContext(r *http.Request, route *router.Route, pathParams ahttp.PathParams) *Context {
	ctx := &Context{
		e:      e,
		Header: make(http.Header),
		route:  route,
		Req: &Request{
			ID:          ess.NewGUID(),
			Host:        ahttp.IdentifyHost(r),
			Path:        r.URL.Path,
			Header:      r.Header,
			pathParams:  pathParams,
			queryParams: r.URL.Query(),
			raw:         r,
		},
	}
	return ctx
}

func (e *Engine) isSameOrigin(ctx *Context) bool {
	origin := ctx.Req.Header.Get(ahttp.HeaderOrigin)
	if ess.IsStrEmpty(origin) {
		ctx.Log().Errorf("WS: No origin header value: %s", ctx.Req.Header.Get(ahttp.HeaderOrigin))
		return false
	}

	o, err := url.Parse(origin)
	if err != nil {
		ctx.Log().Errorf("WS: Unable to parse Origin header URL: %s", ctx.Req.Header.Get(ahttp.HeaderOrigin))
		return false
	}

	// Check whitelisted origins
	for _, u := range e.originWhitelist {
		if strings.EqualFold(u.Host, o.Host) {
			return true
		}
	}

	return strings.EqualFold(ctx.Req.Host, o.Host)
}

func (e *Engine) publishOnErrorEvent(ctx *Context) {
	if e.onError != nil {
		e.onError(EventOnError, ctx)
	}
}
