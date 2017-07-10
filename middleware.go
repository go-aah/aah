// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"net/http"
	"reflect"

	"aahframework.org/log.v0-unstable"
)

var (
	mwStack []MiddlewareFunc
	mwChain []*Middleware
)

type (
	// MiddlewareFunc func type is aah framework middleware signature.
	MiddlewareFunc func(ctx *Context, m *Middleware)

	// Middleware struct is to implement aah framework middleware chain.
	Middleware struct {
		next    MiddlewareFunc
		further *Middleware
	}
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package methods
//___________________________________

// Middlewares method adds given middleware into middleware stack
func Middlewares(middlewares ...MiddlewareFunc) {
	mwStack = mwStack[:len(mwStack)-2]

	for _, m := range middlewares {
		if m != nil {
			mwStack = append(mwStack, m)
		}
	}

	mwStack = append(
		mwStack,
		interceptorMiddleware,
		actionMiddleware,
	)

	invalidateMwChain()
}

// ToMiddleware method expands the possibilities. It helps Golang community to
// convert the third-party or your own net/http middleware into `aah.MiddlewareFunc`
//
// You can register below handler types:
// 1) aah.ToMiddleware(h http.Handler)
// 2) aah.ToMiddleware(h http.HandlerFunc)
// 3) aah.ToMiddleware(func(w http.ResponseWriter, r *http.Request))
func ToMiddleware(handler interface{}) MiddlewareFunc {
	switch handler.(type) {
	case MiddlewareFunc:
		return handler.(MiddlewareFunc)
	case http.Handler:
		h := handler.(http.Handler)
		return func(ctx *Context, m *Middleware) {
			h.ServeHTTP(ctx.Res, ctx.Req.Raw)
			m.Next(ctx)
		}
	case func(http.ResponseWriter, *http.Request):
		return ToMiddleware(http.HandlerFunc(handler.(func(http.ResponseWriter, *http.Request))))
	default:
		log.Errorf("Not a vaild handler: %s", funcName(handler))
		return nil
	}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Middleware methods
//___________________________________

// Next method calls next middleware in the chain if available.
func (mw *Middleware) Next(ctx *Context) {
	if ctx.abort {
		// abort, not to proceed further
		return
	}

	if mw.next != nil {
		mw.next(ctx, mw.further)
	}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Interceptor middleware
//___________________________________

// interceptorMiddleware calls pre-defined actions (Before, Before<ActionName>,
// After, After<ActionName>, Panic, Panic<ActionName>, Finally,
// Finally<ActionName>) from controller.
func interceptorMiddleware(ctx *Context, m *Middleware) {
	target := reflect.ValueOf(ctx.target)
	controller := resolveControllerName(ctx)

	// Finally action and method
	defer func() {
		if ctx.abort {
			return
		}

		if finallyActionMethod := target.MethodByName(incpFinallyActionName + ctx.action.Name); finallyActionMethod.IsValid() {
			log.Debugf("Calling interceptor: %s.%s", controller, incpFinallyActionName+ctx.action.Name)
			finallyActionMethod.Call(emptyArg)
		}

		if finallyAction := target.MethodByName(incpFinallyActionName); finallyAction.IsValid() {
			log.Debugf("Calling interceptor: %s.%s", controller, incpFinallyActionName)
			finallyAction.Call(emptyArg)
		}
	}()

	// Panic action and method
	defer func() {
		if r := recover(); r != nil {
			if ctx.abort {
				return
			}

			if panicActionMethod := target.MethodByName(incpPanicActionName + ctx.action.Name); panicActionMethod.IsValid() {
				log.Debugf("Calling interceptor: %s.%s", controller, incpPanicActionName+ctx.action.Name)
				rv := append([]reflect.Value{}, reflect.ValueOf(r))
				panicActionMethod.Call(rv)
			} else if panicAction := target.MethodByName(incpPanicActionName); panicAction.IsValid() {
				log.Debugf("Calling interceptor: %s.%s", controller, incpPanicActionName)
				rv := append([]reflect.Value{}, reflect.ValueOf(r))
				panicAction.Call(rv)
			} else { // propagate it
				panic(r)
			}
		}
	}()

	// Before action
	if beforeAction := target.MethodByName(incpBeforeActionName); beforeAction.IsValid() {
		log.Debugf("Calling interceptor: %s.%s", controller, incpBeforeActionName)
		beforeAction.Call(emptyArg)
	}

	// Before action method
	if !ctx.abort {
		if beforeActionMethod := target.MethodByName(incpBeforeActionName + ctx.action.Name); beforeActionMethod.IsValid() {
			log.Debugf("Calling interceptor: %s.%s", controller, incpBeforeActionName+ctx.action.Name)
			beforeActionMethod.Call(emptyArg)
		}
	}

	m.Next(ctx)

	// After action method
	if !ctx.abort {
		if afterActionMethod := target.MethodByName(incpAfterActionName + ctx.action.Name); afterActionMethod.IsValid() {
			log.Debugf("Calling interceptor: %s.%s", controller, incpAfterActionName+ctx.action.Name)
			afterActionMethod.Call(emptyArg)
		}
	}

	// After action
	if !ctx.abort {
		if afterAction := target.MethodByName(incpAfterActionName); afterAction.IsValid() {
			log.Debugf("Calling interceptor: %s.%s", controller, incpAfterActionName)
			afterAction.Call(emptyArg)
		}
	}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Action middleware
//___________________________________

// ActionMiddleware calls the requested action on controller
func actionMiddleware(ctx *Context, m *Middleware) {
	target := reflect.ValueOf(ctx.target)
	action := target.MethodByName(ctx.action.Name)

	if !action.IsValid() {
		return
	}

	actionArgs := make([]reflect.Value, len(ctx.action.Parameters))

	// TODO Auto Binder for arguments

	log.Debugf("Calling controller: %s.%s", resolveControllerName(ctx), ctx.action.Name)
	if action.Type().IsVariadic() {
		action.CallSlice(actionArgs)
	} else {
		action.Call(actionArgs)
	}

}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

func invalidateMwChain() {
	mwChain = nil
	cnt := len(mwStack)
	mwChain = make([]*Middleware, cnt)

	for idx := 0; idx < cnt; idx++ {
		mwChain[idx] = &Middleware{next: mwStack[idx]}
	}

	for idx := cnt - 1; idx > 0; idx-- {
		mwChain[idx-1].further = mwChain[idx]
	}

	mwChain[cnt-1].further = &Middleware{}
}

func init() {
	mwStack = append(mwStack,
		interceptorMiddleware,
		actionMiddleware,
	)

	invalidateMwChain()
}
