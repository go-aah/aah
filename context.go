// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org/ws source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ws

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"net"
	"net/http"
	"net/url"
	"reflect"

	"aahframework.org/ainsp.v0"
	"aahframework.org/log.v0"
	"aahframework.org/router.v0"
	"aahframework.org/valpar.v0"

	gws "github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

var (
	ctxPtrType = reflect.TypeOf((*Context)(nil))
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Context struct and its methods
//______________________________________________________________________________

// Context struct holds friendly WebSocket implementation for aah framework.
type Context struct {
	Req    *Request
	Conn   net.Conn
	Header http.Header // These headers are sent to WS client during Connection upgrade

	e          *Engine
	hs         gws.Handshake
	route      *router.Route
	websocket  *ainsp.Target
	action     *ainsp.Method
	target     interface{}
	targetrv   reflect.Value
	actionrv   reflect.Value
	actionArgs []reflect.Value
	logger     log.Loggerer
	reason     error
	abortCode  int
}

// ReadText method reads a text value from WebSocket client.
//
// Note: Method does HTML sanatize internally. Refer to `html.EscapeString`.
func (ctx *Context) ReadText() (string, error) {
	data, err := wsutil.ReadClientText(ctx.Conn)
	if err != nil {
		return "", createError(err)
	}
	return html.EscapeString(string(data)), nil
}

// ReadBinary method reads a binary data from WebSocket client.
func (ctx *Context) ReadBinary() ([]byte, error) {
	data, err := wsutil.ReadClientBinary(ctx.Conn)
	if err != nil {
		return nil, createError(err)
	}
	return data, nil
}

// ReadJSON method reads JSON data from WebSocket client and does unmarshal
// into given object.
func (ctx *Context) ReadJSON(t interface{}) error {
	data, err := wsutil.ReadClientText(ctx.Conn)
	if err != nil {
		return createError(err)
	}
	return json.Unmarshal(data, t)
}

// ReadXML method reads XML data from WebSocket client and does unmarshal
// into given object.
func (ctx *Context) ReadXML(t interface{}) error {
	data, err := wsutil.ReadClientText(ctx.Conn)
	if err != nil {
		return createError(err)
	}
	return xml.Unmarshal(data, t)
}

// ReplyText method sends Text data to the WebSocket client returns error
// if client is gone, network error, etc.
func (ctx *Context) ReplyText(v string) error {
	return createError(wsutil.WriteServerMessage(ctx.Conn, gws.OpText, []byte(v)))
}

// ReplyBinary method sends Binary data to the WebSocket client returns
// error if client is gone, network error, etc.
func (ctx *Context) ReplyBinary(v []byte) error {
	return createError(wsutil.WriteServerMessage(ctx.Conn, gws.OpBinary, v))
}

// ReplyJSON method sends JSON data to the WebSocket client returns
// error if json marshal issue, client is gone, network issue, etc.
func (ctx *Context) ReplyJSON(v interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return createError(wsutil.WriteServerMessage(ctx.Conn, gws.OpText, b))
}

// ReplyXML method sends XML data to the WebSocket client returns
// error if XML marshal issue, client is gone, network issue, etc.
func (ctx *Context) ReplyXML(v interface{}) error {
	b, err := xml.Marshal(v)
	if err != nil {
		return err
	}
	return createError(wsutil.WriteServerMessage(ctx.Conn, gws.OpText, b))
}

// Disconnect method disconnects the WebSocket connection immediately. Could be
// used for force disconnect client from server-side.
//
// Note: After this call, any read/reply will result in error. Since connection
// already closed from server side.
func (ctx *Context) Disconnect() error {
	return ctx.Conn.Close()
}

// Log method adds field WebSocket `Request ID` into current log context and
// returns the logger.
func (ctx *Context) Log() log.Loggerer {
	if ctx.logger == nil {
		ctx.logger = ctx.e.Log().WithFields(log.Fields{
			"reqid": ctx.Req.ID,
		})
	}
	return ctx.logger
}

// ErrorReason method returns error info if error was occurred otherwise nil.
func (ctx *Context) ErrorReason() error {
	return ctx.reason
}

// Abort method is useful for `OnPreConnect` event, aah user could make a choice
// of proceed or abort.
//
// For e.g.:
// 	ctx.Abort(http.StatusUnauthorized)
// 	ctx.Abort(http.StatusForbidden)
func (ctx *Context) Abort(httpErroCode int) {
	ctx.abortCode = httpErroCode
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Context struct Unexported methods
//______________________________________________________________________________

// CallAction method calls the defined action for the WebSocket.
func (ctx *Context) callAction() {
	ctx.Log().Debugf("Calling websocket: %s.%s", ctx.websocket.FqName, ctx.action.Name)
	if ctx.actionrv.Type().IsVariadic() {
		ctx.actionrv.CallSlice(ctx.actionArgs)
	} else {
		ctx.actionrv.Call(ctx.actionArgs)
	}
}

func (ctx *Context) setTarget(targetName, methodName string) error {
	if ctx.websocket = ctx.e.registry.Lookup(targetName); ctx.websocket == nil {
		return ErrNotFound
	}

	if ctx.action = ctx.websocket.Lookup(methodName); ctx.action == nil {
		return ErrNotFound
	}

	target := reflect.New(ctx.websocket.Type)

	// check method exists or not
	ctx.actionrv = reflect.ValueOf(target.Interface()).MethodByName(ctx.action.Name)
	if !ctx.actionrv.IsValid() {
		return ErrNotFound
	}

	targetElem := target.Elem()
	ctxrv := reflect.ValueOf(ctx)
	for _, index := range ctx.websocket.EmbeddedIndexes {
		targetElem.FieldByIndex(index).Set(ctxrv)
	}

	ctx.target = target.Interface()
	ctx.targetrv = reflect.ValueOf(ctx.target)
	return nil
}

var emptyArg = make([]reflect.Value, 0)

func (ctx *Context) parseParameters() error {
	paramCnt := len(ctx.action.Parameters)
	if paramCnt == 0 {
		ctx.actionArgs = emptyArg
		return nil
	}

	params := make(url.Values)
	for k, v := range ctx.Req.pathParams {
		params.Set(k, v)
	}
	for k, v := range ctx.Req.queryParams {
		params[k] = v
	}

	// Parse and Bind parameters
	var err error
	ctx.actionArgs = make([]reflect.Value, paramCnt)
	for idx, val := range ctx.action.Parameters {
		var result reflect.Value
		if vpFn, found := valpar.ValueParser(val.Type); found {
			result, err = vpFn(val.Name, val.Type, params)
			if rule, found := ctx.route.ValidationRule(val.Name); found {
				if !valpar.ValidateValue(result.Interface(), rule) {
					return fmt.Errorf("Path param validation failed [name: %s, rule: %s, value: %v]",
						val.Name, rule, result.Interface())
				}
			}
		} else if val.Kind == reflect.Struct {
			result, err = valpar.Struct("", val.Type, params)
		}

		// check error
		if err != nil {
			if !result.IsValid() {
				ctx.Log().Errorf("Parsed parameter value is invalid or value parser not found [param: %s, type: %s]",
					val.Name, val.Type)
			}
			return err
		}

		// Apply Validation for type `struct`
		if val.Kind == reflect.Struct {
			if errs, _ := valpar.Validate(result.Interface()); errs != nil {
				ctx.Log().Errorf("Param validation failed [name: %s, type: %s], Validation Errors:\n%v",
					val.Name, val.Type, errs.Error())
				return errs
			}
		}

		// set action parameter value
		ctx.actionArgs[idx] = result
	}

	return nil
}
