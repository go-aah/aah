// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"errors"
	"net/url"
	"reflect"
	"strings"

	"aahframework.org/ahttp.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
	"aahframework.org/router.v0"
	"aahframework.org/security.v0"
	"aahframework.org/security.v0/session"
)

var (
	ctxPtrType = reflect.TypeOf((*Context)(nil))

	errTargetNotFound = errors.New("target not found")
)

type (
	// Context type for aah framework, gets embedded in application controller.
	//
	// Note: this is not standard package `context.Context`.
	Context struct {
		// Req is HTTP request instance
		Req *ahttp.Request

		// Res is HTTP response writer compliant. It is highly recommended to use
		// `Reply()` builder for composing response.
		//
		// Note: If you're using `cxt.Res` directly, don't forget to call
		// `Reply().Done()` so that framework will not intervene with your
		// response.
		Res ahttp.ResponseWriter

		controller *controllerInfo
		action     *MethodInfo
		target     interface{}
		domain     *router.Domain
		route      *router.Route
		subject    *security.Subject
		reply      *Reply
		viewArgs   map[string]interface{}
		values     map[string]interface{}
		abort      bool
		decorated  bool
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

// Subdomain method returns the subdomain from the incoming request if available
// as per routes.conf. Otherwise empty string.
func (ctx *Context) Subdomain() string {
	if ctx.domain.IsSubDomain {
		if idx := strings.IndexByte(ctx.Req.Host, '.'); idx > 0 {
			return ctx.Req.Host[:idx]
		}
	}
	return ""
}

// Subject method the subject (aka application user) of current request.
func (ctx *Context) Subject() *security.Subject {
	return ctx.subject
}

// Session method always returns `session.Session` object. Use `Session.IsNew`
// to identify whether sesison is newly created or restored from the request
// which was already created.
func (ctx *Context) Session() *session.Session {
	if ctx.subject.Session == nil {
		ctx.subject.Session = AppSessionManager().NewSession()
	}
	return ctx.subject.Session
}

// Abort method sets the abort to true. It means framework will not proceed with
// next middleware, next interceptor or action based on context it being used.
// Contexts:
//    1) If it's called in the middleware, then middleware chain stops;
// framework starts processing response.
//    2) If it's called in Before interceptor then Before<Action> interceptor,
// mapped <Action>, After<Action> interceptor and After interceptor will not
// execute; framework starts processing response.
//    3) If it's called in Mapped <Action> then After<Action> interceptor and
// After interceptor will not execute; framework starts processing response.
func (ctx *Context) Abort() {
	ctx.abort = true
}

// IsStaticRoute method returns true if it's static route otherwise false.
func (ctx *Context) IsStaticRoute() bool {
	if ctx.route != nil {
		return ctx.route.IsStatic
	}
	return false
}

// SetURL method is to set the request URL to change the behaviour of request
// routing. Ideal for URL rewrting. URL can be relative or absolute URL.
//
// Note: This method only takes effect on `OnRequest` server event.
func (ctx *Context) SetURL(pathURL string) {
	if !ctx.decorated {
		return
	}

	u, err := url.Parse(pathURL)
	if err != nil {
		log.Errorf("invalid URL provided: %s", err)
		return
	}

	rawReq := ctx.Req.Unwrap()
	if !ess.IsStrEmpty(u.Host) {
		log.Debugf("Host have been updated from '%s' to '%s'", ctx.Req.Host, u.Host)
		rawReq.Host = u.Host
		rawReq.URL.Host = u.Host
	}

	log.Debugf("URL path have been updated from '%s' to '%s'", ctx.Req.Path, u.Path)
	rawReq.URL.Path = u.Path

	// Update the context
	ctx.Req.Host = rawReq.Host
	ctx.Req.Path = rawReq.URL.Path
}

// SetMethod method is to set the request `Method` to change the behaviour
// of request routing. Ideal for URL rewrting.
//
// Note: This method only takes effect on `OnRequest` server event.
func (ctx *Context) SetMethod(method string) {
	if !ctx.decorated {
		return
	}

	method = strings.ToUpper(method)
	if _, found := router.HTTPMethodActionMap[method]; !found {
		log.Errorf("given method '%s' is not valid", method)
		return
	}

	log.Debugf("Request method have been updated from '%s' to '%s'", ctx.Req.Method, method)
	ctx.Req.Unwrap().Method = method
	ctx.Req.Method = method
}

// Reset method resets context instance for reuse.
func (ctx *Context) Reset() {
	ctx.Req = nil
	ctx.Res = nil
	ctx.controller = nil
	ctx.action = nil
	ctx.target = nil
	ctx.domain = nil
	ctx.route = nil
	ctx.subject = nil
	ctx.reply = nil
	ctx.viewArgs = make(map[string]interface{})
	ctx.values = make(map[string]interface{})
	ctx.abort = false
	ctx.decorated = false
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

	ctx.controller = controller
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
