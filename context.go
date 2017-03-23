// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"errors"
	"reflect"

	"aahframework.org/ahttp.v0"
	"aahframework.org/router.v0"
)

var (
	ctxPtrType = reflect.TypeOf((*Context)(nil))

	errTargetNotFound = errors.New("target not found")
)

type (
	// Context type for aah framework, gets embedded in application controller.
	Context struct {
		// Req is HTTP request instance
		Req *ahttp.Request

		// Res is HTTP response writer. It is recommended to use
		// `Reply()` builder for composing response.
		Res ahttp.ResponseWriter

		controller string
		action     *MethodInfo
		target     interface{}
		domain     *router.Domain
		route      *router.Route
		reply      *Reply
		viewArgs   map[string]interface{}
		abort      bool
	}
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Context methods
//___________________________________

// Reply method gives you control and convenient way to write
// a response effectively.
func (ctx *Context) Reply() *Reply {
	return ctx.reply
}

// ViewArgs method returns aah framework and request related info that can be
// used in template or view rendering, etc.
func (ctx *Context) ViewArgs() map[string]interface{} {
	return ctx.viewArgs
}

// AddViewArg method adds given key and value into `viewArgs`. These view args
// values accessible on templates. Chained call is possible.
func (ctx *Context) AddViewArg(key string, value interface{}) *Context {
	ctx.viewArgs[key] = value
	return ctx
}

// ReverseURL method returns the URL for given route name and args.
// See `Domain.ReverseURL` for more information.
func (ctx *Context) ReverseURL(routeName string, args ...interface{}) string {
	return createReverseURL(ctx.Req.Host, routeName, nil, args...)
}

// ReverseURLm method returns the URL for given route name and key-value paris.
// See `Domain.ReverseURLm` for more information.
func (ctx *Context) ReverseURLm(routeName string, args map[string]interface{}) string {
	return createReverseURL(ctx.Req.Host, routeName, args)
}

// Msg method returns the i18n value for given key otherwise empty string returned.
func (ctx *Context) Msg(key string, args ...interface{}) string {
	return AppI18n().Lookup(ctx.Req.Locale, key, args...)
}

// Msgl method returns the i18n value for given local and key otherwise
// empty string returned.
func (ctx *Context) Msgl(locale *ahttp.Locale, key string, args ...interface{}) string {
	return AppI18n().Lookup(locale, key, args...)
}

// Abort method sets the abort to true. It means framework will not proceed with
// next middleware, next interceptor or action based on context it being used.
// Contexts: 1) If it's called in the middleware, then middleware chain stops;
// framework starts processing response. 2) If it's called in Before interceptor
// then Before<Action> interceptor, mapped <Action>, After<Action> interceptor and
// After interceptor will not execute; framework starts processing response.
// 3) If it's called in Mapped <Action> then After<Action> interceptor and
// After interceptor will not execute; framework starts processing response.
func (ctx *Context) Abort() {
	ctx.abort = true
}

// Reset method resets context instance for reuse.
func (ctx *Context) Reset() {
	ctx.Req = nil
	ctx.Res = nil
	ctx.target = nil
	ctx.domain = nil
	ctx.controller = ""
	ctx.action = nil
	ctx.reply = nil
	ctx.viewArgs = nil
	ctx.abort = false
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Context Unexported methods
//___________________________________

// setTarget method sets contoller, action, embedded context into
// controller.
func (ctx *Context) setTarget(route *router.Route) error {
	controller := cRegistry.Lookup(route)
	if controller == nil {
		return errTargetNotFound
	}

	ctx.controller = controller.Name()
	ctx.action = controller.FindMethod(route.Action)
	if ctx.action == nil {
		return errTargetNotFound
	}

	targetPtr := reflect.New(controller.Type)
	target := targetPtr.Elem()
	ctxv := reflect.ValueOf(ctx)
	for _, index := range controller.EmbeddedIndexes {
		target.FieldByIndex(index).Set(ctxv)
	}

	ctx.target = targetPtr.Interface()
	return nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

// findEmbeddedContext method does breadth-first search on struct anonymous
// field to find `aah.Context` index positions.
func findEmbeddedContext(controllerType reflect.Type) [][]int {
	var indexes [][]int
	type nodeType struct {
		val   reflect.Value
		index []int
	}

	queue := []nodeType{{reflect.New(controllerType), []int{}}}

	for len(queue) > 0 {
		var (
			node     = queue[0]
			elem     = node.val
			elemType = elem.Type()
		)

		if elemType.Kind() == reflect.Ptr {
			elem = elem.Elem()
			elemType = elem.Type()
		}

		queue = queue[1:]
		if elemType.Kind() != reflect.Struct {
			continue
		}

		for i := 0; i < elem.NumField(); i++ {
			// skip non-anonymous fields
			field := elemType.Field(i)
			if !field.Anonymous {
				continue
			}

			// If it's a `aah.Context`, record the field indexes
			if field.Type == ctxPtrType {
				indexes = append(indexes, append(node.index, i))
				continue
			}

			fieldValue := elem.Field(i)
			queue = append(queue,
				nodeType{fieldValue, append(append([]int{}, node.index...), i)})
		}
	}

	return indexes
}
