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
	"aahframework.org/valpar.v0"

	gws "github.com/gobwas/ws"
)

const (
	// EventOnPreConnect event published before connection gets upgraded to WebSocket.
	// It provides a control of accepting incoming request or reject it
	// using ctx.Abort(errorCode).
	EventOnPreConnect = "OnPreConnect"

	// EventOnPostConnect event published right after the successful WebSocket
	// connection which is established with the aah server.
	EventOnPostConnect = "OnPostConnect"

	// EventOnPostDisconnect event published right after the WebSocket client
	// got disconnected. It could have occurred due to graceful disconnect,
	// network related error, etc.
	EventOnPostDisconnect = "OnPostDisconnect"

	// EventOnError event published whenever error occurs in the lifecycle
	// such as Origin Check failed, WebSocket/WebSocket Action not found,
	// WebSocket Action parameter parse error, and WebSocket upgrade fails.
	//
	//`ctx.ErrorReason()` method can be called to know the reason for the error.
	EventOnError = "OnError"
)

// WebSocket errors
var (
	ErrOriginMismatch        = errors.New("aahws: origin mismatch")
	ErrParameterParseFailed  = errors.New("aahws: parameter parse failed")
	ErrNotFound              = errors.New("aahws: not found")
	ErrConnectFailed         = errors.New("aahws: connect failed")
	ErrAbortRequest          = errors.New("aahws: abort request")
	ErrConnectionClosed      = errors.New("aahws: connection closed")
	ErrUseOfClosedConnection = errors.New("aahws: use of closed ws connection")
)

// IDGenerator func type used to implement custom WebSocket connection ID.
// By default aah generates GUID using MongoDB Object ID algorithm.
type IDGenerator func(ctx *Context) string

// EventCallbackFunc func type used for all WebSocket event callback.
type EventCallbackFunc func(eventName string, ctx *Context)

// aah application interface for minimal purpose
type application interface {
	Config() *config.Config
	Router() *router.Router
	Log() log.Loggerer
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Engine type and its methods
//______________________________________________________________________________

// Engine struct holds the implementation of WebSocket for aah framework.
type Engine struct {
	checkOrigin      bool
	originWhitelist  []*url.URL
	app              application
	registry         *ainsp.TargetRegistry
	onPreConnect     EventCallbackFunc
	onPostConnect    EventCallbackFunc
	onPostDisconnect EventCallbackFunc
	onError          EventCallbackFunc
	idGenerator      IDGenerator
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
// Event published after each WebSocket connection is disconncted from the aah
// server.
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

// SetIDGenerator method used to set Custom ID generator func for WebSocket
// connection.
func (e *Engine) SetIDGenerator(g IDGenerator) {
	e.idGenerator = g
}

// Handle method primarily does upgrades HTTP connection into WebSocket
// connection.
//
// Along with Check Origin, aah WebSocket events such as `OnPreConnect`,
// `OnPostConnect`, `OnPostDisconnect` and `OnError`.
func (e *Engine) Handle(w http.ResponseWriter, r *http.Request) {
	domain := e.app.Router().Lookup(ahttp.Host(r))
	if domain == nil {
		e.Log().Errorf("WS: domain not found: %s", ahttp.Host(r))
		e.replyError(w, http.StatusNotFound)
		return
	}

	if r.Method != ahttp.MethodGet {
		e.Log().Errorf("WS: method not allowed: %s", r.Method)
		e.replyError(w, http.StatusMethodNotAllowed)
		return
	}

	r.Method = "WS" // for route lookup
	route, pathParams, _ := domain.Lookup(r)
	if route == nil {
		e.Log().Errorf("WS: route not found: %s", r.URL.Path)
		e.replyError(w, http.StatusNotFound)
		return
	}

	ctx, err := e.connect(w, r, route, pathParams)
	if err != nil {
		if err == ErrNotFound {
			e.Log().Errorf("WS: route not found: %s", r.URL.Path)
			e.replyError(w, http.StatusNotFound)
		}
		return
	}

	// CallAction method calls the defined action for the WebSocket.
	ctx.callAction()

	if e.onPostDisconnect != nil {
		e.onPostDisconnect(EventOnPostDisconnect, ctx)
	}
}

// Log method provides logging methods at WebSocket engine.
func (e *Engine) Log() log.Loggerer {
	return e.app.Log()
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Engine Unexported methods
//______________________________________________________________________________

func (e *Engine) connect(w http.ResponseWriter, r *http.Request, route *router.Route, pathParams ahttp.PathParams) (*Context, error) {
	ctx := e.newContext(r, route, pathParams)

	// Route constraints validation
	if errs := valpar.ValidateValues(pathParams, route.Constraints); len(errs) > 0 {
		ctx.Log().Error("WS: Route constraints failed")
		ctx.reason = router.ErrRouteConstraintFailed
		e.publishOnErrorEvent(ctx)
		e.replyError(w, http.StatusBadRequest)
		return nil, router.ErrRouteConstraintFailed
	}

	// Check Origin
	if e.checkOrigin && !e.isSameOrigin(ctx) {
		ctx.Log().Error("WS: Origin mismatch")
		ctx.reason = ErrOriginMismatch
		e.publishOnErrorEvent(ctx)
		e.replyError(w, http.StatusBadRequest)
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
		e.replyError(w, http.StatusBadRequest)
		return nil, ErrParameterParseFailed
	}

	if e.onPreConnect != nil {
		e.onPreConnect(EventOnPreConnect, ctx)
		if ctx.abortCode != 0 {
			e.replyError(w, ctx.abortCode)
			return nil, ErrAbortRequest
		}
	}

	r.Method = ahttp.MethodGet // back to GET for upgrade
	conn, _, hs, err := gws.UpgradeHTTP(r, w, ctx.Header)
	if err != nil {
		ctx.Log().Errorf("WS: Unable establish a WebSocket connection for '%s'", ctx.Req.Path)
		ctx.reason = ErrConnectFailed
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

func (e *Engine) newContext(r *http.Request, route *router.Route, pathParams ahttp.PathParams) *Context {
	ctx := &Context{
		e:      e,
		Header: make(http.Header),
		route:  route,
		Req: &Request{
			Host:       ahttp.Host(r),
			Path:       r.URL.Path,
			Header:     r.Header,
			pathParams: pathParams,
			raw:        r,
		},
	}
	ctx.Req.ID = e.createID(ctx)
	return ctx
}

// ReplyError method writes HTTP error response.
func (e *Engine) replyError(w http.ResponseWriter, errCode int) {
	writeHTTPError(w, errCode, fmt.Sprintf("%d %s", errCode, http.StatusText(errCode)))
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

func (e *Engine) createID(ctx *Context) string {
	if e.idGenerator == nil {
		return ess.NewGUID()
	}
	return e.idGenerator(ctx)
}
